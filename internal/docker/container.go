package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/logger"
	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	anyIP       = "0.0.0.0"                      // Refers to any/all IPv4 addresses
	defaultPort = 19132                          // Default port for player connections
	protocol    = "UDP"                          // MC uses UDP
	imageName   = "danhaledocker/craftmine:v1.7" // The name of the docker image to use
	stopTimeout = 10
)

// Container is a docker client which operates on an existing container.
type Container struct {
	client.ContainerAPIClient
	ContainerName, containerID string
}

// GetContainerOrExit is a convenience function for attempting to find an existing docker container with the given name.
// If not found, a helpful error message is printed and the program exits without error.
func GetContainerOrExit(containerName string) *Container {
	d, err := GetContainer(containerName)

	if err != nil {
		// Container was not found
		if _, ok := err.(*ContainerNotFoundError); ok {
			logger.Info.Println(err)
			os.Exit(0)
		} else if _, ok := err.(*NotACraftContainerError); ok {
			logger.Info.Println(err)
			os.Exit(0)
		}

		// Something else went wrong
		panic(err)
	}

	return d
}

// GetContainer returns a new default docker container API client for an existing container. If the given container name
// doesn't exist an error of type ContainerNotFoundError is returned.
func GetContainer(containerName string) (*Container, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	id, err := ContainerID(containerName, c)
	if err != nil {
		return nil, err
	}

	d := Container{
		ContainerAPIClient: c,
		ContainerName:      containerName,
		containerID:        id,
	}

	ok, err := d.IsCraftServer()
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, &NotACraftContainerError{Name: containerName}
	}

	return &d, nil
}

// IsCraftServer returns true if the container has a label with a key matching craftLabel. Craft containers are given
// this label on creation.
func (d *Container) IsCraftServer() (bool, error) {
	containerJSON, err := d.ContainerInspect(context.Background(), d.containerID)
	if err != nil {
		return false, fmt.Errorf("inspecting container: %s", err)
	}

	_, ok := containerJSON.Config.Labels[craftLabel]

	return ok, nil
}

// CopyFileFrom copies the file at the given path, extracts the file body from the tar archive and returns it's bytes.
func (d *Container) CopyFileFrom(containerPath string) ([]byte, error) {
	tr, err := d.CopyFrom(containerPath)
	if err != nil {
		return nil, err
	}

	_, err = tr.Next()
	if err == io.EOF {
		return nil, fmt.Errorf("no file was found at '%s', got EOF reading tar archive", containerPath)
	}

	if err != nil {
		return nil, fmt.Errorf("reading tar archive: %s", err)
	}

	b, err := ioutil.ReadAll(tr)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// CopyFrom returns a tar archive containing the file(s) at the given container path.
func (d *Container) CopyFrom(containerPath string) (*tar.Reader, error) {
	data, _, err := d.CopyFromContainer(
		context.Background(),
		d.containerID,
		containerPath,
	)
	if err != nil {
		return nil, fmt.Errorf("copying data from server at '%s': %s", containerPath, err)
	}

	return tar.NewReader(data), nil
}

// CopyFileTo archives the given bytes as a tar containing one file and copies that file to the destination path.
func (d *Container) CopyFileTo(containerPath string, body []byte) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name: filepath.Base(containerPath),
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing header: %s", err)
	}

	if _, err := tw.Write(body); err != nil {
		return fmt.Errorf("writing body: %s", err)
	}

	return d.CopyTo(filepath.Dir(containerPath), &buf)
}

// CopyTo copies the given tar data to the destination path in the container.
func (d *Container) CopyTo(destPath string, tar *bytes.Buffer) error {
	err := d.CopyToContainer(
		context.Background(),
		d.containerID,
		destPath,
		tar,
		docker.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("copying files to '%s': %s", destPath, err)
	}

	return nil
}

// Command runs the given arguments separated by spaces as a command in the bedrock_server process cli.
func (d *Container) Command(args []string) error {
	// Attach to the container
	waiter, err := d.ContainerAttach(
		context.Background(),
		d.containerID,
		docker.ContainerAttachOptions{
			Stdin:  true,
			Stream: true,
		},
	)

	if err != nil {
		return err
	}

	commandString := strings.Join(args, " ") + "\n"

	// Write the command to the bedrock_server process cli
	_, err = waiter.Conn.Write([]byte(
		commandString,
	))
	if err != nil {
		return err
	}

	return nil
}

// CommandWriter returns a *net.Conn which streams to the mc server process stdin.
func (d *Container) CommandWriter() (net.Conn, error) {
	// Attach to the container
	waiter, err := d.ContainerAttach(
		context.Background(),
		d.containerID,
		docker.ContainerAttachOptions{
			Stdin:  true,
			Stream: true,
		},
	)
	if err != nil {
		return nil, err
	}

	return waiter.Conn, err
}

// Stop stops the docker container.
func (d *Container) Stop() error {
	timeout := time.Duration(stopTimeout)

	return d.ContainerStop(
		context.Background(),
		d.containerID,
		&timeout,
	)
}

// LogReader returns a buffer with the stdout and stderr from the running mc server process. New output will continually
// be sent to the buffer. A negative tail value will result in the 'all' value being used.
func (d *Container) LogReader(tail int) (*bufio.Reader, error) {
	logs, err := d.ContainerLogs(
		context.Background(),
		d.containerID,
		docker.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       strconv.Itoa(tail),
			Follow:     true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("getting docker container logs: %s", err)
	}

	return bufio.NewReader(logs), nil
}

// GetPort returns the port players use to connect to this server
func (d *Container) GetPort() (int, error) {
	c, err := d.ContainerInspect(context.Background(), d.containerID)
	if err != nil {
		return 0, err
	}

	portBindings := c.HostConfig.PortBindings

	if len(portBindings) == 0 {
		return 0, fmt.Errorf("no ports bound for container %s", d.ContainerName)
	}

	var port int

	for _, v := range portBindings {
		p, err := strconv.Atoi(v[0].HostPort)
		if err != nil {
			return 0, fmt.Errorf("error reading container port: %s", err)
		}

		port = p
	}

	if port == 0 {
		panic("port is 0")
	}

	return port, nil
}

func (d *Container) Stat(path string) (docker.ContainerPathStat, error) {
	return d.ContainerStatPath(context.Background(), d.containerID, path)
}

// ContainerNotFoundError tells the caller that no containers were found with the given name.
type ContainerNotFoundError struct {
	Name string
}

func (e *ContainerNotFoundError) Error() string {
	return fmt.Sprintf("container with name '%s' not found.", e.Name)
}

// NotACraftContainerError reports the instance where a container is found with a given name but lacks the label
// indicating that it is managed using craft.
type NotACraftContainerError struct {
	Name string
}

func (e *NotACraftContainerError) Error() string {
	return fmt.Sprintf("container found with name '%s' but it does not appear to be a craft server.", e.Name)
}

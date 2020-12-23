package craft

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/danhale-git/craft/internal/files"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	client.ContainerAPIClient
	containerName, containerID string
}

// NewDockerClientOrExit is a convenience function for attempting to find a docker client with the given name. If not
// found, a helpful error message is printed and the program exits without error.
func NewDockerClientOrExit(containerName string) *DockerClient {
	d, err := NewDockerClient(containerName)

	if err != nil {
		// Container was not found
		if _, ok := err.(*ContainerNotFoundError); ok {
			fmt.Printf("Error: server with name '%s' does not exist\n", containerName)
			os.Exit(0)
		}

		// Something else went wrong
		panic(err)
	}

	return d
}

// NewDockerClient returnsa new default Docker Container API client. If the given container name doesn't exist, an error
// of type ContainerNotFoundError is returned.
func NewDockerClient(containerName string) (*DockerClient, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	id, err := ContainerID(containerName, c)
	if err != nil {
		return nil, err
	}

	d := DockerClient{
		ContainerAPIClient: c,
		containerName:      containerName,
		containerID:        id,
	}

	return &d, nil
}

// ContainerFromName returns the ID of the container with the given name or an error if that container doesn't exist.
func ContainerID(name string, client client.ContainerAPIClient) (string, error) {
	containers, err := client.ContainerList(context.Background(), docker.ContainerListOptions{})
	if err != nil {
		return "", fmt.Errorf("listing all containers: %s", err)
	}

	for _, ctr := range containers {
		if strings.Trim(ctr.Names[0], "/") == name {
			return ctr.ID, nil
		}
	}

	return "", &ContainerNotFoundError{Name: name}
}

// Command runs the given arguments separated by spaces as a command in the bedrock_server process cli.
func (d *DockerClient) Command(args []string) error {
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

// LogReader returns a buffer with the stdout and stderr from the running mc server process. New output will continually
// be sent to the buffer.
func (d *DockerClient) LogReader(tail int) (*bufio.Reader, error) {
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

func (d *DockerClient) copyFrom(containerPath string) (*files.Archive, error) {
	data, _, err := d.CopyFromContainer(
		context.Background(),
		d.containerID,
		containerPath,
	)
	if err != nil {
		return nil, fmt.Errorf("copying data from server at '%s': %s", containerPath, err)
	}

	archive, err := files.NewArchiveFromTar(data)
	if err != nil {
		return nil, fmt.Errorf("reading tar data from '%s' to file archive: %s", containerPath, err)
	}

	return archive, nil
}

func (d *DockerClient) copyTo(destPath string, files *files.Archive) error {
	t, err := files.Tar()
	if err != nil {
		return fmt.Errorf("creating tar archive: %s", err)
	}

	err = d.CopyToContainer(
		context.Background(),
		d.containerID,
		destPath,
		t,
		docker.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("copying files to '%s': %s", destPath, err)
	}

	return nil
}

// ContainerNotFoundError tells the caller that no containers were found with the given name.
type ContainerNotFoundError struct {
	Name string
}

func (e *ContainerNotFoundError) Error() string {
	return fmt.Sprintf("container with name '%s' not found.", e.Name)
}

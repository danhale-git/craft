package dockerwrapper

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/danhale-git/craft/internal/logger"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const CraftLabel = "danhale-git/craft"

// Server is a wrapper for docker's client.ContainerAPIClient which operates on a specific container.
type Server struct {
	client.ContainerAPIClient
	ContainerName, ContainerID string
}

// New returns a Server struct representing an existing server. If the given name doesn't exist an error of type
// ContainerNotFoundError is returned.
func New(containerName string) (*Server, error) {
	cl, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	id, err := containerID(containerName, cl)
	if err != nil {
		return nil, err
	}

	c := Server{
		ContainerAPIClient: cl,
		ContainerName:      containerName,
		ContainerID:        id,
	}

	containerJSON, err := cl.ContainerInspect(context.Background(), c.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("inspecting container: %s", err)
	}

	_, ok := containerJSON.Config.Labels[CraftLabel]

	if !ok {
		return nil, &NotACraftContainerError{Name: containerName}
	}

	return &c, nil
}

// Command attaches to the container and runs the given arguments separated by spaces.
func (c *Server) Command(args []string) error {
	conn, err := c.CommandWriter()
	if err != nil {
		return err
	}

	commandString := strings.Join(args, " ") + "\n"

	_, err = conn.Write([]byte(commandString))
	if err != nil {
		return err
	}

	return nil
}

// CommandWriter returns a *net.Conn which streams to the container process stdin.
func (c *Server) CommandWriter() (net.Conn, error) {
	waiter, err := c.ContainerAttach(
		context.Background(),
		c.ContainerID,
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

// LogReader returns a buffer with the stdout and stderr from the running mc server process. New output will continually
// be sent to the buffer. A negative tail value will result in the 'all' value being used.
func (c *Server) LogReader(tail int) (*bufio.Reader, error) {
	logs, err := c.ContainerLogs(
		context.Background(),
		c.ContainerID,
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
func (c *Server) GetPort() (int, error) {
	cj, err := c.ContainerInspect(context.Background(), c.ContainerID)
	if err != nil {
		return 0, err
	}

	portBindings := cj.HostConfig.PortBindings

	if len(portBindings) == 0 {
		return 0, fmt.Errorf("no ports bound for container %s", c.ContainerName)
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

func containerID(name string, client client.ContainerAPIClient) (string, error) {
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

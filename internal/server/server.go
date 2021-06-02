package server

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/danhale-git/craft/internal/logger"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const CraftLabel = "danhale-git/craft"

// Server is a wrapper for docker's client.ContainerAPIClient which operates on a specific container.
type Server struct {
	client.ContainerAPIClient
	ContainerName, ContainerID string
}

// New creates a new craft server container and returns a docker client for it.
// It is the equivalent of the following docker command:
//
//    docker run -d -e EULA=TRUE -p <HOST_PORT>:19132/udp <imageName>
func New(hostPort int, name string) (*Server, error) {
	if hostPort == 0 {
		hostPort = nextAvailablePort()
	}

	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	ctx := context.Background()

	hostBinding := nat.PortBinding{
		HostIP:   anyIP,
		HostPort: strconv.Itoa(hostPort),
	}

	// -p <HOST_PORT>:19132/udp
	containerPort, err := nat.NewPort(protocol, strconv.Itoa(defaultPort))
	if err != nil {
		return nil, fmt.Errorf("creating container port: %s", err)
	}

	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	// docker run -d -e EULA=TRUE
	createResp, err := c.ContainerCreate(
		ctx,
		&container.Config{
			Image:        imageName,
			Env:          []string{"EULA=TRUE"},
			ExposedPorts: nat.PortSet{containerPort: struct{}{}},
			AttachStdin:  true, AttachStdout: true, AttachStderr: true,
			Tty:       true,
			OpenStdin: true,
			Labels:    map[string]string{CraftLabel: ""},
		},
		&container.HostConfig{
			PortBindings: portBinding,
			AutoRemove:   true,
		},
		nil, nil, name,
	)
	if err != nil {
		return nil, fmt.Errorf("creating docker container: %s", err)
	}

	err = c.ContainerStart(ctx, createResp.ID, docker.ContainerStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("starting container: %s", err)
	}

	s := Server{
		ContainerAPIClient: c,
		ContainerName:      name,
		ContainerID:        createResp.ID,
	}

	return &s, nil
}

// Get returns a Server struct representing a server which was already running. If the given name doesn't exist an error
// of type NotFoundError is returned.
func Get(cl client.ContainerAPIClient, containerName string) (*Server, error) {
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
		return nil, &NotCraftError{Name: containerName}
	}

	return &c, nil
}

// Command attaches to the container and runs the given arguments separated by spaces.
func (s *Server) Command(args []string) error {
	conn, err := s.CommandWriter()
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
func (s *Server) CommandWriter() (net.Conn, error) {
	waiter, err := s.ContainerAttach(
		context.Background(),
		s.ContainerID,
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
func (s *Server) LogReader(tail int) (*bufio.Reader, error) {
	logs, err := s.ContainerLogs(
		context.Background(),
		s.ContainerID,
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

// Port returns the port players use to connect to this server.
func (s *Server) Port() (int, error) {
	cj, err := s.ContainerInspect(context.Background(), s.ContainerID)
	if err != nil {
		return 0, err
	}

	portBindings := cj.HostConfig.PortBindings

	if len(portBindings) == 0 {
		return 0, fmt.Errorf("no ports bound for container %s", s.ContainerName)
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

// All returns a client for each active server.
func All(c client.ContainerAPIClient) ([]*Server, error) {
	containers, err := c.ContainerList(
		context.Background(),
		docker.ContainerListOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("listing docker containers: %s", err)
	}

	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = strings.Replace(c.Names[0], "/", "", 1)
	}

	servers := make([]*Server, 0)

	for _, n := range names {
		s, err := Get(c, n)
		if err != nil {
			if _, ok := err.(*NotCraftError); ok {
				continue
			}

			return nil, fmt.Errorf("creating client for container '%s': %s", n, err)
		}

		servers = append(servers, s)
	}

	return servers, nil
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

	return "", &NotFoundError{Name: name}
}

// NotFoundError tells the caller that no containers were found with the given name.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("container with name '%s' not found.", e.Name)
}

// NotCraftError reports the instance where a container is found with a given name but lacks the label
// indicating that it is managed using craft.
type NotCraftError struct {
	Name string
}

func (e *NotCraftError) Error() string {
	return fmt.Sprintf("container found with name '%s' but it does not appear to be a craft server.", e.Name)
}

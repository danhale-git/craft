package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/volume"

	"github.com/danhale-git/craft/internal/files"

	"github.com/docker/docker/api/types/mount"

	"github.com/danhale-git/craft/internal/logger"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	CraftLabel   = "danhale-git/craft" // Label used to identify craft servers
	volumeLabel  = "danhale-git_craft"
	anyIP        = "0.0.0.0"                        // Refers to any/all IPv4 addresses
	defaultPort  = 19132                            // Default port for player connections
	protocol     = "UDP"                            // MC uses UDP
	ImageName    = "craft_bedrock_server:autobuild" // The name of the docker image to use
	stopTimeout  = 30
	RunMCCommand = "cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server"
)

func dockerClient() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
}

// Server is a wrapper for docker's client.ContainerAPIClient which operates on a specific container.
type Server struct {
	client.ContainerAPIClient
	ContainerName, ContainerID string
}

// New creates a new craft server container and returns a docker client for it.
// It is the equivalent of the following docker command:
//
//    docker run -d -e EULA=TRUE -p <HOST_PORT>:19132/udp <imageName>
//
// If mountVolume is true, a local volume will also be mounted and autoremove will be disabled.
func New(hostPort int, name string, mountVolume bool) (*Server, error) {
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

	var mounts []mount.Mount
	if mountVolume {
		volName := fmt.Sprintf("%s-%s", volumeLabel, name)
		vol, err := c.VolumeCreate(ctx, volume.VolumeCreateBody{
			Name: volName,
		})
		if err != nil {
			return nil, fmt.Errorf("creating vol '%s': %w", volName, err)
		}

		mounts = []mount.Mount{{
			Type:   mount.TypeVolume,
			Source: vol.Name,
			Target: files.Directory,
		}}
	}

	// docker run -d -e EULA=TRUE
	createResp, err := c.ContainerCreate(
		ctx,
		&container.Config{
			Image:        ImageName,
			Env:          []string{"EULA=TRUE"},
			ExposedPorts: nat.PortSet{containerPort: struct{}{}},
			AttachStdin:  true, AttachStdout: true, AttachStderr: true,
			Tty:       true,
			OpenStdin: true,
			Labels:    map[string]string{CraftLabel: ""},
		},
		&container.HostConfig{
			PortBindings: portBinding,
			AutoRemove:   !mountVolume,
			Mounts:       mounts,
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

// Get searches for a server with the given name (stopped or running) and checks that it has a label identifying it as
// a craft server. If no craft server with that name exists, an error of type NotFoundError. If the server is found but
// is not a craft server, an error of type NotCraftError is returned.
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

// Stop executes a stop command first in the server process cli then on the container itself, stopping the
// server. The server must be saved separately to persist the world and settings.
func (s *Server) Stop() error {
	if err := s.Command([]string{"stop"}); err != nil {
		return fmt.Errorf("%s: running 'stop' command in server cli to stop server process: %s", s.ContainerName, err)
	}

	logger.Info.Printf("stopping %s\n", s.ContainerName)

	timeout := time.Duration(stopTimeout)

	err := s.ContainerStop(
		context.Background(),
		s.ContainerID,
		&timeout,
	)

	if err != nil {
		return fmt.Errorf("%s: stopping docker container: %s", s.ContainerName, err)
	}

	return nil
}

func (s *Server) IsRunning() bool {
	inspect, err := s.ContainerInspect(context.Background(), s.ContainerID)
	if err != nil {
		logger.Error.Panic(err)
	}

	return inspect.State.Running
}

func (s *Server) HasVolume() bool {
	inspect, err := s.ContainerInspect(context.Background(), s.ContainerID)
	if err != nil {
		logger.Error.Panic(err)
	}

	return len(inspect.Mounts) > 0
}

// RunBedrock runs the bedrock server process and waits for confirmation from the server that the process has started.
// The server should be join-able when this function returns.
func (s *Server) RunBedrock() error {
	// New the bedrock_server process
	if err := s.Command(strings.Split(RunMCCommand, " ")); err != nil {
		s.StopOrPanic()
		return err
	}

	logs, err := s.LogReader(-1) // Negative number results in all logs
	if err != nil {
		s.StopOrPanic()
		return err
	}

	scanner := bufio.NewScanner(logs)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		if scanner.Text() == "[INFO] Server started." {
			// Server has finished starting
			return nil
		}
	}

	return fmt.Errorf("reached end of log reader without finding the 'Server started' message")
}

// StopOrPanic stops the server's container. The server process may not be stopped gracefully, call Server.Stop() to
// safely stop the server. If an error occurs while attempting to stop the server the program exits with a panic.
func (s *Server) StopOrPanic() {
	logger.Info.Printf("stopping %s\n", s.ContainerName)

	timeout := time.Duration(stopTimeout)

	err := s.ContainerStop(
		context.Background(),
		s.ContainerID,
		&timeout,
	)

	if err != nil {
		logger.Error.Panicf("while stopping %s another error occurred: %s\n", s.ContainerName, err)
	}
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
		docker.ContainerListOptions{All: true},
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
			if errors.Is(err, &NotCraftError{}) || !s.IsRunning() {
				continue
			}

			return nil, fmt.Errorf("creating client for container '%s': %s", n, err)
		}

		servers = append(servers, s)
	}

	return servers, nil
}

func containerID(name string, client client.ContainerAPIClient) (string, error) {
	containers, err := client.ContainerList(context.Background(), docker.ContainerListOptions{All: true})
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

// nextAvailablePort returns the next available port, starting with the default mc port. It checks the first exposed
// port of all running containers to determine if a port is in use.
func nextAvailablePort() int {
	servers, err := All(dockerClient())
	if err != nil {
		panic(err)
	}

	usedPorts := make([]int, len(servers))

	for i, s := range servers {
		p, err := s.Port()
		if err != nil {
			panic(err)
		}

		usedPorts[i] = p
	}

	// Iterate 100 ports starting with the default
OUTER:
	for p := defaultPort; p < defaultPort+100; p++ {
		for _, up := range usedPorts {
			if p == up {
				// Another server is using this port
				continue OUTER
			}
		}

		// The port is available
		return p
	}

	panic("100 ports were not available")
}

// NotFoundError tells the caller that no containers were found with the given name.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("container with name '%s' not found.", e.Name)
}

// Is implements Is(error) to support errors.Is
func (e *NotFoundError) Is(tgt error) bool {
	_, ok := tgt.(*NotFoundError)
	return ok
}

// NotCraftError reports the instance where a container is found with a given name but lacks the label
// indicating that it is managed using craft.
type NotCraftError struct {
	Name string
}

func (e *NotCraftError) Error() string {
	return fmt.Sprintf("container found with name '%s' but it does not appear to be a craft server.", e.Name)
}

// Is implements Is(error) to support errors.Is
func (e *NotCraftError) Is(tgt error) bool {
	_, ok := tgt.(*NotCraftError)
	return ok
}

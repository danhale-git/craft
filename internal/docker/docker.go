package docker

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Client creates a default docker client.
func Client() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
}

// RunContainer creates a new craft server container and returns a docker client for it.
// It is the equivalent of the following docker command:
//
//    docker run -d -e EULA=TRUE -p <HOST_PORT>:19132/udp <IMAGE_NAME>
func RunContainer(hostPort int, name string) (*Container, error) {
	if hostPort == 0 {
		hostPort = nextAvailablePort()
	}

	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	ctx := context.Background()

	// Create port binding between host ip:port and container port
	hostBinding := nat.PortBinding{
		HostIP:   anyIP,
		HostPort: strconv.Itoa(hostPort),
	}

	containerPort, err := nat.NewPort(protocol, strconv.Itoa(defaultPort))
	if err != nil {
		return nil, fmt.Errorf("creating container port: %s", err)
	}

	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	// Request creation of container
	createResp, err := c.ContainerCreate(
		ctx,
		&container.Config{
			Image: imageName,
			Env:   []string{"EULA=TRUE"},
			ExposedPorts: nat.PortSet{
				containerPort: struct{}{},
			},
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
			OpenStdin:    true,
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

	// Start the container
	err = c.ContainerStart(ctx, createResp.ID, docker.ContainerStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("starting container: %s", err)
	}

	d := Container{
		ContainerAPIClient: c,
		ContainerName:      name,
		containerID:        createResp.ID,
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

// ContainerNames returns a slice containing the names of all running containers.
func ContainerNames() ([]string, error) {
	containers, err := Client().ContainerList(
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

	return names, nil
}

// nextAvailablePort returns the next available port, starting with the default mc port. It checks the first exposed
// port of all running containers to determine if a port is in use.
func nextAvailablePort() int {
	clients, err := ActiveServerClients()
	if err != nil {
		panic(err)
	}

	usedPorts := make([]int, len(clients))

	for i, c := range clients {
		p, err := c.GetPort()
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

// ActiveServerClients returns a Container for each active server.
func ActiveServerClients() ([]*Container, error) {
	names, err := ContainerNames()
	if err != nil {
		return nil, fmt.Errorf("getting server names: %s", err)
	}

	clients := make([]*Container, len(names))

	for i, n := range names {
		c, err := NewContainer(n)
		if err != nil {
			return nil, fmt.Errorf("creating client for container '%s': %s", n, err)
		}

		clients[i] = c
	}

	return clients, nil
}

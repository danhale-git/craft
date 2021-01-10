package docker

import (
	"context"
	"fmt"
	"log"
	"strings"

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

// ActiveServerClients returns a DockerClient for each active server.
func ActiveServerClients() ([]*DockerClient, error) {
	names, err := ContainerNames()
	if err != nil {
		return nil, fmt.Errorf("getting server names: %s", err)
	}

	clients := make([]*DockerClient, len(names))

	for i, n := range names {
		c, err := NewDockerClient(n)
		if err != nil {
			return nil, fmt.Errorf("creating client for container '%s': %s", n, err)
		}

		clients[i] = c
	}

	return clients, nil
}

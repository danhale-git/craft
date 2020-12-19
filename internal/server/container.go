package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Container embeds dockers client.Client* and refers to existing container, created using Run()
type Container struct {
	*client.Client
	ID string
}

// ContainerFromName returns the container with the given name or exits with an error if that container doesn't exist.
func GetContainerOrExit(name string) *Container {
	c := GetContainer(name)
	if c == nil {
		fmt.Printf("Container with name '%s' does not exist.\n", name)
		os.Exit(0)
	}

	return c
}

// dockerClient creates a default docker client.
func dockerClient() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
}

// ContainerFromName returns the container with the given name or nil if that container doesn't exist.
func GetContainer(name string) *Container {
	cl := dockerClient()

	containers, err := cl.ContainerList(context.Background(), docker.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	foundCount := 0

	var dc *docker.Container

	for _, ctr := range containers {
		if strings.Trim(ctr.Names[0], "/") == name {
			dc = &ctr
			foundCount++
		}
	}

	// This should never happen as docker doesn't allow containers with matching namess
	if foundCount > 1 {
		panic(fmt.Sprintf("ERROR: more than 1 docker containers exist with name: %s\n", name))
	}

	if dc == nil {
		return nil
	}

	return &Container{
		Client: cl,
		ID:     dc.ID,
	}
}

func (c *Container) name() string {
	ci, err := c.ContainerInspect(
		context.Background(),
		c.ID,
	)
	if err != nil {
		log.Fatalf("Error inspecting container '%s': %s", c.ID, err)
	}

	return ci.Name
}

func (c *Container) copyFrom(containerPath string) (*Archive, error) {
	data, _, err := dockerClient().CopyFromContainer(
		context.Background(),
		c.ID,
		containerPath,
	)
	if err != nil {
		return nil, fmt.Errorf("copying world data from server: %s", err)
	}

	archive, err := FromTar(data)
	if err != nil {
		return nil, fmt.Errorf("reading tar data to file archive: %s", err)
	}

	return archive, nil
}

func (c *Container) copyTo(path string, files *Archive) error {
	t, err := files.Tar()
	if err != nil {
		return err
	}

	err = dockerClient().CopyToContainer(
		context.Background(),
		c.ID,
		path,
		t,
		docker.CopyToContainerOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) logReader() *bufio.Reader {
	logs, err := c.ContainerLogs(
		context.Background(),
		c.ID,
		docker.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "0",
			Follow:     true,
		},
	)

	if err != nil {
		log.Fatalf("creating client: %s", err)
	}

	return bufio.NewReader(logs)
}

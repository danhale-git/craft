package docker

import (
	"bufio"
	"context"
	"log"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Container embeds dockers client.Client* and refers to existing container, created using Run()
type Container struct {
	*client.Client
	ID string
}

// GetContainer constructs a Container struct with the given ID.
func GetContainer(containerID string) *Container {
	return &Container{
		Client: newClient(),
		ID:     containerID,
	}
}

func (s *Container) name() string {
	c, err := s.ContainerInspect(
		context.Background(),
		s.ID,
	)
	if err != nil {
		log.Fatalf("Error inspecting container '%s': %s", s.ID, err)
	}

	return c.Name
}

func (s *Container) logReader() *bufio.Reader {
	logs, err := s.ContainerLogs(
		context.Background(),
		s.ID,
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

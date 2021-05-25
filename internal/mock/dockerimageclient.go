package mock

import (
	"github.com/docker/docker/client"
)

type DockerImageClient interface {
	client.ImageAPIClient
}

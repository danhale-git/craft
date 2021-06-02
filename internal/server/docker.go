package server

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/moby/term"

	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/danhale-git/craft/internal/logger"

	_ "embed" // use embed package in this script

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	anyIP       = "0.0.0.0"                        // Refers to any/all IPv4 addresses
	defaultPort = 19132                            // Default port for player connections
	protocol    = "UDP"                            // MC uses UDP
	imageName   = "craft_bedrock_server:autobuild" // The name of the docker image to use
)

func DockerClient() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
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

// GetPort returns the port players use to connect to this server.
func GetPort(s *Server) (int, error) {
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

// DockerImageExists returns true if the craft server image exists.
func DockerImageExists(c client.ImageAPIClient) (bool, error) {
	images, err := c.ImageList(context.Background(), docker.ImageListOptions{})
	if err != nil {
		return false, err
	}

	for _, img := range images {
		if len(img.RepoTags) > 0 && img.RepoTags[0] == imageName {
			return true, nil
		}
	}

	return false, nil
}

//go:embed Dockerfile
var dockerfile []byte //nolint:gochecknoglobals // embed needs a global

// BuildDockerImage builds the server image.
func BuildDockerImage(serverURL string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Tar embedded dockerfile
	hdr := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing header: %s", err)
	}

	if _, err := tw.Write(dockerfile); err != nil {
		return fmt.Errorf("writing body: %s", err)
	}

	if err := tw.Close(); err != nil {
		return err
	}

	// Build image
	response, err := DockerClient().ImageBuild(
		context.Background(),
		&buf,
		docker.ImageBuildOptions{
			Dockerfile: "Dockerfile",
			Tags:       []string{imageName},
			BuildArgs: map[string]*string{
				"URL": &serverURL,
			},
		},
	)

	if err != nil {
		return err
	}

	// Output from build process
	termFd, isTerm := term.GetFdInfo(os.Stderr)

	err = jsonmessage.DisplayJSONMessagesStream(
		response.Body,
		os.Stderr,
		termFd,
		isTerm,
		nil,
	)
	if err != nil {
		return err
	}

	return response.Body.Close()
}

// nextAvailablePort returns the next available port, starting with the default mc port. It checks the first exposed
// port of all running containers to determine if a port is in use.
func nextAvailablePort() int {
	servers, err := All(DockerClient())
	if err != nil {
		panic(err)
	}

	usedPorts := make([]int, len(servers))

	for i, s := range servers {
		p, err := GetPort(s)
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

package craft

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/danhale-git/craft/internal/server"

	"github.com/moby/term"

	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/danhale-git/craft/internal/logger"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	_ "embed" // use embed package in this script

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	anyIP       = "0.0.0.0"                      // Refers to any/all IPv4 addresses
	defaultPort = 19132                          // Default port for player connections
	protocol    = "UDP"                          // MC uses UDP
	imageName   = "danhaledocker/craftmine:v1.9" // The name of the docker image to use
)

func newClient() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
}

// ServerClients returns a Container for each active server.
func ServerClients() ([]*server.Server, error) {
	names, err := containerNames()
	if err != nil {
		return nil, fmt.Errorf("getting server names: %s", err)
	}

	servers := make([]*server.Server, 0)

	for _, n := range names {
		s, err := server.New(n)
		if err != nil {
			if _, ok := err.(*server.NotCraftError); ok {
				continue
			}

			return nil, fmt.Errorf("creating client for container '%s': %s", n, err)
		}

		servers = append(servers, s)
	}

	return servers, nil
}

//go:embed Dockerfile
var dockerfile []byte //nolint:gochecknoglobals // embed needs a global

// BuildImage builds the server image.
func BuildImage() error {
	c := newClient()

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
	response, err := c.ImageBuild(
		context.Background(),
		&buf,
		docker.ImageBuildOptions{
			Dockerfile: "Dockerfile",
			Tags:       []string{imageName},
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

// CheckImage returns true if the craft server image exists.
func CheckImage() (bool, error) {
	c := newClient()

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

// RunContainer creates a new craft server container and returns a docker client for it.
// It is the equivalent of the following docker command:
//
//    docker run -d -e EULA=TRUE -p <HOST_PORT>:19132/udp <imageName>
func RunContainer(hostPort int, name string) (*server.Server, error) {
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
			Labels:    map[string]string{server.CraftLabel: ""},
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

	s := server.Server{
		ContainerAPIClient: c,
		ContainerName:      name,
		ContainerID:        createResp.ID,
	}

	return &s, nil
}

// containerNames returns a slice containing the names of all running containers.
func containerNames() ([]string, error) {
	containers, err := newClient().ContainerList(
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
	servers, err := ServerClients()
	if err != nil {
		panic(err)
	}

	usedPorts := make([]int, len(servers))

	for i, s := range servers {
		p, err := getPort(s)
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

// getPort returns the port players use to connect to this server
func getPort(s *server.Server) (int, error) {
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

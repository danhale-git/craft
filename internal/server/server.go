package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

var c *client.Client

const (
	defaultPort = 19132
	protocol    = "UDP"
	imageName   = "danhaledocker/craftmine:v1.3"
	anyIP       = "0.0.0.0"
)

type Server struct {
	cli *docker.HijackedResponse

	// Terminal io to docker container bedrock_server process
	// Container things
}

func (s *Server) Read(p []byte) (n int, err error) {
	n, err = s.cli.Reader.Read(p)

	if n > 2 {
		log.Printf("Read %d bytes", n)
	}

	return
}

func (s *Server) Write(p []byte) (n int, err error) {
	n, err = s.cli.Conn.Write(p)

	if len(p) > 1 {
		fmt.Printf("Wrote %d bytes: '%s'\n", n, p)
	}

	return
}

// docker run -d -e EULA=TRUE -p <HOST_PORT>:19132/udp danhaledocker/craftmine:<VERSION>
func Run(hostPort int, name string) error {
	if c == nil {
		newClient()
	}

	ctx := context.Background()

	// Create port binding between host ip:port and container port
	hostBinding := nat.PortBinding{
		HostIP:   anyIP,
		HostPort: strconv.Itoa(hostPort),
	}

	containerPort, err := nat.NewPort(protocol, strconv.Itoa(defaultPort))
	if err != nil {
		return err
	}

	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	//shell := strslice.StrSlice{}

	// Request creation of container
	createResp, err := c.ContainerCreate(
		ctx,
		&container.Config{
			Image: imageName,
			Env:   []string{"EULA=TRUE"},
			ExposedPorts: nat.PortSet{
				containerPort: struct{}{},
			},
			//Shell: shell,
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
		return err
	}

	waiter, err := c.ContainerAttach(ctx, createResp.ID, docker.ContainerAttachOptions{
		Stderr: true,
		Stdout: true,
		Stdin:  true,
		Stream: true,
	})

	server := Server{cli: &waiter}

	if err != nil {
		return err
	}

	go func() {
		if _, err = io.Copy(os.Stdout, &server); err != nil {
			panic(err)
		}
	}()

	go func() {
		if _, err = io.Copy(&server, os.Stdin); err != nil {
			panic(err)
		}
	}()

	// Start the container
	err = c.ContainerStart(ctx, createResp.ID, docker.ContainerStartOptions{})
	if err != nil {
		return err
	}

	// Wait for the container to stop running
	statusCh, errCh := c.ContainerWait(ctx, createResp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	return nil
}

func newClient() {
	var err error
	if c, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err != nil {
		log.Fatalf("Error: Failed to create new docker client: %s", err)
	}
}

func PrintRunningContainers() {
	containers, err := c.ContainerList(context.Background(), docker.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, ctr := range containers {
		fmt.Printf("%s %s\n", ctr.ID[:10], ctr.Image)
	}
}

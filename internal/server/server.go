package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	defaultPort = 19132
	protocol    = "UDP"
	imageName   = "danhaledocker/craftmine:v1.3"
	anyIP       = "0.0.0.0"
)

// Run is the equivalent of the following docker command
//
//    docker run -d -e EULA=TRUE -p <HOST_PORT>:19132/udp danhaledocker/craftmine:<VERSION>
func Run(hostPort int, name string) error {
	c := newClient()
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
		return err
	}

	// Stream server output to stdout
	waiter, err := c.ContainerAttach(ctx, createResp.ID, docker.ContainerAttachOptions{
		Stderr: true,
		Stdout: true,
		Stream: true,
	})
	if err != nil {
		return err
	}

	go func() {
		if _, err = io.Copy(os.Stdout, waiter.Reader); err != nil {
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

func Command(serverID string, args []string) ([]byte, error) {
	waiter, err := newClient().ContainerAttach(
		context.Background(),
		serverID,
		docker.ContainerAttachOptions{
			Stderr: true,
			Stdout: true,
			Stdin:  true,
			Stream: true,
		},
	)

	if err != nil {
		return nil, err
	}

	// Write the command to the server cli
	_, err = waiter.Conn.Write([]byte(
		strings.Join(args, " "),
	))
	if err != nil {
		return nil, err
	}

	cli := bufio.NewReader(waiter.Reader)

	// Discard the echo of the command
	if _, err := cli.ReadString('\n'); err != nil {
		return nil, err
	}

	// Read the response to the command
	out, err := cli.ReadString('\n')
	if err != nil {
		return nil, err
	}

	return []byte(strings.TrimSpace(out)), nil
}

func newClient() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
}

// ContainerFromName returns the container with the given name.
func ContainerFromName(name string) (c *docker.Container, b bool) {
	containers, err := newClient().ContainerList(context.Background(), docker.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	foundCount := 0

	for _, ctr := range containers {
		if strings.Trim(ctr.Names[0], "/") == name {
			c = &ctr
			b = true
			foundCount++
		}
	}

	if foundCount > 1 {
		panic(fmt.Sprintf("WARNING: more than 1 docker containers exist with name: %s\n", name))
	}

	return
}

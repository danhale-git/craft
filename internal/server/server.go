package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"golang.org/x/crypto/ssh/terminal"

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
	// Terminal io to docker container bedrock_server process
	// Container things
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

	if err != nil {
		return err
	}

	go io.Copy(os.Stdout, waiter.Reader)
	//go io.Copy(os.Stderr, waiter.Reader) // this causes an index out of range exception in bufio.go, presumably because the reader is being read from twice. It should probably be copied.
	go io.Copy(waiter.Conn, os.Stdin)

	// Start the container
	err = c.ContainerStart(ctx, createResp.ID, docker.ContainerStartOptions{})
	if err != nil {
		return err
	}

	// Save terminal state
	fd := int(os.Stdin.Fd())

	var oldState *terminal.State

	if terminal.IsTerminal(fd) {
		oldState, err = terminal.MakeRaw(fd)
		if err != nil {
			return err
		}
		defer terminal.Restore(fd, oldState)
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

	// Restore terminal state
	terminal.Restore(fd, oldState)

	// Panics if AutoRemove=true
	/*// Print container logs
	out, err := c.ContainerLogs(ctx, createResp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadAll(out)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(b))*/

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

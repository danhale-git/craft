package docker

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	defaultPort = 19132                          // Default port for player connections
	protocol    = "UDP"                          // MC uses UDP
	imageName   = "danhaledocker/craftmine:v1.6" // The name of the docker image to use
	anyIP       = "0.0.0.0"                      // Refers to any/all IPv4 addresses

	// Directory to save imported world files
	worldDirectory           = "/bedrock/worlds/Bedrock level"
	mcDirectory              = "/bedrock"
	serverPropertiesFileName = "server.properties"

	// Run the bedrock_server executable and append its output to log.txt
	runMCCommand = "cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server" // >> log.txt 2>&1"

	saveHoldDelayMilliseconds = 100
)

// newClient creates a default docker client.
func newClient() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
}

// Run is the equivalent of the following docker command
//
//    docker run -d -e EULA=TRUE -p <HOST_PORT>:19132/udp <IMAGE_NAME>
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

	// Start the container
	err = c.ContainerStart(ctx, createResp.ID, docker.ContainerStartOptions{})
	if err != nil {
		return err
	}

	return nil
}

// CopyWorldToContainer reads a .mcworld zip file and copies the contents to the active world directory for this
// container.
func CopyWorldToContainer(containerID, mcworldPath string) error {
	// Open a zip archive for reading.
	r, err := zip.OpenReader(mcworldPath)
	if err != nil {
		return err
	}

	w, err := FromZip(r)
	if err != nil {
		return err
	}

	if err = r.Close(); err != nil {
		return err
	}

	return copyToContainer(containerID, worldDirectory, w)
}

// CopyServerPropertiesToContainer copies the fle at the given path to the mc server directory on the container. The
// file is always renamed to the value of serverPropertiesFileName (server.properties).
func CopyServerPropertiesToContainer(containerID, propsPath string) error {
	propsFile, err := os.Open(propsPath)
	if err != nil {
		return fmt.Errorf("opening file '%s': %s", propsPath, err)
	}

	p, err := FromFiles([]*os.File{propsFile})
	if err != nil {
		return fmt.Errorf("creating archive: %s", err)
	}

	p.Files[0].Name = serverPropertiesFileName

	return copyToContainer(containerID, mcDirectory, p)
}

func copyToContainer(containerID, path string, files *Archive) error {
	t, err := files.Tar()
	if err != nil {
		return err
	}

	err = newClient().CopyToContainer(
		context.Background(),
		containerID,
		path,
		t,
		docker.CopyToContainerOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

// RunMC runs the mc server process on a container.
func RunMC(containerID string) error {
	waiter, err := newClient().ContainerAttach(
		context.Background(),
		containerID,
		docker.ContainerAttachOptions{
			Stdin:  true,
			Stream: true,
		},
	)
	if err != nil {
		return fmt.Errorf("attaching container: %s", err)
	}

	// Write the command to the server cli
	_, err = waiter.Conn.Write([]byte(runMCCommand + "\n"))
	if err != nil {
		return fmt.Errorf("writing to mc cli: %s", err)
	}

	return nil
}

func Backup(containerID, containerName, destPath string) error {
	logs := logReader(containerID)

	// save hold
	sh, err := commandResponse(containerID, "save hold", logs)
	if err != nil {
		return err
	}

	switch strings.TrimSpace(sh) {
	case "Saving...":
	case "The command is already running":
		break
	default:
		return fmt.Errorf("unexpected response to `save hold`: '%s'", sh)
	}

	// save query
	for {
		sq, err := commandResponse(containerID, "save query", logs)
		if err != nil {
			return err
		}

		// Ready for backup
		if strings.HasPrefix(sq, "Data saved. Files are now ready to be copied.") {
			if err := copyWorldFromContainer(containerID, destPath, containerName); err != nil {
				return fmt.Errorf("copying world data: %s", err)
			}

			// This command returns two lines in response. Read the second one to discard it.
			if _, err := logs.ReadString('\n'); err != nil {
				return fmt.Errorf("reading 'save query' file list response: %s", err)
			}

			break
		}

		// Give the game time to hold.
		time.Sleep(saveHoldDelayMilliseconds * time.Millisecond)
	}

	// save resume
	sr, err := commandResponse(containerID, "save resume", logs)
	if err != nil {
		return err
	}

	if strings.TrimSpace(sr) != "Changes to the level are resumed." {
		return fmt.Errorf("unexpected response to `save resume`: '%s'", sr)
	}

	return nil
}

func copyWorldFromContainer(containerID, destPath, containerName string) error {
	data, _, err := newClient().CopyFromContainer(
		context.Background(),
		containerID,
		worldDirectory,
	)
	if err != nil {
		return fmt.Errorf("copying world data from server: %s", err)
	}

	archive, err := FromTar(data)
	if err != nil {
		return fmt.Errorf("reading tar data to file archive: %s", err)
	}

	// Remove 'Bedrock level' directory.
	files := make([]File, 0)

	for _, f := range archive.Files {
		f.Name = strings.Replace(f.Name, "Bedrock level/", "", 1)

		// Skip the file representing the 'Bedrock level' directory.
		if len(strings.TrimSpace(f.Name)) == 0 {
			continue
		}

		files = append(files, f)
	}

	archive.Files = files

	// Convert backup data from tar to zip and save to disk
	z, err := archive.Zip()
	if err != nil {
		return fmt.Errorf("creating zip data from file archive: %s", err)
	}

	y, m, d := time.Now().Date()
	fileName := fmt.Sprintf("%s_backup_%d-%d-%d.mcworld", containerName, y, m, d)
	p := path.Join(destPath, fileName)

	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("creating backup file at '%s': %s", destPath, err)
	}

	_, err = f.Write(z.Bytes())
	if err != nil {
		return fmt.Errorf("writing zip data bytes: %s", err)
	}

	return nil
}

func commandResponse(containerID, cmd string, logs *bufio.Reader) (string, error) {
	err := command(containerID, cmd)
	if err != nil {
		return "", fmt.Errorf("running command `%s`: %s", cmd, err)
	}

	// Read command echo to discard it
	if _, err := logs.ReadString('\n'); err != nil {
		return "", fmt.Errorf("retrieving echo for command `%s`: %s", cmd, err)
	}

	// Read command response
	response, err := logs.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("retrieving `%s` response: %s", cmd, err)
	}

	return response, nil
}

func logReader(containerID string) *bufio.Reader {
	logs, err := newClient().ContainerLogs(
		context.Background(),
		containerID,
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

func Tail(containerID string, tail int) *bufio.Reader {
	logs, err := newClient().ContainerLogs(
		context.Background(),
		containerID,
		docker.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       strconv.Itoa(tail),
			Follow:     true,
		},
	)

	if err != nil {
		log.Fatalf("creating client: %s", err)
	}

	return bufio.NewReader(logs)
}

// Command runs the given arguments separated by spaces as a command in  the bedrock_server process cli on a container.
func Command(containerID string, args []string) error {
	// TODO: Log this command
	// Attach to the container
	waiter, err := newClient().ContainerAttach(
		context.Background(),
		containerID,
		docker.ContainerAttachOptions{
			Stdin:  true,
			Stream: true,
		},
	)

	if err != nil {
		return err
	}

	commandString := strings.Join(args, " ") + "\n"

	// Write the command to the bedrock_server process cli
	_, err = waiter.Conn.Write([]byte(
		commandString,
	))
	if err != nil {
		return err
	}

	return nil
}
func command(containerID string, cmd string) error {
	return Command(containerID, strings.Split(cmd, " "))
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

	// This should never happen as docker doesn't allow containers with matching namess
	if foundCount > 1 {
		panic(fmt.Sprintf("ERROR: more than 1 docker containers exist with name: %s\n", name))
	}

	return
}

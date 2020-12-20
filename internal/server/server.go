package server

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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
	saveHoldQueryRetries      = 100
)

// Run is the equivalent of the following docker command
//
//    docker run -d -e EULA=TRUE -p <HOST_PORT>:19132/udp <IMAGE_NAME>
func Run(hostPort int, name string) error {
	c := dockerClient()
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

// LoadBackup reads the file at backupPath as a zip archive. The archive must contain a valid .mcworld file.
func LoadBackup(c *Container, backupPath string) error {
	// Open a zip archive for reading.
	z, err := zip.OpenReader(backupPath)
	if err != nil {
		return err
	}

	defer z.Close()

	foundWorld := false

	for _, file := range z.File {
		f, err := file.Open()
		if err != nil {
			return err
		}

		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		if strings.HasSuffix(file.Name, ".mcworld") {
			// World file is copied to the 'Bedrock level' directory
			foundWorld = true

			z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
			if err != nil {
				return err
			}

			w, err := NewArchiveFromZip(z)
			if err != nil {
				return err
			}

			err = c.copyTo(worldDirectory, w)
			if err != nil {
				return err
			}
		} else {
			// Other files are copied to the directory containing the mc server executable
			a := Archive{}

			a.AddFile(File{
				Name: file.Name,
				Body: b,
			})

			return c.copyTo(mcDirectory, &a)
		}
	}

	if !foundWorld {
		return fmt.Errorf("no .mcworld file present in backup")
	}

	return nil
}

// LoadWorld reads a .mcworld zip file and copies the contents to the active world directory for this
// container.
func LoadWorld(c *Container, mcworldPath string) error {
	// Open a zip archive for reading.
	r, err := zip.OpenReader(mcworldPath)
	if err != nil {
		return err
	}

	w, err := NewArchiveFromZip(&r.Reader)
	if err != nil {
		return err
	}

	if err = r.Close(); err != nil {
		return err
	}

	return c.copyTo(worldDirectory, w)
}

// LoadServerProperties copies the fle at the given path to the mc server directory on the container. The
// file is always renamed to the value of serverPropertiesFileName (server.properties).
func LoadServerProperties(c *Container, propsPath string) error {
	propsFile, err := os.Open(propsPath)
	if err != nil {
		return fmt.Errorf("opening file '%s': %s", propsPath, err)
	}

	a, err := NewArchiveFromFiles([]*os.File{propsFile})
	if err != nil {
		return fmt.Errorf("creating archive: %s", err)
	}

	a.Files[0].Name = serverPropertiesFileName

	return c.copyTo(mcDirectory, a)
}

// RunServer runs the mc server process on a container.
func RunServer(c *Container) error {
	waiter, err := c.ContainerAttach(
		context.Background(),
		c.ID,
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

// Backup runs the save hold/query/resume command sequence and saves a .mcworld file snapshot to the given local path.
func Backup(c *Container, destPath string) error {
	logs := c.logReader()

	saveHold, err := commandResponse(c.ID, "save hold", logs)
	if err != nil {
		return err
	}

	switch strings.TrimSpace(saveHold) {
	case "Saving...":
	case "The command is already running":
		break
	default:
		return fmt.Errorf("unexpected response to `save hold`: '%s'", saveHold)
	}

	// Query until ready for backup
	for i := 0; i < saveHoldQueryRetries; i++ {
		time.Sleep(saveHoldDelayMilliseconds * time.Millisecond)

		saveQuery, err := commandResponse(c.ID, "save query", logs)
		if err != nil {
			return err
		}

		// Ready for backup
		if strings.HasPrefix(saveQuery, "Data saved. Files are now ready to be copied.") {
			err = backupServer(c, destPath, logs)
			if err != nil {
				return fmt.Errorf("backing up server: %s", err)
			}

			break
		}
	}

	// save resume
	saveResume, err := commandResponse(c.ID, "save resume", logs)
	if err != nil {
		return err
	}

	if strings.TrimSpace(saveResume) != "Changes to the level are resumed." {
		return fmt.Errorf("unexpected response to `save resume`: '%s'", saveResume)
	}

	return nil
}

func backupServer(c *Container, destPath string, logs *bufio.Reader) error {
	y, m, d := time.Now().Date()

	backupName := fmt.Sprintf("%s_%d-%d-%d_%s",
		c.name(), y, m, d,
		strings.Replace(time.Now().Format(time.Kitchen), ":", "-", 1))

	serverBackup := Archive{}

	// Back up world
	worldArchive, err := copyWorldFromContainer(c)
	if err != nil {
		return fmt.Errorf("copying world data from container: %s", err)
	}

	wz, err := worldArchive.Zip()
	if err != nil {
		return err
	}

	serverBackup.AddFile(File{
		Name: fmt.Sprintf("%s.mcworld", backupName),
		Body: wz.Bytes(),
	})

	// Back up settings
	serverPropertiesArchive, err := copyServerPropertiesFromContainer(c)
	if err != nil {
		return err
	}

	serverBackup.AddFile(File{
		Name: serverPropertiesFileName,
		Body: serverPropertiesArchive.Files[0].Body,
	})

	// Save to disk
	err = serverBackup.SaveZip(path.Join(destPath, c.name()), fmt.Sprintf("%s.zip", backupName))
	if err != nil {
		return fmt.Errorf("saving server backup: %s", err)
	}

	// A second line is returned with a list of files, read it to discard it.
	if _, err := logs.ReadString('\n'); err != nil {
		return fmt.Errorf("reading 'save query' file list response: %s", err)
	}

	return nil
}

func copyWorldFromContainer(c *Container) (*Archive, error) {
	// Copy the world directory and it's contents from the container
	a, err := c.copyFrom(worldDirectory)
	if err != nil {
		return nil, err
	}

	// Remove 'Bedrock level' directory
	files := make([]File, 0)

	for _, f := range a.Files {
		f.Name = strings.Replace(f.Name, "Bedrock level/", "", 1)

		// Skip the file representing the 'Bedrock level' directory
		if len(strings.TrimSpace(f.Name)) == 0 {
			continue
		}

		files = append(files, f)
	}

	a.Files = files

	return a, nil
}

func copyServerPropertiesFromContainer(c *Container) (*Archive, error) {
	a, err := c.copyFrom(path.Join(mcDirectory, serverPropertiesFileName))
	if err != nil {
		return nil, fmt.Errorf("copying '%s' from container path %s: %s", serverPropertiesFileName, mcDirectory, err)
	}

	return a, nil
}

func Tail(containerID string, tail int) *bufio.Reader {
	logs, err := dockerClient().ContainerLogs(
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
	waiter, err := dockerClient().ContainerAttach(
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

func command(containerID string, cmd string) error {
	return Command(containerID, strings.Split(cmd, " "))
}

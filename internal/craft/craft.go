package craft

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"time"

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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
	RunMCCommand = "cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server" // >> log.txt 2>&1"

	saveHoldDelayMilliseconds = 100
	saveHoldQueryRetries      = 100

	backupFilenameTimeLayout = "02-01-2006_15-04"
)

// dockerClient creates a default docker client.
func dockerClient() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
}

// ListNames returns the name of all containers as a slice of strings.
func ListNames() ([]string, error) {
	containers, err := dockerClient().ContainerList(
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

func BackupServerNames(backupDir string) ([]string, error) {
	infos, err := ioutil.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory '%s': %s", backupDir, err)
	}

	names := make([]string, len(infos))
	for i, f := range infos {
		names[i] = f.Name()
	}

	return names, nil
}

func LatestServerBackup(serverName, backupDir string) (string, error) {
	infos, err := ioutil.ReadDir(path.Join(backupDir, serverName))
	if err != nil {
		return "", fmt.Errorf("reading directory '%s': %s", backupDir, err)
	}

	var mostRecent time.Time

	var mostRecentFileName string

	for _, f := range infos {
		name := f.Name()

		prefix := fmt.Sprintf("%s_", serverName)
		if strings.HasPrefix(name, prefix) {
			backupTime := strings.Replace(name, prefix, "", 1)
			backupTime = strings.Split(backupTime, ".")[0]

			t, err := time.Parse(backupFilenameTimeLayout, backupTime)
			if err != nil {
				return "", fmt.Errorf("parsing time from file name '%s': %s", name, err)
			}

			if t.After(mostRecent) {
				mostRecent = t
				mostRecentFileName = name
			}
		}
	}

	return mostRecentFileName, nil
}

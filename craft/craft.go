package craft

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/danhale-git/craft/internal/configure"
	"github.com/danhale-git/craft/internal/server"

	"github.com/danhale-git/craft/internal/docker"
)

const (
	RunMCCommand = "cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server"
)

// CreateServer spawns a new craft server. Only the name is required. Full path to a .mcworld file, port and a slice of
// "property=newvalue" strings may also be provided.
func CreateServer(name, mcworld string, port int, props []string) error {
	// Check the server doesn't already exist
	for _, b := range backupServerNames() {
		if name == b {
			return fmt.Errorf("server name '%s' is in use by a backup, run 'craft list -a'", name)
		}
	}

	// Create a container for the server
	c, err := docker.RunContainer(port, name)
	if err != nil {
		return fmt.Errorf("creating new container: %s", err)
	}

	// Copy world files to the server
	if mcworld != "" {
		if err := LoadMCWorldFile(mcworld, c); err != nil {
			if err := c.Stop(); err != nil {
				panic(err)
			}

			return fmt.Errorf("loading world file: %s", err)
		}
	}

	// Set the properties
	if err := SetServerProperties(props, c); err != nil {
		return fmt.Errorf("setting server properties: %s", err)
	}

	// Run the server process
	if err = RunServer(c); err != nil {
		return fmt.Errorf("starting server process: %s", err)
	}

	return nil
}

// RunLatestBackup sorts all available backup files per date and starts a server from the latest backup.
func RunLatestBackup(name string, port int) (*docker.Container, error) {
	c, err := docker.RunContainer(port, name)
	if err != nil {
		return nil, fmt.Errorf("%s: running server: %s", name, err)
	}

	f := latestBackupFileName(name)

	err = restoreBackup(c, f.Name())
	if err != nil {
		if err := c.Stop(); err != nil {
			panic(err)
		}

		return nil, fmt.Errorf("%s: loading backup file to server: %s", name, err)
	}

	if err = RunServer(c); err != nil {
		if err := c.Stop(); err != nil {
			panic(err)
		}

		return nil, fmt.Errorf("%s: starting server process: %s", name, err)
	}

	return c, nil
}

// runServer executes the server binary and waits for the server to be ready before returning.
func RunServer(c *docker.Container) error {
	// Run the bedrock_server process
	if err := c.Command(strings.Split(RunMCCommand, " ")); err != nil {
		return err
	}

	logs, err := c.LogReader(-1) // Negative number results in all logs
	if err != nil {
		return err
	}

	s := bufio.NewScanner(logs)
	s.Split(bufio.ScanLines)

	for s.Scan() {
		if s.Text() == "[INFO] Server started." {
			// Server has finished starting
			return nil
		}
	}

	return fmt.Errorf("reached end of log reader without finding the 'Server started' message")
}

func Stop(c *docker.Container) error {
	if err := c.Command([]string{"stop"}); err != nil {
		return fmt.Errorf("%s: running 'stop' command in server cli to stop server process: %s", c.ContainerName, err)
	}

	if err := c.Stop(); err != nil {
		return fmt.Errorf("%s: stopping docker container: %s", c.ContainerName, err)
	}

	return nil
}

func SetServerProperties(propFlags []string, c *docker.Container) error {
	if len(propFlags) > 0 {
		k := make([]string, len(propFlags))
		v := make([]string, len(propFlags))

		for i, p := range propFlags {
			s := strings.Split(p, "=")
			if !strings.ContainsRune(p, '=') || len(s[0]) == 0 || len(s[1]) == 0 {
				return fmt.Errorf("invalid property '%s' should be 'key=value'", p)
			}

			k[i] = s[0]
			v[i] = s[1]
		}

		b, err := c.CopyFileFrom(server.FilePaths.ServerProperties)
		if err != nil {
			return err
		}

		updated, err := configure.SetProperties(k, v, b)
		if err != nil {
			return err
		}

		if err = c.CopyFileTo(server.FilePaths.ServerProperties, updated); err != nil {
			return err
		}
	}

	return nil
}

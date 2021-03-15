package craft

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/danhale-git/craft/internal/backup"
	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/configure"
	"github.com/danhale-git/craft/internal/server"

	"github.com/danhale-git/craft/internal/docker"
)

const (
	RunMCCommand = "cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server"
)

// CreateServer spawns a new craft server. Only the name is required. Full path to a .mcworld file, port and a slice of
// "property=newvalue" strings may also be provided.
func CreateServer(name string, port int, props []string, mcworld ZipOpener) error {
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
	if mcworld != nil {
		zr, err := mcworld.Open()
		if err != nil {
			if err := c.Stop(); err != nil {
				logger.Panic(err)
			}

			return fmt.Errorf("inavlid world file: %s", err)
		}

		if err = backup.RestoreMCWorld(&zr.Reader, c.CopyTo); err != nil {
			return fmt.Errorf("restoring backup: %s", err)
		}

		if err = zr.Close(); err != nil {
			logger.Panic(err)
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
	if _, err := docker.NewContainer(name); err == nil {
		return nil, fmt.Errorf("server '%s' is already running (run `craft list`)", name)
	}

	if !BackupExists(name) {
		return nil, fmt.Errorf("stopped server with name '%s' doesn't exist", name)
	}

	c, err := docker.RunContainer(port, name)
	if err != nil {
		return nil, fmt.Errorf("%s: running server: %s", name, err)
	}

	f, err := latestBackupFile(name)
	if err != nil {
		return nil, err
	}

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

// Stop executes a stop command first in the server process cli then on the container itself, stopping the
// server. The server must be saves separately to persist the world and settings.
func Stop(c *docker.Container) error {
	if err := c.Command([]string{"stop"}); err != nil {
		return fmt.Errorf("%s: running 'stop' command in server cli to stop server process: %s", c.ContainerName, err)
	}

	if err := c.Stop(); err != nil {
		return fmt.Errorf("%s: stopping docker container: %s", c.ContainerName, err)
	}

	return nil
}

// SetServerProperties takes a collection of key=value pairs and applies them to the server.properties configuration
// file. If a key is missing, an error will be returned and no changes will be made.
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

		b, err := c.CopyFileFrom(server.FullPaths.ServerProperties)
		if err != nil {
			return err
		}

		updated, err := configure.SetProperties(k, v, b)
		if err != nil {
			return err
		}

		if err = c.CopyFileTo(server.FullPaths.ServerProperties, updated); err != nil {
			return err
		}
	}

	return nil
}

// PrintServers prints a list of servers. If all is true then stopped servers will be printed. Running servers show the
// port players should connect on and stopped servers show the date and time at which they were stopped.
func PrintServers(all bool) error {
	w := tabwriter.NewWriter(os.Stdout, 3, 3, 3, ' ', tabwriter.TabIndent)

	servers, err := docker.ActiveServerClients()
	if err != nil {
		return fmt.Errorf("getting server clients: %s", err)
	}

	for _, s := range servers {
		c, err := docker.NewContainer(s.ContainerName)
		if err != nil {
			return fmt.Errorf("creating docker client for container '%s': %s", s.ContainerName, err)
		}

		port, err := c.GetPort()
		if err != nil {
			return fmt.Errorf("getting port for container '%s': '%s'", s.ContainerName, err)
		}

		if _, err := fmt.Fprintf(w, "%s\trunning - port %d\n", s.ContainerName, port); err != nil {
			return fmt.Errorf("writing to table: %s", err)
		}
	}

	if !all {
		if err = w.Flush(); err != nil {
			return fmt.Errorf("writing output to console: %s", err)
		}

		return nil
	}

	for _, n := range backupServerNames() {
		if func() bool { // if n is an active server
			for _, s := range servers {
				if s.ContainerName == n {
					return true
				}
			}
			return false
		}() {
			continue
		}

		f, err := latestBackupFile(n)
		if err != nil {
			continue
		}

		t, err := backup.FileTime(f.Name())
		if err != nil {
			panic(err)
		}

		if _, err := fmt.Fprintf(w, "%s\tstopped - %s\n", n, t.Format("02 Jan 2006 3:04PM")); err != nil {
			logger.Error.Fatalf("Error writing to table: %s", err)
		}
	}

	if err = w.Flush(); err != nil {
		logger.Error.Fatalf("Error writing output to console: %s", err)
	}

	return nil
}

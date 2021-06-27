package craft

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/danhale-git/craft/internal/files"

	"github.com/danhale-git/craft/mcworld"

	docker "github.com/docker/docker/api/types"

	"github.com/docker/docker/client"

	"github.com/danhale-git/craft/internal/backup"
	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/configure"
	"github.com/danhale-git/craft/server"
)

func DockerClient() *client.Client {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error.Fatalf("Error: Failed to create new docker client: %s", err)
	}

	return c
}

// GetServerOrExit is a convenience function for attempting to find an existing docker container with the given name.
// If not found, a helpful error message is printed and the program exits without error.
func GetServerOrExit(containerName string) *server.Server {
	s, err := server.Get(DockerClient(), containerName)
	if err != nil {
		// Container was not found
		if errors.Is(err, &server.NotFoundError{}) {
			logger.Info.Println(err)
			os.Exit(0)
		} else if errors.Is(err, &server.NotCraftError{}) {
			logger.Info.Println(err)
			os.Exit(0)
		} else if !s.IsRunning() {
			logger.Info.Println(err)
			os.Exit(0)
		}

		// Something else went wrong
		logger.Error.Panic(err)
	}

	return s
}

// NewServer spawns a new craft server. Only the name is required. Full path to a .mcworld file, port and a slice of
// "property=newvalue" strings may also be provided.
func NewServer(name string, port int, props []string, mcw mcworld.ZipOpener, useVolume bool) (*server.Server, error) {
	// Check the server doesn't already exist
	if backupExists(name) {
		return nil, fmt.Errorf("server name '%s' is in use by a backup, run 'craft list -a'", name)
	}

	// Create a container for the server
	c, err := server.New(port, name, useVolume)
	if err != nil {
		return nil, fmt.Errorf("creating new container: %s", err)
	}

	// Copy world files to the server
	if mcw != nil {
		zr, err := mcw.Open()
		if err != nil {
			c.StopOrPanic()
			return nil, fmt.Errorf("inavlid world file: %s", err)
		}

		if err = backup.RestoreMCWorld(&zr.Reader, c.ContainerID, DockerClient()); err != nil {
			c.StopOrPanic()
			return nil, fmt.Errorf("restoring backup: %s", err)
		}

		if err = zr.Close(); err != nil {
			logger.Panic(err)
		}
	}

	// Set the properties
	if err := SetServerProperties(props, c); err != nil {
		return nil, fmt.Errorf("setting server properties: %s", err)
	}

	return c, nil
}

// StartServer sorts all available backup files by date and starts a server from the latest backup.
func StartServer(name string, port int) (*server.Server, error) {
	s, err := server.Get(DockerClient(), name)
	if errors.Is(err, &server.NotFoundError{}) {
		if !backupExists(name) {
			return nil, fmt.Errorf("stopped server with name '%s' doesn't exist", name)
		}

		s, err = startServerFromBackup(name, port)
		if err != nil {
			return nil, fmt.Errorf("starting server from backup: %w", err)
		}

		return s, nil
	}
	if err != nil {
		return nil, err
	}

	if s.IsRunning() {
		return nil, fmt.Errorf("server '%s' is already running (run 'craft list')", name)
	}

	err = s.ContainerStart(
		context.Background(),
		s.ContainerID,
		docker.ContainerStartOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("%s: starting docker container: %s", s.ContainerName, err)
	}

	return s, nil
}

func startServerFromBackup(name string, port int) (*server.Server, error) {
	s, err := server.New(port, name, false)
	if err != nil {
		return nil, fmt.Errorf("%s: running server: %s", name, err)
	}

	f, err := latestBackupFile(name)
	if err != nil {
		s.StopOrPanic()
		return nil, err
	}

	backupPath := filepath.Join(backupDirectory(), s.ContainerName)

	// Open backup zip
	zr, err := zip.OpenReader(filepath.Join(backupPath, f.Name()))
	if err != nil {
		s.StopOrPanic()
		return nil, err
	}

	if err = backup.Restore(&zr.Reader, s.ContainerID, DockerClient()); err != nil {
		s.StopOrPanic()
		return nil, err
	}

	if err = zr.Close(); err != nil {
		s.StopOrPanic()
		return nil, fmt.Errorf("closing zip: %s", err)
	}

	return s, nil
}

// SetServerProperties takes a slice of key=value strings and applies them to the server.properties configuration
// file. If a key is missing, an error will be returned and no changes will be made.
func SetServerProperties(propFlags []string, s *server.Server) error {
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

		containerPath := files.FullPaths.ServerProperties

		data, _, err := s.CopyFromContainer(
			context.Background(),
			s.ContainerID,
			containerPath,
		)
		if err != nil {
			return fmt.Errorf("copying data from server at '%s': %s", containerPath, err)
		}

		tr := tar.NewReader(data)

		_, err = tr.Next()
		if err == io.EOF {
			return fmt.Errorf("no file was found at '%s', got EOF reading tar archive", files.FullPaths.ServerProperties)
		}

		if err != nil {
			return fmt.Errorf("reading tar archive: %s", err)
		}

		b, err := ioutil.ReadAll(tr)
		if err != nil {
			return err
		}

		b, err = configure.SetProperties(k, v, b)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		hdr := &tar.Header{
			Name: filepath.Base(containerPath),
			Size: int64(len(b)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("writing header: %s", err)
		}

		if _, err := tw.Write(b); err != nil {
			return fmt.Errorf("writing body: %s", err)
		}

		err = s.CopyToContainer(
			context.Background(),
			s.ContainerID,
			filepath.Dir(containerPath),
			&buf,
			docker.CopyToContainerOptions{},
		)
		if err != nil {
			return fmt.Errorf("copying files to '%s': %s", filepath.Dir(containerPath), err)
		}
	}

	return nil
}

// PrintServers prints a list of servers. If all is true then stopped servers will be printed. Running servers show the
// port players should connect on and stopped servers show the date and time at which they were stopped.
func PrintServers(all bool) error {
	w := tabwriter.NewWriter(os.Stdout, 3, 3, 3, ' ', tabwriter.TabIndent)

	stoppedContainers := make([]*server.Server, 0)

	servers, err := server.All(DockerClient())
	if err != nil {
		return fmt.Errorf("getting server clients: %s", err)
	}

	// Print running servers
	for _, s := range servers {
		s, err := server.Get(DockerClient(), s.ContainerName)
		if err != nil {
			return fmt.Errorf("creating docker client: %s", err)
		}
		if !s.IsRunning() {
			stoppedContainers = append(stoppedContainers, s)
			continue
		}

		port, err := s.Port()
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

	// Print stopped servers without mounted volumes
	for _, n := range stoppedServerNames() {
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

	// Print stopped servers with mounted volumes
	for _, s := range stoppedContainers {
		inspect, err := s.ContainerInspect(context.Background(), s.ContainerID)
		if err != nil {
			return err
		}

		layout := "2006-01-02T15:04:05.0000000Z"

		t, err := time.Parse(layout, inspect.State.FinishedAt)
		if err != nil {
			return fmt.Errorf("failed to pass stopped time for server '%s': %w", s.ContainerName, err)
		}

		p, err := s.Port()
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "%s\tstopped (volume) - port %d - %s\n", s.ContainerName, p, t.Format("02 Jan 2006 3:04PM")); err != nil {
			logger.Error.Fatalf("Error writing to table: %s", err)
		}
	}

	if err = w.Flush(); err != nil {
		logger.Error.Fatalf("Error writing output to console: %s", err)
	}

	return nil
}

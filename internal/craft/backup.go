package craft

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/docker"
)

const (
	serverDirectoryPath      = "/bedrock"          // The directory where the mc server files are unpacked
	worldsDirectoryName      = "worlds"            // The directory where worlds are stored
	serverPropertiesFileName = "server.properties" // Name of the server properties (configuration) file

	BackupFilenameTimeLayout = "02-01-2006_15-04" // The format of the file timestamp for the Go time package formatter

	saveQueryRetries = 100 // The number of times save query can run without the expected response
	saveQueryDelayMS = 100 // The delay between save query retries, in milliseconds
)

// files backed up for Craft
var craftFiles = []string{
	serverPropertiesFileName, // server.properties
}

func RestoreBackup(d *docker.DockerClient, zr *zip.ReadCloser) error {
	// Write zipped files to tar archive
	for _, f := range zr.File {
		var data bytes.Buffer
		tw := tar.NewWriter(&data)

		file, err := f.Open()
		if err != nil {
			return err
		}

		b, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		// Create tar file
		name := filepath.Base(f.Name)
		dir := filepath.Dir(f.Name)

		hdr := &tar.Header{
			Name: name,
			Size: int64(len(b)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatal(err)
		}

		if _, err := tw.Write(b); err != nil {
			log.Fatal(err)
		}

		// Close zip file and tar writer
		if err = file.Close(); err != nil {
			return err
		}

		if err = tw.Close(); err != nil {
			return err
		}

		err = d.CopyToTar(filepath.Join(serverDirectoryPath, dir), &data)
		if err != nil {
			fmt.Printf("%T", err)

			return err
		}
	}

	return nil
}

func TakeBackup(d *docker.DockerClient, out, command io.Writer, logs *bufio.Reader) error {
	runCommand := func(cmd string) {
		_, err := command.Write([]byte(cmd + "\n"))
		if err != nil {
			log.Fatalf("running command `%s`: %s", cmd, err)
		}

		// Read command echo to discard it
		if _, err := logs.ReadString('\n'); err != nil {
			log.Fatalf("retrieving echo for command `%s`: %s", cmd, err)
		}
	}

	readLine := func() string {
		res, err := logs.ReadString('\n')
		if err != nil {
			log.Fatalf("reading logs: %s", err)
		}

		return res
	}

	// save hold
	runCommand("save hold")

	saveHoldResponse := readLine()

	switch strings.TrimSpace(saveHoldResponse) {
	case "Saving...":
	case "The command is already running":
		break
	default:
		return fmt.Errorf("unexpected response to `save hold`: '%s'", saveHoldResponse)
	}

	// Query until ready for backup
	for i := 0; i < saveQueryRetries; i++ {
		time.Sleep(saveQueryDelayMS * time.Millisecond)

		// save query
		runCommand("save query")

		saveQueryResponseOne := readLine()

		// Ready for backup
		if strings.HasPrefix(saveQueryResponseOne, "Data saved. Files are now ready to be copied.") {
			saveQueryResponseTwo := readLine()

			// files needed by mc server
			files := strings.Split(saveQueryResponseTwo, ", ")
			for i, f := range files {
				files[i] = filepath.Join(worldsDirectoryName, strings.Split(f, ":")[0])
			}

			// files needed by craft
			files = append(files, craftFiles...)

			if err := copyBackupFiles(d, files, out); err != nil {
				return fmt.Errorf("copying files from container to disk: %s", err)
			}

			// save resume
			runCommand("save resume")

			saveResumeResponse := readLine()

			if strings.TrimSpace(saveResumeResponse) != "Changes to the level are resumed." {
				return fmt.Errorf("unexpected response to `save resume`: '%s'", saveResumeResponse)
			}

			return nil
		}
	}

	return fmt.Errorf("exceeded %d retries of the 'save query' command", saveQueryRetries)
}

func copyBackupFiles(d *docker.DockerClient, filePaths []string, out io.Writer) error {
	// Write zip data to out file
	zw := zip.NewWriter(out)

	for _, p := range filePaths {
		tr, err := d.CopyFromTar(filepath.Join(serverDirectoryPath, p))
		if err != nil {
			return err
		}

		for {
			// Next file or end of archive
			_, err := tr.Next()
			if err == io.EOF {
				break
			}

			if err != nil {
				log.Fatal(err)
			}

			// Read from tar archive
			b, err := ioutil.ReadAll(tr)
			if err != nil {
				return err
			}

			// Create file in zip archive
			f, err := zw.Create(p)
			if err != nil {
				log.Fatal(err)
			}

			// Write file to zip archive
			_, err = f.Write(b)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	err := zw.Close()
	if err != nil {
		return err
	}

	return nil
}

func NewBackupTimeStamp(d *docker.DockerClient) string {
	return fmt.Sprintf("%s_%s",
		d.ContainerName,
		time.Now().Format(BackupFilenameTimeLayout),
	)
}

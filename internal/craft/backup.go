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
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
)

const (
	serverDirectoryPath      = "/bedrock"          // The directory where the mc server files are unpacked
	worldsDirectoryName      = "worlds"            // The directory where worlds are stored
	serverPropertiesFileName = "server.properties" // Name of the server properties (configuration) file

	backupFilenameTimeLayout = "02-01-2006_15-04" // The format of the file timestamp for the Go time package formatter

)

// files backed up for Craft
var craftFiles = []string{
	serverPropertiesFileName, // server.properties
}

func SaveBackup(d *DockerClient) error {
	dirPath := d.backupDirectory()
	fileName := fmt.Sprintf("%s.zip", d.newBackupTimeStamp())

	// Create the directory if it doesn't exist
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			return err
		}
	}

	// Create the file
	f, err := os.Create(path.Join(dirPath, fileName))
	if err != nil {
		return err
	}

	c, err := d.CommandWriter()
	if err != nil {
		return err
	}

	l, err := d.LogReader(0)
	if err != nil {
		return err
	}

	// Copy server files and write as zip data
	err = d.takeBackup(f, c, l)
	if err != nil {
		return err
	}

	return nil
}

func RestoreLatestBackup(d *DockerClient) error {
	backupName, _, err := LatestServerBackup(d.ContainerName)
	if err != nil {
		return err
	}

	// Open backup zip
	zr, err := zip.OpenReader(filepath.Join(d.backupDirectory(), backupName))
	if err != nil {
		return err
	}

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

		err = d.copyToTar(filepath.Join(serverDirectoryPath, dir), &data)
		if err != nil {
			fmt.Printf("%T", err)

			return err
		}
	}

	return nil
}

func (d *DockerClient) takeBackup(out, command io.Writer, logs *bufio.Reader) error {
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

			if err := d.copyBackupFiles(files, out); err != nil {
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

func (d *DockerClient) copyBackupFiles(filePaths []string, out io.Writer) error {
	// Write zip data to out file
	zw := zip.NewWriter(out)

	for _, p := range filePaths {
		tr, err := d.copyFromTar(filepath.Join(serverDirectoryPath, p))
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

func (d *DockerClient) backupDirectory() string {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalf("getting home directory: %s", err)
	}

	backupDir := path.Join(home, backupDirName, d.ContainerName)

	// Create directory if it doesn't exist
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		err = os.MkdirAll(backupDir, 0755)
		if err != nil {
			log.Fatalf("checking backup directory exists: %s", err)
		}
	}

	return backupDir
}

func (d *DockerClient) newBackupTimeStamp() string {
	return fmt.Sprintf("%s_%s",
		d.ContainerName,
		time.Now().Format(backupFilenameTimeLayout),
	)
}

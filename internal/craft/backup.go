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
)

const (
	serverDirectoryPath      = "/bedrock"          // The directory where the mc server files are unpacked
	worldsDirectoryName      = "worlds"            // The directory where worlds are stored
	serverPropertiesFileName = "server.properties" // Name of the server properties (configuration) file

	saveQueryRetries = 100 // The number of times save query can run without the expected response
	saveQueryDelayMS = 100 // The delay between save query retries, in milliseconds
)

// files backed up for Craft
var craftFiles = []string{
	serverPropertiesFileName, // server.properties
}

func backupFilePaths(saveQueryResponseFiles string) []string {
	// Files needed by mc server
	files := strings.Split(saveQueryResponseFiles, ", ")
	for i, f := range files {
		files[i] = filepath.Join(worldsDirectoryName, strings.Split(f, ":")[0])
	}

	// Files needed by craft
	return append(files, craftFiles...)
}

func takeBackup(out, command io.Writer, logs *bufio.Reader, copyFromFunc func(string) (*tar.Reader, error)) error {
	// $ save hold
	runCommand("save hold", command, logs)

	saveHoldResponse := readLine(logs)
	if strings.TrimSpace(saveHoldResponse) != "Saving..." {
		return fmt.Errorf("unexpected response to `save hold`: '%s'", saveHoldResponse)
	}

	// Query until ready for backups
	for i := 0; i < saveQueryRetries; i++ {
		time.Sleep(saveQueryDelayMS * time.Millisecond)

		// $ save query
		runCommand("save query", command, logs)

		saveQueryResponseOne := readLine(logs)

		// Ready for backup
		if strings.HasPrefix(saveQueryResponseOne, "Data saved. Files are now ready to be copied.") {
			// Write zip data to out file
			zw := zip.NewWriter(out)

			saveQueryResponseFiles := readLine(logs)
			for _, p := range backupFilePaths(saveQueryResponseFiles) {
				tr, err := copyFromFunc(filepath.Join(serverDirectoryPath, p))
				if err != nil {
					return err
				}

				err = addTarToZip(p, tr, zw)
				if err != nil {
					return fmt.Errorf("copying file from server to zip: %s", err)
				}
			}

			err := zw.Close()
			if err != nil {
				return err
			}

			// $ save resume
			runCommand("save resume", command, logs)

			saveResumeResponse := readLine(logs)
			if strings.TrimSpace(saveResumeResponse) != "Changes to the level are resumed." {
				return fmt.Errorf("unexpected response to `save resume`: '%s'", saveResumeResponse)
			}

			return nil
		}
	}

	return fmt.Errorf("exceeded %d retries of the 'save query' command", saveQueryRetries)
}

func runCommand(cmd string, cli io.Writer, logs *bufio.Reader) {
	_, err := cli.Write([]byte(cmd + "\n"))
	if err != nil {
		log.Fatalf("running command `%s`: %s", cmd, err)
	}

	// Read command echo to discard it
	if _, err := logs.ReadString('\n'); err != nil {
		log.Fatalf("retrieving echo for command `%s`: %s", cmd, err)
	}
}

func readLine(logs *bufio.Reader) string {
	res, err := logs.ReadString('\n')
	if err != nil {
		log.Fatalf("reading logs: %s", err)
	}

	return res
}

func addTarToZip(path string, tr *tar.Reader, zw *zip.Writer) error {
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
		f, err := zw.Create(path)
		if err != nil {
			log.Fatal(err)
		}

		// Write file to zip archive
		_, err = f.Write(b)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func restoreBackup(zr *zip.ReadCloser, copyToFunc func(string, *bytes.Buffer) error) error {
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

		err = copyToFunc(filepath.Join(serverDirectoryPath, dir), &data)
		if err != nil {
			fmt.Printf("%T", err)

			return err
		}
	}

	return nil
}

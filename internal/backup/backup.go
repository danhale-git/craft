package backup

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

	saveQueryRetries = 100 // The number of times save query can run without the expected response
	saveQueryDelayMS = 100 // The delay between save query retries, in milliseconds

	backupDirName      = "craft_backups"    // Name of the local directory where backups are stored
	FileNameTimeLayout = "02-01-2006_15-04" // The format of the file timestamp for the Go time package formatter

)

// serverFiles is a collection of files needed by craft to return the server to its previous state.
var serverFiles = []string{
	serverPropertiesFileName, // server.properties
}

// Directory returns the path to the directory where backup files are saved.
func Directory() string {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalf("getting home directory: %s", err)
	}

	backupDir := path.Join(home, backupDirName)

	// Create directory if it doesn't exist
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		err = os.MkdirAll(backupDir, 0755)
		if err != nil {
			log.Fatalf("checking backup directory exists: %s", err)
		}
	}

	return backupDir
}

// LatestFile returns the name and backup time of the latest backup file from a slice of standard FileInfos.
func LatestFile(serverName string) (string, *time.Time, error) {
	backupDir := Directory()
	infos, err := ioutil.ReadDir(path.Join(backupDir, serverName))

	if err != nil {
		return "", nil, fmt.Errorf("reading directory '%s': %s", backupDir, err)
	}

	var mostRecentTime time.Time

	var mostRecentFileName string

	for _, f := range infos {
		name := f.Name()

		prefix := fmt.Sprintf("%s_", serverName)
		if strings.HasPrefix(name, prefix) {
			backupTime := strings.Replace(name, prefix, "", 1)
			backupTime = strings.Split(backupTime, ".")[0]

			t, err := time.Parse(FileNameTimeLayout, backupTime)
			if err != nil {
				return "", nil, fmt.Errorf("parsing time from file name '%s': %s", name, err)
			}

			if t.After(mostRecentTime) {
				mostRecentTime = t
				mostRecentFileName = name
			}
		}
	}

	return mostRecentFileName, &mostRecentTime, nil
}

// Restore reads from the given zip.ReadCloser, copying each of the files to the directory containing the server
// files.
func Restore(zr *zip.Reader, copyToFunc func(string, *bytes.Buffer) error) error {
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
			return err
		}

		if _, err := tw.Write(b); err != nil {
			return err
		}

		// Close zipped file and tar writer
		if err = file.Close(); err != nil {
			return err
		}

		if err = tw.Close(); err != nil {
			return err
		}

		err = copyToFunc(filepath.Join(serverDirectoryPath, dir), &data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Copy runs the set of queries described in the bedrock server documentation for taking a backup without server
// interruption. All files needed for server and world persistence are copied from the server to a local zip file. The
// paths in the zip file extend to the server directory root.
func Copy(out, command io.Writer, logs *bufio.Reader, copyFunc func(string) (*tar.Reader, error)) error {
	// `save hold`
	runCommand("save hold", command, logs)

	saveHoldResponse := readLine(logs)
	if strings.TrimSpace(saveHoldResponse) != "Saving..." {
		return fmt.Errorf("unexpected response to `save hold`: '%s'", saveHoldResponse)
	}

	// Query until ready for backups
	for i := 0; i < saveQueryRetries; i++ {
		time.Sleep(saveQueryDelayMS * time.Millisecond)

		// `save query`
		runCommand("save query", command, logs)

		saveQueryResponse := readLine(logs)

		// Ready for backup
		if strings.HasPrefix(saveQueryResponse, "Data saved. Files are now ready to be copied.") {
			// Write zip data to out file
			zw := zip.NewWriter(out)

			worldFilesString := readLine(logs)
			// Files needed by mc server
			worldFiles := strings.Split(worldFilesString, ", ")
			for i, f := range worldFiles {
				worldFiles[i] = filepath.Join(worldsDirectoryName, strings.Split(f, ":")[0])
			}

			for _, p := range append(worldFiles, serverFiles...) {
				tr, err := copyFunc(filepath.Join(serverDirectoryPath, p))
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

			// `save resume`
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
		log.Fatalf("backup.go: running command `%s`: %s", cmd, err)
	}

	// Read command echo to discard it
	if _, err := logs.ReadString('\n'); err != nil {
		log.Fatalf("backup.go: retrieving echo for command `%s`: %s", cmd, err)
	}
}

func readLine(logs *bufio.Reader) string {
	res, err := logs.ReadString('\n')
	if err != nil {
		log.Fatalf("backup.go: reading logs: %s", err)
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

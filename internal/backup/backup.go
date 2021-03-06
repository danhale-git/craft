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
	"path/filepath"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/server"
)

const (
	saveQueryRetries = 100 // The number of times save query can run without the expected response
	saveQueryDelayMS = 100 // The delay between save query retries, in milliseconds

	FileNameTimeLayout = "02-01-2006_15-04" // The format of the file timestamp for the Go time package formatter
)

// FileTime returns the time.Time the backup was taken, given the file name.
func FileTime(name string) (time.Time, error) {
	split := strings.SplitN(name, "_", 2)
	if len(split) < 2 { //nolint:gomnd
		return time.Time{}, fmt.Errorf("invalid file name: '%s'", name)
	}

	backupTime := strings.Split(split[1], ".")[0]

	t, err := time.Parse(FileNameTimeLayout, backupTime)
	if err != nil {
		if _, ok := err.(*time.ParseError); ok {
			return time.Time{}, err
		}

		panic(err)
	}

	return t, nil
}

// Restore reads from the given zip.ReadCloser, copying each of the files to the directory containing the server
// files.
func Restore(zr *zip.Reader, copyToFunc func(string, *bytes.Buffer) error) error {
	for _, f := range zr.File {
		if err := restoreFile(f, server.Directory, copyToFunc); err != nil {
			return fmt.Errorf("restoring %s: %s", f.Name, err)
		}
	}

	return nil
}

// Restore reads from the given zip.Reader, copying each of the files to the default world directory.
func RestoreMCWorld(zr *zip.Reader, copyToFunc func(string, *bytes.Buffer) error) error {
	for _, f := range zr.File {
		if err := restoreFile(f, server.FullPaths.DefaultWorld, copyToFunc); err != nil {
			return fmt.Errorf("restoring %s: %s", f.Name, err)
		}
	}

	return nil
}

func restoreFile(f *zip.File, dest string, copyToFunc func(string, *bytes.Buffer) error) error {
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

	return copyToFunc(filepath.Join(dest, dir), &data)
}

// Copy runs the set of queries described in the bedrock server documentation for taking a backup without server
// interruption. All files needed for server and world persistence are copied from the server to a local zip file. The
// paths in the zip file extend to the server directory root.
func SaveHoldQuery(command io.Writer, logs *bufio.Reader) ([]string, error) {
	// `save hold`
	runCommand("save hold", command, logs)

	saveHoldResponse := readLine(logs)
	if strings.TrimSpace(saveHoldResponse) != "Saving..." {
		return nil, fmt.Errorf("unexpected response to `save hold`: '%s'", saveHoldResponse)
	}

	// Query until ready for backups
	for i := 0; i < saveQueryRetries; i++ {
		time.Sleep(saveQueryDelayMS * time.Millisecond)

		// `save query`
		runCommand("save query", command, logs)

		saveQueryResponse := readLine(logs)

		// Ready for backup
		if strings.HasPrefix(saveQueryResponse, "Data saved. Files are now ready to be copied.") {
			worldFilesString := readLine(logs)

			worldFiles := strings.Split(worldFilesString, ", ")
			for i, f := range worldFiles {
				worldFiles[i] = filepath.Join(
					server.LocalPaths.Worlds,
					strings.Split(f, ":")[0],
				)
			}

			return worldFiles, nil
		}
	}

	return nil, fmt.Errorf("exceeded %d retries of the 'save query' command", saveQueryRetries)
}

func SaveResume(command io.Writer, logs *bufio.Reader) error {
	// `save resume`
	runCommand("save resume", command, logs)

	saveResumeResponse := readLine(logs)
	if strings.TrimSpace(saveResumeResponse) != "Changes to the level are resumed." {
		return fmt.Errorf("unexpected response to `save resume`: '%s'", saveResumeResponse)
	}

	return nil
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

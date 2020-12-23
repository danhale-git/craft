package craft

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/files"
)

// type Backup struct { }

// Backup runs the save hold/query/resume command sequence and saves a .mcworld file snapshot to the given local path.
func Backup(d *DockerClient, destPath string) error {
	logs, err := d.LogReader(0)

	saveHold, err := commandResponse(d, "save hold", logs)
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

		saveQuery, err := commandResponse(d, "save query", logs)
		if err != nil {
			return err
		}

		// Ready for backup
		if strings.HasPrefix(saveQuery, "Data saved. Files are now ready to be copied.") {
			err = backupServer(d, destPath, logs)
			if err != nil {
				return fmt.Errorf("backing up server: %s", err)
			}

			break
		}
	}

	// save resume
	saveResume, err := commandResponse(d, "save resume", logs)
	if err != nil {
		return err
	}

	if strings.TrimSpace(saveResume) != "Changes to the level are resumed." {
		return fmt.Errorf("unexpected response to `save resume`: '%s'", saveResume)
	}

	return nil
}

func backupServer(d *DockerClient, destPath string, logs *bufio.Reader) error {
	backupName := fmt.Sprintf("%s_%s",
		d.containerName,
		time.Now().Format(backupFilenameTimeLayout),
	)

	serverBackup := files.Archive{}

	// Back up world
	worldArchive, err := copyWorldFromContainer(d)
	if err != nil {
		return fmt.Errorf("copying world data from container: %s", err)
	}

	wz, err := worldArchive.Zip()
	if err != nil {
		return err
	}

	serverBackup.AddFile(files.File{
		Name: fmt.Sprintf("%s.mcworld", backupName),
		Body: wz.Bytes(),
	})

	// Back up settings
	serverPropertiesArchive, err := copyServerPropertiesFromContainer(d)
	if err != nil {
		return err
	}

	serverBackup.AddFile(files.File{
		Name: serverPropertiesFileName,
		Body: serverPropertiesArchive.Files[0].Body,
	})

	// Save to disk
	err = serverBackup.SaveZip(path.Join(destPath, d.containerName), fmt.Sprintf("%s.zip", backupName))
	if err != nil {
		return fmt.Errorf("saving server backup: %s", err)
	}

	// A second line is returned with a list of files, read it to discard it.
	if _, err := logs.ReadString('\n'); err != nil {
		return fmt.Errorf("reading 'save query' file list response: %s", err)
	}

	return nil
}

func copyWorldFromContainer(d *DockerClient) (*files.Archive, error) {
	// Copy the world directory and it's contents from the container
	a, err := d.copyFrom(worldDirectory)
	if err != nil {
		return nil, err
	}

	// Remove 'Bedrock level' directory
	newFiles := make([]files.File, 0)

	for _, f := range a.Files {
		f.Name = strings.Replace(f.Name, "Bedrock level/", "", 1)

		// Skip the file representing the 'Bedrock level' directory
		if len(strings.TrimSpace(f.Name)) == 0 {
			continue
		}

		newFiles = append(newFiles, f)
	}

	a.Files = newFiles

	return a, nil
}

func copyServerPropertiesFromContainer(d *DockerClient) (*files.Archive, error) {
	a, err := d.copyFrom(path.Join(mcDirectory, serverPropertiesFileName))
	if err != nil {
		return nil, fmt.Errorf("copying '%s' from container path %s: %s", serverPropertiesFileName, mcDirectory, err)
	}

	return a, nil
}

func commandResponse(d *DockerClient, cmd string, logs *bufio.Reader) (string, error) {
	err := command(d, cmd)
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

func command(d *DockerClient, cmd string) error {
	return d.Command(strings.Split(cmd, " "))
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

			w, err := files.NewArchiveFromZip(z)
			if err != nil {
				return err
			}

			err = c.copyTo(worldDirectory, w)
			if err != nil {
				return err
			}
		} else {
			// Other files are copied to the directory containing the mc server executable
			a := files.Archive{}

			a.AddFile(files.File{
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

// // // //

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

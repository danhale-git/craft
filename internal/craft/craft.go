package craft

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/mitchellh/go-homedir"
)

// Run the bedrock_server executable
const (
	backupDirName = "craft_backups" // Name of the local directory where backups are stored

	RunMCCommand = "cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server"
)

// RunServer runs RunMCCommand on the given docker client.
func RunServer(d *docker.DockerClient) error {
	return d.Command(strings.Split(RunMCCommand, " "))
}

// BackupServerNames returns a slice with the names of all backed up servers.
func BackupServerNames() ([]string, error) {
	backupDir := backupDirectory()
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

// LatestServerBackup returns the path and backup time of the latest backup for the given server.
func LatestServerBackup(serverName string) (string, *time.Time, error) {
	backupDir := backupDirectory()
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

			t, err := time.Parse(docker.BackupFilenameTimeLayout, backupTime)
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

// SaveBackup takes a new backup and saves it to the default backup directory.
func SaveBackup(d *docker.DockerClient) error {
	backupPath := filepath.Join(backupDirectory(), d.ContainerName)
	fileName := fmt.Sprintf("%s.zip", d.NewBackupTimeStamp())

	// Create the directory if it doesn't exist
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		err = os.MkdirAll(backupPath, 0755)
		if err != nil {
			return err
		}
	}

	// Create the file
	f, err := os.Create(path.Join(backupPath, fileName))
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
	err = d.TakeBackup(f, c, l)
	if err != nil {
		return err
	}

	return nil
}

// RestoreBackup finds the latest backup and restores it to the server.
func RestoreLatestBackup(d *docker.DockerClient) error {
	backupPath := filepath.Join(backupDirectory(), d.ContainerName)
	backupName, _, err := LatestServerBackup(d.ContainerName)
	if err != nil {
		return err
	}

	// Open backup zip
	zr, err := zip.OpenReader(filepath.Join(backupPath, backupName))
	if err != nil {
		return err
	}

	return d.RestoreBackup(zr)
}

func backupDirectory() string {
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

/*if f.Name == docker.serverPropertiesFileName {
	updated, err := setProperty(f.Body, field, value)
	if err != nil {
		return fmt.Errorf("updating file data: %s", err)
	}

	f.Body = updated

	return nil
}*/
func setProperty(data []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	alteredLines := make([]byte, 0)

	changed := false

	// Read file data line by line and amend the chosen key's value
	for _, line := range lines {
		l := strings.TrimSpace(line)

		property := strings.Split(l, "=")

		// Empty line, comment or other property
		if len(l) == 0 || string(l[0]) == "#" || property[0] != key {
			alteredLines = append(alteredLines, []byte(fmt.Sprintf("%s\n", line))...)
			continue
		}

		// Found property, alter value
		alteredLines = append(alteredLines, []byte(fmt.Sprintf("%s=%s\n", key, value))...)
		changed = true
	}

	if !changed {
		return nil, fmt.Errorf("no key was found with name '%s'", key)
	}

	return alteredLines, nil
}

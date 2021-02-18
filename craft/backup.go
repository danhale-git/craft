package craft

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/logger"

	server2 "github.com/danhale-git/craft/internal/server"

	"github.com/mitchellh/go-homedir"

	"github.com/danhale-git/craft/internal/backup"

	"github.com/danhale-git/craft/internal/docker"
)

const (
	backupDirName = "craft_backups" // Name of the local directory where backups are stored
)

// serverFiles is a collection of files needed by craft to return the server to its previous state.
var serverFiles = []string{
	server2.FileNames.ServerProperties, // server.properties
}

func TrimBackups(name string, keep int, skip bool) ([]string, error) {
	deleted := make([]string, 0)

	backups := listBackups(name)
	if keep >= len(backups) {
		// No backups need to be deleted
		return nil, nil
	}

	remove := backups[:len(backups)-keep]
	d := filepath.Join(backupDirectory(), name)

	// Check before removing files
	if !skip {
		fmt.Println()

		for _, f := range remove {
			fmt.Println(f.Name())
		}

		fmt.Print("Remove these files? (y/n): ")

		text, _ := bufio.NewReader(os.Stdin).ReadString('\n')

		if strings.TrimSpace(text) != "y" {
			fmt.Println("cancelled")
			return nil, nil
		}
	}

	for _, f := range remove {
		if err := os.Remove(filepath.Join(d, f.Name())); err != nil {
			logger.Error.Printf("removing file: %s", err)
			continue
		}

		deleted = append(deleted, f.Name())
	}

	return deleted, nil
}

func listBackups(server string) []os.FileInfo {
	infos := make([]os.FileInfo, 0)
	d := filepath.Join(backupDirectory(), server)

	err := filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("Error getting backup file: %s", err)
		}
		infos = append(infos, info)
		return nil
	})
	if err != nil {
		panic(err)
	}

	return backup.SortFilesByDate(infos)
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

func latestBackupFileName(serverName string) os.FileInfo {
	backupDir := backupDirectory()

	infos, err := ioutil.ReadDir(path.Join(backupDir, serverName))
	if err != nil {
		panic(err)
	}

	return backup.SortFilesByDate(infos)[len(infos)-1]
}

func CopyBackup(d *docker.Container) (string, error) {
	backupPath := filepath.Join(backupDirectory(), d.ContainerName)
	fileName := fmt.Sprintf("%s_%s.zip", d.ContainerName, time.Now().Format(backup.FileNameTimeLayout))
	backupFilePath := path.Join(backupPath, fileName)

	// Create the directory if it doesn't exist
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		err = os.MkdirAll(backupPath, 0755)
		if err != nil {
			return "", err
		}
	}

	// Create the file
	f, err := os.Create(backupFilePath)
	if err != nil {
		return "", err
	}

	// Write to server CLI
	c, err := d.CommandWriter()
	if err != nil {
		return "", err
	}

	// Read from server CLI
	l, err := d.LogReader(0)
	if err != nil {
		return "", err
	}

	// Copy server files and write as zip data
	if err = backup.Copy(f, c, l, d.CopyFrom, serverFiles); err != nil {
		if err := f.Close(); err != nil {
			logger.Error.Printf("failed to close backup file after error")
		}

		// Clean up bad backup file
		if err := os.Remove(backupFilePath); err != nil {
			logger.Error.Printf("failed to remove backup file after error: %s", err)
		}

		return "", err
	}

	return fileName, nil
}

// RestoreLatestBackup finds the latest backup and restores it to the server.
func restoreBackup(d *docker.Container, backupName string) error {
	backupPath := filepath.Join(backupDirectory(), d.ContainerName)

	// Open backup zip
	zr, err := zip.OpenReader(filepath.Join(backupPath, backupName))
	if err != nil {
		return err
	}

	if err = backup.Restore(&zr.Reader, d.CopyTo); err != nil {
		return err
	}

	if err = zr.Close(); err != nil {
		return fmt.Errorf("closing zip: %s", err)
	}

	return nil
}

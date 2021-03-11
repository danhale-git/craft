package craft

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/server"

	"github.com/mitchellh/go-homedir"

	"github.com/danhale-git/craft/internal/backup"

	"github.com/danhale-git/craft/internal/docker"
)

const (
	backupDirName = "craft_backups" // Name of the local directory where backups are stored
)

// serverFiles returns the paths to all files in the server directory which are not part of the world backup. World
// files are retrieved as part of the server's built in backup function. Other files required to persist the server
// may also be included here.
func serverFiles() []string {
	return []string{
		server.LocalPaths.ServerProperties, // server.properties
	}
}

func TrimBackups(name string, keep int, skip bool) ([]string, error) {
	deleted := make([]string, 0)

	backups := backupFiles(name)
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

// BackupExists returns true if a backed up server with the given server name exists.
func BackupExists(name string) bool {
	for _, b := range backupServerNames() {
		if name == b {
			return true
		}
	}

	return false
}

func latestBackupFile(name string) os.FileInfo {
	backups := backupFiles(name)
	return backups[len(backups)-1]
}

func backupFiles(server string) []os.FileInfo {
	infos := make([]os.FileInfo, 0)
	d := filepath.Join(backupDirectory(), server)

	err := filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error.Printf("Error getting backup file: %s", err)
		}
		infos = append(infos, info)
		return nil
	})
	if err != nil {
		panic(err)
	}

	return backup.SortFilesByDate(infos)
}

// backupServerNames returns a slice with the names of all backed up servers.
func backupServerNames() []string {
	backupDir := backupDirectory()
	infos, err := ioutil.ReadDir(backupDir)

	if err != nil {
		logger.Panicf("reading directory '%s': %s", backupDir, err)
	}

	names := make([]string, 0)

	for _, f := range infos {
		if !f.IsDir() {
			continue
		}

		names = append(names, f.Name())
	}

	return names
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

func CopyBackup(c *docker.Container) (string, error) {
	backupPath := filepath.Join(backupDirectory(), c.ContainerName)
	fileName := fmt.Sprintf("%s_%s.zip", c.ContainerName, time.Now().Format(backup.FileNameTimeLayout))
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
	cmd, err := c.CommandWriter()
	if err != nil {
		return "", err
	}

	// Read from server CLI
	logs, err := c.LogReader(0)
	if err != nil {
		return "", err
	}

	paths, err := backup.SaveHoldQuery(cmd, logs)
	if err != nil {
		return "", err
	}

	paths = append(paths, serverFiles()...)

	// Copy server files and write as zip data
	if err = copyFiles(c, f, paths); err != nil {
		if err := f.Close(); err != nil {
			logger.Error.Printf("failed to close backup file after error")
		}

		// Clean up bad backup file
		if err := os.Remove(backupFilePath); err != nil {
			logger.Error.Printf("failed to remove backup file after error: %s", err)
		}

		return "", err
	}

	if err := backup.SaveResume(cmd, logs); err != nil {
		logger.Error.Printf("error when running `save resume` (server may be in a bad state)")
	}

	return fileName, nil
}

func copyFiles(c *docker.Container, f io.Writer, paths []string) error {
	// Write zip data to out file
	zw := zip.NewWriter(f)

	for _, p := range paths {
		tr, err := c.CopyFrom(filepath.Join(server.Directory, p))
		if err != nil {
			return err
		}

		err = addTarToZip(p, tr, zw)
		if err != nil {
			return fmt.Errorf("copying file from server to zip: %s", err)
		}
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("closing zip writer: %s", err)
	}

	return nil
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

package craft

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/files"

	"github.com/danhale-git/craft/mcworld"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/server"

	"github.com/mitchellh/go-homedir"

	"github.com/danhale-git/craft/internal/backup"
)

const (
	backupDirName = "craft_backups" // Name of the local directory where backups are stored
)

// serverFiles returns the paths to all files in the server directory which are not part of the world backup. World
// files are retrieved as part of the server's built in backup function. Other files required to persist the server
// may also be included here.
func serverFiles() []string {
	return []string{
		files.LocalPaths.ServerProperties, // server.properties
	}
}

// CopyBackup copies the server world files to the server backup directory.
func CopyBackup(s *server.Server) (string, error) {
	backupPath := filepath.Join(backupDirectory(), s.ContainerName)
	fileName := fmt.Sprintf("%s_%s.zip", s.ContainerName, time.Now().Format(backup.FileNameTimeLayout))
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
	cmd, err := s.CommandWriter()
	if err != nil {
		return "", err
	}

	// Read from server CLI
	logs, err := s.LogReader(0)
	if err != nil {
		return "", err
	}

	paths, err := backup.SaveHoldQuery(cmd, logs)
	if err != nil {
		return "", err
	}

	// Prepend path from server directory to world directory
	for i, p := range paths {
		paths[i] = filepath.Join(files.LocalPaths.Worlds, p)
	}

	paths = append(paths, serverFiles()...)

	// Copy server files and write as zip data
	if err = copyFiles(s, f, files.Directory, paths); err != nil {
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

// ExportMCWorld copies the server's current world files to a zipped .mcworld file at the given destination which must
// be a directory.
func ExportMCWorld(s *server.Server, dest string) error {
	if dest == "" {
		dest = backupDirectory()
	}

	dir, err := os.Stat(dest)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dest, fmt.Sprintf("%s.mcworld", s.ContainerName))

	if !dir.Mode().IsDir() {
		return fmt.Errorf("'%s' is not a directory", dest)
	}

	// Create the file
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	// Write to server CLI
	cmd, err := s.CommandWriter()
	if err != nil {
		return err
	}

	// Read from server CLI
	logs, err := s.LogReader(0)
	if err != nil {
		return err
	}

	paths, err := backup.SaveHoldQuery(cmd, logs)
	if err != nil {
		return err
	}

	// Prepend path from server directory to world directory
	for i, p := range paths {
		paths[i] = filepath.Join(strings.Split(p, "/")[1:]...)
	}

	// Copy server files and write as zip data
	if err = copyFiles(s, f, files.FullPaths.DefaultWorld, paths); err != nil {
		if err := f.Close(); err != nil {
			logger.Error.Printf("failed to close backup file after error")
		}

		// Clean up bad backup file
		if err := os.Remove(dest); err != nil {
			logger.Error.Printf("failed to remove backup file after error: %s", err)
		}

		return err
	}

	if err := backup.SaveResume(cmd, logs); err != nil {
		logger.Error.Printf(`error when running save resume (server may be in a bad state - try running 'craft
cmd <server> save resume')`)
	}

	mcw := mcworld.MCWorld{Path: filePath}
	if err := mcw.Check(); err != nil {
		return fmt.Errorf("invalid world file after exporting: %s", err)
	}

	return nil
}

func copyFiles(s *server.Server, f io.Writer, containerPrefix string, paths []string) error {
	// Write zip data to out file
	zw := zip.NewWriter(f)

	for _, p := range paths {
		containerPath := filepath.Join(containerPrefix, p)

		data, _, err := s.CopyFromContainer(
			context.Background(),
			s.ContainerID,
			containerPath,
		)
		if err != nil {
			return fmt.Errorf("copying data from server at '%s': %s", containerPath, err)
		}

		tr := tar.NewReader(data)

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

func TrimBackups(name string, keep int, skip bool) ([]string, error) {
	deleted := make([]string, 0)

	backups := serverBackups(name)
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

// backupExists returns true if a backed up server with the given server name exists.
func backupExists(name string) bool {
	for _, b := range stoppedServerNames() {
		if name == b && len(serverBackups(name)) > 0 {
			return true
		}
	}

	return false
}

// latestBackupFile returns an os.FileInfo for the most recent backup
func latestBackupFile(name string) (os.FileInfo, error) {
	backups := serverBackups(name)

	switch len(backups) {
	case 0:
		return nil, fmt.Errorf("no backups files found for server '%s'", name)
	case 1:
		return backups[0], nil
	default:
		return backups[len(backups)-1], nil
	}
}

// serverBackups returns a slice of os.FileInfo with each of the backups for the named server, ordered oldest first.
func serverBackups(server string) []os.FileInfo {
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

// stoppedServerNames returns a slice with the names of all backed up servers.
func stoppedServerNames() []string {
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
		logger.Error.Fatalf("getting home directory: %s", err)
	}

	backupDir := filepath.Join(home, backupDirName)

	// Create directory if it doesn't exist
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		err = os.MkdirAll(backupDir, 0755)
		if err != nil {
			logger.Error.Fatalf("checking backup directory exists: %s", err)
		}
	}

	return backupDir
}

func addTarToZip(path string, tr *tar.Reader, zw *zip.Writer) error {
	for {
		// Next file or end of archive
		_, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			logger.Error.Fatal(err)
		}

		// Read from tar archive
		b, err := ioutil.ReadAll(tr)
		if err != nil {
			return err
		}

		// Create file in zip archive
		f, err := zw.Create(path)
		if err != nil {
			logger.Error.Fatal(err)
		}

		// Write file to zip archive
		_, err = f.Write(b)
		if err != nil {
			logger.Error.Fatal(err)
		}
	}

	return nil
}

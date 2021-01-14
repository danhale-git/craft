package cmd

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

	"github.com/mitchellh/go-homedir"

	"github.com/danhale-git/craft/internal/backup"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

const (
	backupDirName      = "craft_backups"    // Name of the local directory where backups are stored
	FileNameTimeLayout = "02-01-2006_15-04" // The format of the file timestamp for the Go time package formatter
)

// backupCmd represents the backup command
func init() {
	backupCmd := &cobra.Command{
		Use:   "backup <server name>",
		Short: "Take a backup",
		Long: `
Save the current world and server.properties to a zip file in the backup directory.
If two backups are taken in the same minute, the second will overwrite the first.
Backups are saved to a default directory under the user's home directory.
The backed up world is usually a few seconds behind the world state at the time of backup.`,
		// Allow only one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		// save the world files to a backup archive
		Run: func(cmd *cobra.Command, args []string) {
			d := docker.NewContainerOrExit(args[0])

			err := copyBackup(d)
			if err != nil {
				log.Fatalf("Error taking backup: %s", err)
			}
		},
	}

	rootCmd.AddCommand(backupCmd)
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

func latestFile(serverName string) (string, *time.Time, error) {
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

func copyBackup(d *docker.Container) error {
	backupPath := filepath.Join(backupDirectory(), d.ContainerName)
	fileName := fmt.Sprintf("%s_%s.zip", d.ContainerName, time.Now().Format(FileNameTimeLayout))

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

	// Write to server CLI
	c, err := d.CommandWriter()
	if err != nil {
		return err
	}

	// Read from server CLI
	l, err := d.LogReader(0)
	if err != nil {
		return err
	}

	// Copy server files and write as zip data
	err = backup.Copy(f, c, l, d.CopyFrom)
	if err != nil {
		return err
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

package cmd

import (
	"archive/zip"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/danhale-git/craft/internal/backup"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
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

func copyBackup(d *docker.Container) error {
	backupPath := filepath.Join(backup.Directory(), d.ContainerName)
	fileName := fmt.Sprintf("%s_%s.zip", d.ContainerName, time.Now().Format(backup.FileNameTimeLayout))

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
	backupPath := filepath.Join(backup.Directory(), d.ContainerName)

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

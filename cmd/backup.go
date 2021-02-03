package cmd

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

	server2 "github.com/danhale-git/craft/internal/server"

	"github.com/mitchellh/go-homedir"

	"github.com/danhale-git/craft/internal/backup"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

const (
	backupDirName = "craft_backups" // Name of the local directory where backups are stored
)

// serverFiles is a collection of files needed by craft to return the server to its previous state.
var serverFiles = []string{
	server2.FileNames.ServerProperties, // server.properties
}

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
			c := docker.NewContainerOrExit(args[0])

			l, err := cmd.Flags().GetBool("list")
			if err != nil {
				panic(err)
			}

			// List backups and exit
			if l {
				backups := listBackups(c.ContainerName)
				for i := len(backups) - 1; i >= 0; i-- {
					fmt.Println(backups[i].Name())
				}
				return
			}

			// Take a new backup
			err = copyBackup(c)
			if err != nil {
				log.Fatalf("Error taking backup: %s", err)
			}

			trim, err := cmd.Flags().GetInt("trim")
			if err != nil {
				panic(err)
			}
			skip, err := cmd.Flags().GetBool("skip-trim-file-removal-check")
			if err != nil {
				panic(err)
			}

			// Delete old backups
			if trim > 0 {
				backups := listBackups(c.ContainerName)
				if trim >= len(backups) {
					fmt.Printf("Only %d backup files exist, trimming %d would remove them all", len(backups), trim)
					return
				}

				remove := backups[:len(backups)-trim]
				d := filepath.Join(backupDirectory(), c.ContainerName)

				// Check before removing files
				if !skip {
					for _, f := range remove {
						fmt.Println(f.Name())
					}

					fmt.Print("Remove the following files? (y/n): ")
					text, _ := bufio.NewReader(os.Stdin).ReadString('\n')

					if strings.TrimSpace(text) != "y" {
						fmt.Println("cancelled")
						return
					}
				}

				fmt.Print("Deleted:")
				for _, f := range remove {
					if err = os.Remove(filepath.Join(d, f.Name())); err != nil {
						fmt.Println()
						log.Fatalf("Error removing file: %s", err)
					}
					fmt.Print(" ", f.Name())
				}
				fmt.Println()
			}
		},
	}

	backupCmd.Flags().IntP("trim", "t", 0,
		"Delete the oldest backup files, leaving the given count of newest files in place.")
	backupCmd.Flags().Bool("skip-trim-file-removal-check", false,
		"Don't prompt the user before removing files. Useful for automating backups.")
	backupCmd.Flags().BoolP("list", "l", false,
		"List backup files and take no other action.")
	rootCmd.AddCommand(backupCmd)
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

func copyBackup(d *docker.Container) error {
	backupPath := filepath.Join(backupDirectory(), d.ContainerName)
	fileName := fmt.Sprintf("%s_%s.zip", d.ContainerName, time.Now().Format(backup.FileNameTimeLayout))
	backupFilePath := path.Join(backupPath, fileName)

	// Create the directory if it doesn't exist
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		err = os.MkdirAll(backupPath, 0755)
		if err != nil {
			return err
		}
	}

	// Create the file
	f, err := os.Create(backupFilePath)
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
	if err = backup.Copy(f, c, l, d.CopyFrom, serverFiles); err != nil {
		// Clean up bad backup file
		if err := os.Remove(backupFilePath); err != nil {
			log.Panicf("failed to remove file after error in backup process: %s", err)
		}

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

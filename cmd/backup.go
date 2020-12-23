package cmd

import (
	"fmt"
	"log"

	"github.com/danhale-git/craft/internal/craft"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
func init() {
	backupCmd := &cobra.Command{
		Use:   "backup <server name>",
		Short: "Manual local back up of craft servers and settings.",
		Long: "Copy your current world files to a .mcworld export and save to a zip archive along with the current" +
			" server.properties. The zip file name will be the date and time the backup was taken. If two backups" +
			" are taken in the same minute, the second will overwrite the first.",
		// Allow only one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		// Save the world files to a backup archive
		RunE: func(cmd *cobra.Command, args []string) error {
			d := craft.NewDockerClientOrExit(args[0])
			return runBackup(d)
		},
	}

	rootCmd.AddCommand(backupCmd)
}

func runBackup(d *craft.DockerClient) error {
	b, err := craft.NewBackup(d)
	if err != nil {
		log.Fatalf("Error taking backup: %s", err)
	}

	p, err := b.Save()
	if err != nil {
		log.Fatalf("Error saving backup: %s", err)
	}

	fmt.Printf("Backup saved to to %s\n", p)

	return nil
}

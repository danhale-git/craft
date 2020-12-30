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
			d := craft.NewDockerClientOrExit(args[0])

			_, p, err := craft.SaveBackup(d)
			if err != nil {
				log.Fatalf("Error taking backup: %s", err)
			}

			fmt.Printf("Backup saved to to %s\n", p)
		},
	}

	rootCmd.AddCommand(backupCmd)
}

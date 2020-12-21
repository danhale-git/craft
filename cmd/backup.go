package cmd

import (
	"fmt"

	"github.com/danhale-git/craft/internal/server"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use: "backup",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(1, 1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		c := server.GetContainerOrExit(args[0])
		return RunBackup(c)
	},
}

func RunBackup(c *server.Container) error {
	out, err := rootCmd.PersistentFlags().GetString("backup-dir")
	if err != nil {
		return err
	}

	fmt.Printf("Backing up to %s\n", out)

	if err = server.Backup(c, out); err != nil {
		return fmt.Errorf("backing up world: %s", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(backupCmd)
}

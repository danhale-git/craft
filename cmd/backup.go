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
		c, ok := server.ContainerFromName(args[0])
		if !ok {
			return fmt.Errorf("container '%s' does not exist", args[0])
		}

		out, err := cmd.Flags().GetString("out-dir")
		if err != nil {
			return err
		}

		if err = server.Backup(c.ID, c.Names[0], out); err != nil {
			return fmt.Errorf("backing up world: %s", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringP("out-dir", "o", "", "The directory where this backup will be saved as a .mcworld file.")
	_ = backupCmd.MarkFlagRequired("out-dir")
}

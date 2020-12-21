package cmd

import (
	"fmt"

	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use: "list",
	RunE: func(cmd *cobra.Command, args []string) error {
		backupDir, err := rootCmd.PersistentFlags().GetString("backup-dir")
		if err != nil {
			return err
		}

		activeNames, err := craft.ListNames()
		if err != nil {
			return err
		}

		backupNames, err := craft.BackupServerNames(backupDir)
		if err != nil {
			return err
		}

		fmt.Println("Active servers:")
		for _, n := range activeNames {
			fmt.Println(n)
		}
		fmt.Println()

		fmt.Println("Backed up servers:")
		for _, n := range backupNames {
			latest, err := craft.LatestServerBackup(n, backupDir)
			if err != nil {
				return fmt.Errorf("getting latest backup file name: %s", err)
			}
			fmt.Printf("%s\t%s\n", n, latest)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

package cmd

import (
	"log"
	"path"

	"github.com/danhale-git/craft/internal/server"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use: "start",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(1, 1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Find the most recent backup
		backupDir, err := rootCmd.PersistentFlags().GetString("backup-dir")
		if err != nil {
			return err
		}

		mostRecentBackup, err := server.LatestServerBackup(name, backupDir)
		if err != nil {
			log.Fatalf("getting most recent backup file name: %s", err)
		}

		// Create a container for the server
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}

		err = server.Run(port, name)
		if err != nil {
			log.Fatalf("Error running server: %s", err)
		}

		// Get the container ID
		c := server.GetContainerOrExit(name)

		err = server.LoadBackup(c, path.Join(backupDir, name, mostRecentBackup))
		if err != nil {
			log.Fatalf("loading backup file to server: %s", err)
		}

		// Run the bedrock_server process
		err = server.RunServer(c)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// TODO: automatically chose an unused port if not given instead of using default port
	// This TODO and flag is also in run. Consider a better way of managing this. Probably automate it completely.
	startCmd.Flags().IntP("port", "p", 19132, "External port players connect to.")
}

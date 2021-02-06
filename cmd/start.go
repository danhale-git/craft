package cmd

import (
	"log"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

func init() {
	// startCmd represents the start command
	startCmd := &cobra.Command{
		Use:   "start <servers...>",
		Short: "Start a stopped server.",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(1)(cmd, args)
		},
		Run: StartCommand,
	}

	rootCmd.AddCommand(startCmd)

	startCmd.Flags().IntP("port", "p", 0, "External port players connect to.")
}

func StartCommand(cmd *cobra.Command, args []string) {
	for _, name := range args {
		// Create a container for the server
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			panic(err)
		}

		d, err := docker.RunContainer(port, name)
		if err != nil {
			log.Fatalf("Error running server: %s", err)
		}

		f := latestBackupFileName(d.ContainerName)

		err = restoreBackup(d, f.Name())
		if err != nil {
			log.Fatalf("Error loading backup file to server: %s", err)
		}

		if err = runServer(d); err != nil {
			log.Fatalf("Error starting server process: %s", err)
		}
	}
}

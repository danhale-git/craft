package cmd

import (
	"log"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/danhale-git/craft/internal/craft"
	"github.com/spf13/cobra"
)

func init() {
	// stopCmd represents the stop command
	stopCmd := &cobra.Command{
		Use:   "stop <server>",
		Short: "Back up and stop a running server",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			d := docker.NewDockerClientOrExit(args[0])

			noBackup, err := cmd.Flags().GetBool("no-backup")
			if err != nil {
				return err
			}

			// Attempt to back up the server unless instructed otherwise.
			if !noBackup {
				err := craft.SaveBackup(d)
				if err != nil {
					log.Fatalf("Error taking backup: %s", err)
				}
			}

			// Stop the game server process
			err = d.Command([]string{"stop"})
			if err != nil {
				log.Fatalf("Error running 'stop' command: %s", err)
			}

			// Stop the docker container
			return d.Stop()
		},
	}

	rootCmd.AddCommand(stopCmd)

	stopCmd.Flags().Bool("no-backup", false, "Stop the server without backing up first.")
}

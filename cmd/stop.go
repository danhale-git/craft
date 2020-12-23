package cmd

import (
	"fmt"
	"log"

	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use: "stop",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(1, 1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement DockerClient here
		d := craft.NewDockerClientOrExit(args[0])

		c := craft.GetContainerOrExit(args[0])

		noBackup, err := cmd.Flags().GetBool("no-backup")
		if err != nil {
			return err
		}

		// Attempt to back up the server unless instructed otherwise. Abandon stop command if backup fails.
		if !noBackup {
			if err = runBackup(d); err != nil {
				fmt.Printf("error trying to take backup of server %s: %s\n", args[0], err)
				log.Fatalf("Aborting stop command")
			}
		}

		// Stop the game server process
		err = d.Command([]string{"stop"})
		if err != nil {
			log.Fatalf("running 'stop' command: %s", err)
		}

		// Stop the containers
		return c.Stop()
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	stopCmd.Flags().Bool("no-backup", false, "Stop the server without backing up first.")
}

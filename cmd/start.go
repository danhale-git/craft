package cmd

import (
	"log"
	"strings"

	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

func init() {
	// startCmd represents the start command
	startCmd := &cobra.Command{
		Use: "start",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Create a container for the server
			port, err := cmd.Flags().GetInt("port")
			if err != nil {
				return err
			}

			d, err := craft.NewContainer(port, name)
			if err != nil {
				log.Fatalf("Error running server: %s", err)
			}

			err = craft.RestoreLatestBackup(d)
			if err != nil {
				log.Fatalf("loading backup file to server: %s", err)
			}

			// Run the bedrock_server process
			err = d.Command(strings.Split(craft.RunMCCommand, " "))
			if err != nil {
				log.Fatalf("starting mc server process: %s", err)
			}

			return nil
		},
	}

	rootCmd.AddCommand(startCmd)

	// TODO: automatically chose an unused port if not given instead of using default port
	// This TODO and flag is also in run. Consider a better way of managing this. Probably automate it completely.
	startCmd.Flags().IntP("port", "p", 19132, "External port players connect to.")
}

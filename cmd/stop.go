package cmd

import (
	"strings"

	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

// NewStopCmd returns the stop command which takes a backup and stops the server.
func NewStopCmd() *cobra.Command {
	// stopCmd represents the stop command
	stopCmd := &cobra.Command{
		Use:   "stop <servers...>",
		Short: "Back up and stop a running server.",
		Long:  `Back up the server then stop it. If the backup process fails, the server will not be stopped. `,
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			stopped := make([]string, 0)
			c := craft.GetServerOrExit(args[0])

			if _, err := craft.CopyBackup(c); err != nil {
				logger.Error.Fatalf("%s: error while taking backup: %s", c.ContainerName, err)
			}

			if err := craft.Stop(c); err != nil {
				logger.Error.Fatalf("%s: stopping server: %s", c.ContainerName, err)
			}
			stopped = append(stopped, c.ContainerName)

			logger.Info.Println("stopped:", strings.Join(stopped, " "))
		},
	}

	return stopCmd
}

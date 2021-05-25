package cmd

import (
	"strings"

	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

// NewCommandCmd returns the 'command' command which executes a mc server command on the given server.
func NewCommandCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "command <server> <mc command>",
		Aliases: []string{"cmd"},
		Short:   "Run a command on a server",
		Long:    `The first argument is the serer name. The following arguments will be executed in the server CLI.`,
		Example: "craft command myserver give PlayerName stone 1",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(2)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := craft.GetServerOrExit(args[0])

			logs, err := c.LogReader(0)
			if err != nil {
				logger.Error.Fatalf("retrieving log reader: %s", err)
			}

			if err = c.Command(args[1:]); err != nil {
				logger.Error.Fatalf("running command '%s': %s", strings.Join(args[1:], " "), err)
			}

			// Wait for command to return before exiting
			for i := 0; i < 2; i++ {
				response, err := logs.ReadString('\n')
				if err != nil {
					logger.Error.Fatalf("reading response %d from logs: %s", i, err)
				}

				if strings.HasPrefix(response, "Syntax error:") {
					logger.Error.Printf("error reported from server cli: %s", response)
				}
			}
		},
	}
}

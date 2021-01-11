package cmd

import (
	"github.com/danhale-git/craft/internal/docker"
	"github.com/spf13/cobra"
)

// cmdCmd represents the cmd command
func init() {
	cmdCmd := &cobra.Command{
		Use:     "cmd <server> <mc command>",
		Example: "craft cmd myserver give PlayerName stone 1",
		Short:   "Run a command on a server",
		Long: `
The first argument is the serer name.
Any number of following arguments may be provided as a mc server command.`,
		// Accept at 2 or more arguments
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(2, len(args))(cmd, args)
		},
		// Send the given command to the container
		RunE: func(cmd *cobra.Command, args []string) error {
			containerName := args[0]
			command := args[1:]

			d := docker.NewContainerOrExit(containerName)

			err := d.Command(command)
			if err != nil {
				return err
			}

			return nil
		},
	}

	rootCmd.AddCommand(cmdCmd)
}

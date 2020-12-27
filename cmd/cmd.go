package cmd

import (
	"github.com/danhale-git/craft/internal/craft"
	"github.com/spf13/cobra"
)

// cmdCmd represents the cmd command
func init() {
	cmdCmd := &cobra.Command{
		Use:   "cmd <server name> <mc command>",
		Short: "Run the given command on the named craft server.",
		Long: `The first argument is the serer name.
Any number of following arguments may be provided as a mc server command, for example 'kill MyPlayer'.
To kill MyPlayer on the server 'myserver' enter: craft cmd myserver kill MyPlayer.`,
		// Accept at 2 or more arguments
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(2, len(args))(cmd, args)
		},
		// Send the given command to the container
		RunE: func(cmd *cobra.Command, args []string) error {
			containerName := args[0]
			command := args[1:]

			d := craft.NewDockerClientOrExit(containerName)

			err := d.Command(command)
			if err != nil {
				return err
			}

			return nil
		},
	}

	rootCmd.AddCommand(cmdCmd)
}

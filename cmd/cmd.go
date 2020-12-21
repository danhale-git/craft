package cmd

import (
	"github.com/danhale-git/craft/internal/craft"
	"github.com/spf13/cobra"
)

// cmdCmd represents the cmd command
var cmdCmd = &cobra.Command{
	Use: "cmd",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(2, len(args))(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		containerName := args[0]
		command := args[1:]

		c := craft.GetContainerOrExit(containerName)

		err := c.Command(command)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cmdCmd)
}

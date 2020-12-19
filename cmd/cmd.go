package cmd

import (
	"github.com/danhale-git/craft/internal/docker"
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

		c := docker.GetContainerOrExit(containerName)

		err := docker.Command(c.ID, command)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cmdCmd)
}

package cmd

import (
	"fmt"

	"github.com/danhale-git/craft/internal/server"
	"github.com/spf13/cobra"
)

// cmdCmd represents the cmd command
var cmdCmd = &cobra.Command{
	Use: "cmd",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("not enough arguments: 'craft run cmd <servername> <command>`")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		c, ok := server.ContainerFromName(args[0])
		if !ok {
			return fmt.Errorf("container '%s' not found", args[0])
		}

		err := server.Command(c.ID, append(args[1:], "\n"))
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cmdCmd)
}

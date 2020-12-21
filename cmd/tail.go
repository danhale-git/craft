package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/danhale-git/craft/internal/server"

	"github.com/spf13/cobra"
)

// tailCmd represents the tail command
var tailCmd = &cobra.Command{
	Use: "tail",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(1, 1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		c := server.GetContainerOrExit(args[0])

		if _, err := io.Copy(os.Stdout, c.Tail(20)); err != nil {
			return fmt.Errorf("copying server output to stdout: %s", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tailCmd)
}

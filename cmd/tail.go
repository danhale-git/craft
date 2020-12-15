package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/danhale-git/craft/internal/server"

	"github.com/spf13/cobra"
)

// tailCmd represents the tail command
var tailCmd = &cobra.Command{
	Use: "tail",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(1, 2)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		c, ok := server.ContainerFromName(args[0])
		if !ok {
			log.Fatal("container doesn't exist")
		}

		if _, err := io.Copy(os.Stdout, server.Tail(c.ID, 20)); err != nil {
			return fmt.Errorf("copying server output to stdout: %s", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tailCmd)
}

package cmd

import (
	"io"
	"log"
	"os"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

func init() {
	logsCmd := &cobra.Command{
		Use:   "logs <server>",
		Short: "Output server logs",
		// Accept only one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		// Read logs from a container and copy them to stdout
		Run: func(cmd *cobra.Command, args []string) {
			c := docker.NewContainerOrExit(args[0])

			tail, err := cmd.Flags().GetInt("tail")
			if err != nil {
				panic(err)
			}

			logs, err := c.LogReader(tail)
			if err != nil {
				log.Fatalf("Error reading logs from server: %s", err)
			}

			if _, err := io.Copy(os.Stdout, logs); err != nil {
				log.Fatalf("Error copying server output to stdout: %s", err)
			}
		},
	}

	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().IntP("tail", "t", 20,
		"The number of previous log lines to print immediately.")
}

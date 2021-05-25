package cmd

import (
	"io"
	"os"

	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

// NewLogsCmd returns the logs command which tails the server cli output.
func NewLogsCmd() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs <server>",
		Short: "Output server logs",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := craft.GetServerOrExit(args[0])

			tail, err := cmd.Flags().GetInt("tail")
			if err != nil {
				panic(err)
			}

			logs, err := c.LogReader(tail)
			if err != nil {
				logger.Error.Fatalf("reading logs from server: %s", err)
			}

			if _, err := io.Copy(os.Stdout, logs); err != nil {
				logger.Error.Fatalf("copying server output to stdout: %s", err)
			}
		},
	}

	logsCmd.Flags().IntP("tail", "t", 20,
		"The number of previous log lines to print immediately.")

	return logsCmd
}

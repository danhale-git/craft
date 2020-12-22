package cmd

import (
	"io"
	"log"
	"os"

	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

func init() {
	logsCmd := &cobra.Command{
		Use:   "logs <server name>",
		Short: "Copy server log output to your console.",
		Long: `One argument is required which is the server name.
Run 'craft list' to see a list of active server names.`,
		// Accept only one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		// Read logs from a container and copy them to stdout
		Run: func(cmd *cobra.Command, args []string) {
			c := craft.NewDockerClientOrExit(args[0])

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

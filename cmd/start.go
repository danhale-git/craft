package cmd

import (
	"strings"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

func init() {
	// startCmd represents the start command
	startCmd := &cobra.Command{
		Use:   "start <servers...>",
		Short: "Start a stopped server.",
		Long: `Start creates a new server from the latest backup for the given server name(s).

If no port is specified then an unused one will be chosen. Whether the port is unused is determined by examining all
other craft containers. The lowest available port between 19132 and 19232 will be assigned.
If multiple arguments are provided, the --port flag is ignored and ports are assigned automatically.`,
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(1)(cmd, args)
		},
		Run: StartCommand,
	}

	rootCmd.AddCommand(startCmd)

	startCmd.Flags().IntP("port", "p", 0,
		"External port for players connect to. Default (0 value) is to auto-assign a port.")
}

func StartCommand(cmd *cobra.Command, args []string) {
	started := make([]string, 0)

	var port int

	var err error

	if len(args) > 1 {
		port = 0
	} else {
		port, err = cmd.Flags().GetInt("port")
		if err != nil {
			panic(err)
		}
	}

	for _, name := range args {
		c, err := docker.RunContainer(port, name)
		if err != nil {
			logger.Error.Printf("%s: running server: %s", name, err)
			continue
		}

		f := latestBackupFileName(c.ContainerName)

		err = restoreBackup(c, f.Name())
		if err != nil {
			logger.Error.Printf("%s: loading backup file to server: %s", name, err)

			if err := c.Stop(); err != nil {
				panic(err)
			}

			continue
		}

		if err = runServer(c); err != nil {
			logger.Error.Printf("%s: starting server process: %s", name, err)

			if err := c.Stop(); err != nil {
				panic(err)
			}

			continue
		}

		started = append(started, name)
	}

	logger.Info.Println("started:", strings.Join(started, " "))
}

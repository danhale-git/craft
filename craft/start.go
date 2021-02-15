package craft

import (
	"strings"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

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

		if err = RunServer(c); err != nil {
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

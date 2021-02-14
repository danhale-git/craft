package cmd

import (
	"strings"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

// StopCommand attempts to take a backup unless the no-backup flag is true. If the backup is successful the
// server process is stopped then the docker container is stopped.
func StopCommand(cmd *cobra.Command, args []string) {
	stopped := make([]string, 0)

	noBackup, err := cmd.Flags().GetBool("no-backup")
	if err != nil {
		panic(err)
	}

	for _, name := range args {
		c := docker.NewContainerOrExit(name)

		if !noBackup {
			_, err = copyBackup(c)
			if err != nil {
				logger.Error.Printf("%s: taking backup: %s", name, err)
				continue
			}
		}

		if err = c.Command([]string{"stop"}); err != nil {
			logger.Error.Printf("%s: running 'stop' command: %s", name, err)
			continue
		}

		if err := c.Stop(); err != nil {
			logger.Error.Printf("%s: stopping container: %s", name, err)
			continue
		}

		stopped = append(stopped, c.ContainerName)
	}

	logger.Info.Println("stopped:", strings.Join(stopped, " "))
}

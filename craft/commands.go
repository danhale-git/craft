package craft

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/danhale-git/craft/internal/backup"
	"github.com/danhale-git/craft/internal/docker"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

func ListCommand(cmd *cobra.Command, args []string) {
	w := tabwriter.NewWriter(os.Stdout, 3, 3, 3, ' ', tabwriter.TabIndent)

	// List running servers
	servers, err := docker.ActiveServerClients()
	if err != nil {
		log.Fatalf("Error getting server clients: %s", err)
	}

	for _, s := range servers {
		c, err := docker.NewContainer(s.ContainerName)
		if err != nil {
			log.Fatalf("Error creating docker client for container '%s': %s", s.ContainerName, err)
		}

		port, err := c.GetPort()
		if err != nil {
			log.Fatalf("Error getting port for container '%s': '%s'", s.ContainerName, err)
		}

		if _, err := fmt.Fprintf(w, "%s\trunning - port %d\n", s.ContainerName, port); err != nil {
			log.Fatalf("Error writing to table: %s", err)
		}
	}

	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		panic(err)
	}

	if !all {
		if err = w.Flush(); err != nil {
			log.Fatalf("Error writing output to console: %s", err)
		}

		return
	}

	for _, n := range backupServerNames() {
		// If n is in list of active server names
		if func() bool {
			for _, s := range servers {
				if s.ContainerName == n {
					return true
				}
			}
			return false
		}() {
			continue
		}

		f := latestBackupFileName(n)

		t, err := backup.FileTime(f.Name())
		if err != nil {
			panic(err)
		}

		if _, err := fmt.Fprintf(w, "%s\tstopped - %s\n", n, t.Format(timeFormat)); err != nil {
			log.Fatalf("Error writing to table: %s", err)
		}
	}

	if err = w.Flush(); err != nil {
		log.Fatalf("Error writing output to console: %s", err)
	}
}

func ConfigureCommand(cmd *cobra.Command, args []string) {
	c := docker.NewContainerOrExit(args[0])

	props, err := cmd.Flags().GetStringSlice("prop")
	if err != nil {
		panic(err)
	}

	if err := setServerProperties(props, c); err != nil {
		logger.Error.Fatalf("setting server properties: %s", err)
	}
}
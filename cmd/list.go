package cmd

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

var timeFormat = "02 Jan 2006 3:04PM"

func init() {
	// listCmd represents the list command
	listCmd := &cobra.Command{
		Use:   "list <server>",
		Short: "List servers",
		Run: func(cmd *cobra.Command, args []string) {
			w := tabwriter.NewWriter(os.Stdout, 3, 3, 3, ' ', tabwriter.TabIndent)

			// List running servers
			servers, err := craft.ActiveServerClients()
			if err != nil {
				log.Fatalf("Error getting server clients: %s", err)
			}

			for _, s := range servers {
				c, err := craft.NewDockerClient(s.ContainerName)
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

			// List backed up servers
			backupNames, err := craft.BackupServerNames()
			if err != nil {
				log.Fatalf("Error getting backups: %s", err)
			}

			for _, n := range backupNames {
				// name is in activeNames
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

				_, t, err := craft.LatestServerBackup(n)
				if err != nil {
					log.Fatalf("Error getting latest backup: %s", err)
				}

				if _, err := fmt.Fprintf(w, "%s\tstopped - %s\n", n, t.Format(timeFormat)); err != nil {
					log.Fatalf("Error writing to table: %s", err)
				}
			}

			if err = w.Flush(); err != nil {
				log.Fatalf("Error writing output to console: %s", err)
			}
		},
	}

	listCmd.Flags().BoolP("all", "a", false, "Show all servers. Defaults to only running servers.")

	rootCmd.AddCommand(listCmd)
}

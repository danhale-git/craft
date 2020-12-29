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
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeNames, err := craft.ServerNames()
			if err != nil {
				return err
			}

			backupNames, err := craft.BackupServerNames()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 3, 3, 3, ' ', tabwriter.TabIndent)

			for _, name := range activeNames {
				c, err := craft.NewDockerClient(name)
				if err != nil {
					log.Fatalf("Error creating docker client for container '%s': %s", name, err)
				}

				port, err := c.GetPort()
				if err != nil {
					log.Fatalf("Error getting port for container '%s': '%s'", name, err)
				}

				if _, err := fmt.Fprintf(w, "%s\trunning - port %d\n", name, port); err != nil {
					log.Fatalf("Error writing to table: %s", err)
				}
			}
			for _, n := range backupNames {
				// name is in activeNames
				if func() bool {
					for _, an := range activeNames {
						if an == n {
							return true
						}
					}
					return false
				}() {
					continue
				}

				_, t, err := craft.LatestServerBackup(n)
				if err != nil {
					return fmt.Errorf("getting latest backup file name: %s", err)
				}

				if _, err := fmt.Fprintf(w, "%s\tstopped - %s\n", n, t.Format(timeFormat)); err != nil {
					log.Fatalf("Error writing to table: %s", err)
				}
			}

			if err = w.Flush(); err != nil {
				log.Fatalf("Error writing output to console: %s", err)
			}

			return nil
		},
	}

	rootCmd.AddCommand(listCmd)
}

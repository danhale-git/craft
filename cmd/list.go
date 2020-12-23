package cmd

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

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

			for _, n := range activeNames {
				if _, err := fmt.Fprintf(w, "%s\trunning\n", n); err != nil {
					log.Fatalf("Error writing to table: %s", err)
				}
			}
			for _, n := range backupNames {
				_, t, err := craft.LatestServerBackup(n)
				if err != nil {
					return fmt.Errorf("getting latest backup file name: %s", err)
				}
				if _, err := fmt.Fprintf(w, "%s\tstopped - %s\n", n, t.Format(time.Stamp)); err != nil {
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

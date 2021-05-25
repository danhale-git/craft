package cmd

import (
	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

// NewListCmd returns the list command which lists running and backed up servers.
func NewListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list <server>",
		Short: "List servers",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			all, err := cmd.Flags().GetBool("all")
			if err != nil {
				panic(err)
			}

			if err := craft.PrintServers(all); err != nil {
				logger.Error.Fatal(err)
			}
		},
	}

	listCmd.Flags().BoolP("all", "a", false,
		"Show all servers. The Default is to show only running servers.")

	return listCmd
}

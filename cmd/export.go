package cmd

import (
	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

// NewExportCommand returns the version command which prints the current craft version
func NewExportCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "export",
		Short: "Export the current world to a .mcworld file.",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			dir, err := cmd.Flags().GetString("destination")
			if err != nil {
				panic(err)
			}

			err = craft.ExportMCWorld(
				craft.GetServerOrExit(args[0]),
				dir,
			)
			if err != nil {
				logger.Error.Fatal(err)
			}
		},
	}

	command.Flags().StringP("destination", "d", "",
		"Directory to save the .mcworld file.")

	return command
}

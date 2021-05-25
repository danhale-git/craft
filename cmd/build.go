package cmd

import (
	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/craft"
	"github.com/spf13/cobra"
)

// NewBuildCommand returns the build command which builds the server container image
func NewBuildCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "build",
		Short: "Build the server image.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := craft.BuildImage(); err != nil {
				logger.Error.Fatalf("Error building image: %s", err)
			}
		},
	}

	return command
}

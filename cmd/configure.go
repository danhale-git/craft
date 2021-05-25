package cmd

import (
	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

// NewConfigureCmd returns the configure command which operates on server files.
func NewConfigureCmd() *cobra.Command {
	configureCmd := &cobra.Command{
		Use:   "configure <server>",
		Short: "Configure server properties.",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := craft.GetServerOrExit(args[0])

			props, err := cmd.Flags().GetStringSlice("prop")
			if err != nil {
				logger.Panic(err)
			}

			if err := craft.SetServerProperties(props, c); err != nil {
				logger.Error.Fatalf("setting server properties: %s", err)
			}
		},
	}

	configureCmd.Flags().StringSlice("prop", []string{},
		"A server property name and value e.g. 'gamemode=creative'.")
	_ = configureCmd.MarkFlagRequired("prop")

	return configureCmd
}

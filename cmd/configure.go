package cmd

import (
	"log"
	"strings"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/configure"
	"github.com/danhale-git/craft/internal/docker"
	"github.com/danhale-git/craft/internal/server"

	"github.com/spf13/cobra"
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure server properties, whitelist and mods.",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(1, 1)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		c := docker.NewContainerOrExit(args[0])

		props, err := cmd.Flags().GetStringSlice("prop")
		if err != nil {
			panic(err)
		}

		if err := setServerProperties(props, c); err != nil {
			logger.Error.Fatalf("setting server properties: %s", err)
		}
	},
}

func setServerProperties(propFlags []string, c *docker.Container) error {
	if len(propFlags) > 0 {
		k := make([]string, len(propFlags))
		v := make([]string, len(propFlags))

		for i, p := range propFlags {
			s := strings.Split(p, "=")
			if !strings.ContainsRune(p, '=') || len(s[0]) == 0 || len(s[1]) == 0 {
				log.Fatalf("Invalid property '%s' should be 'key=value'", p)
			}

			k[i] = s[0]
			v[i] = s[1]
		}

		b, err := c.CopyFileFrom(server.FilePaths.ServerProperties)
		if err != nil {
			return err
		}

		updated, err := configure.SetProperties(k, v, b)
		if err != nil {
			return err
		}

		if err = c.CopyFileTo(server.FilePaths.ServerProperties, updated); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configureCmd)

	configureCmd.Flags().StringSlice("prop", []string{}, "A server property name and value e.g. 'gamemode=creative'.")
}

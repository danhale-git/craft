package cmd

import (
	"fmt"
	"log"
	"strings"

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

		// Parse prop flags
		props, err := cmd.Flags().GetStringSlice("prop")
		if err != nil {
			panic(err)
		}

		if len(props) > 0 {
			k := make([]string, len(props))
			v := make([]string, len(props))

			for i, p := range props {
				s := strings.Split(p, "=")
				if !strings.ContainsRune(p, '=') || len(s[0]) == 0 || len(s[1]) == 0 {
					log.Fatalf("Invalid property '%s' should be 'key=value'", p)
				}

				k[i] = s[0]
				v[i] = s[1]
			}

			fmt.Println(server.FilePaths.ServerProperties)
			b, err := c.CopyFileFrom(server.FilePaths.ServerProperties)
			if err != nil {
				log.Fatalf("Error copying current server.properties from server: %s", err)
			}

			updated, err := configure.SetProperties(k, v, b)
			if err != nil {
				log.Fatalf("Error updating values: %s", err)
			}

			if err = c.CopyFileTo(server.RootDirectory, server.FileNames.ServerProperties, updated); err != nil {
				log.Fatalf("Error copying updated server.properties to server: %s", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)

	configureCmd.Flags().StringSlice("prop", []string{}, "A server property name and value e.g. 'gamemode=creative'.")
}

package cmd

import (
	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use: "run",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(1, 1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Create a container for the server
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}

		err = craft.Run(port, name)
		if err != nil {
			return err
		}

		// Get the container ID
		c := craft.GetContainerOrExit(name)

		// TODO: Warn the user when --backup is given alongside --world or --server-properties
		// If a world is specified, copy it
		backupPath, _ := cmd.Flags().GetString("backup")
		if backupPath != "" {
			err = craft.LoadBackup(c, backupPath)
			if err != nil {
				return err
			}

		} else {
			// If a world is specified, copy it
			worldPath, _ := cmd.Flags().GetString("world")
			if worldPath != "" {
				err = craft.LoadWorld(c, worldPath)
				if err != nil {
					return err
				}
			}

			// If a world is specified, copy it
			propsPath, _ := cmd.Flags().GetString("server-properties")
			if propsPath != "" {
				err = craft.LoadServerProperties(c, propsPath)
				if err != nil {
					return err
				}
			}
		}

		// Run the bedrock_server process
		err = craft.RunServer(c)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().String("world", "", "Path to a .mcworld file to be loaded.")
	runCmd.Flags().String("backup", "", "Path to a .zip server backup.")
	runCmd.Flags().String("server-properties", "", "Path to a server.properties file to be loaded.")

	// TODO: automatically chose an unused port if not given instead of using default port
	runCmd.Flags().IntP("port", "p", 19132, "External port players connect to.")
}

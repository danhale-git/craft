package cmd

import (
	"log"

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

		d := craft.NewDockerClientOrExit(name)

		var sb *craft.ServerBackup

		backupPath, _ := cmd.Flags().GetString("backup")
		if backupPath != "" {
			sb, err = craft.LoadBackup(d, backupPath)
			if err != nil {
				log.Fatalf("loading backup files from disk: %s", err)
			}

			err = sb.Restore()
			if err != nil {
				log.Fatalf("restoring backup: %s", err)
			}
		} else {
			worldPath, _ := cmd.Flags().GetString("world")
			propsPath, _ := cmd.Flags().GetString("server-properties")

			sb = &craft.ServerBackup{Docker: d}

			if worldPath != "" {
				err = sb.LoadFile(worldPath)
				if err != nil {
					return err
				}
			}

			if propsPath != "" {
				err = sb.LoadFile(propsPath)
				if err != nil {
					return err
				}
			}
		}

		if len(sb.Files) > 0 {
			err := sb.Restore()
			if err != nil {
				log.Fatalf("Error loading files to server: %s", err)
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

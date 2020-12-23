package cmd

import (
	"log"
	"strings"

	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

func init() {
	// runCmd represents the run command
	runCmd := &cobra.Command{
		Use:   "run <server name>",
		Short: "Create a new craft server with the given name.",
		Long:  "A craft backup .zip file may be provided, or a .mcworld file and/or server.properties.",
		// Require exactly one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		// Create a new docker container, copy files and run the mc server binary
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := cmd.Flags().GetInt("port")
			if err != nil {
				return err
			}

			// Create a container for the server
			d, err := craft.NewContainer(port, args[0])
			if err != nil {
				log.Fatalf("Error creating new container: %s", err)
			}

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

				if worldPath != "" || propsPath != "" {
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
			}

			if sb != nil && len(sb.Files) > 0 {
				err := sb.Restore()
				if err != nil {
					log.Fatalf("Error loading files to server: %s", err)
				}
			}

			// Run the bedrock_server process
			err = d.Command(strings.Split(craft.RunMCCommand, " "))
			if err != nil {
				return err
			}

			return nil
		},
	}

	rootCmd.AddCommand(runCmd)

	runCmd.Flags().String("world", "", "Path to a .mcworld file to be loaded.")
	runCmd.Flags().String("backup", "", "Path to a .zip server backup.")
	runCmd.Flags().String("server-properties", "", "Path to a server.properties file to be loaded.")

	// TODO: automatically chose an unused port if not given instead of using default port
	runCmd.Flags().IntP("port", "p", 19132, "External port players connect to.")
}

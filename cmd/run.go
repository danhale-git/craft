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

			var worldPath, propsPath string
			if worldPath, err = cmd.Flags().GetString("world"); err != nil {
				log.Fatal(err)
			}

			if propsPath, err = cmd.Flags().GetString("server-properties"); err != nil {
				log.Fatal(err)
			}

			// ServerFiles is used as a mechanism to load
			sb := &craft.ServerFiles{Docker: d}

			if worldPath != "" || propsPath != "" {
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

			if sb.Archive != nil && len(sb.Files) > 0 {
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
	runCmd.Flags().String("server-properties", "", "Path to a server.properties file to be loaded.")

	runCmd.Flags().IntP("port", "p", 0, "External port players connect to.")
}

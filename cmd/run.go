package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/danhale-git/craft/internal/craft"

	"github.com/spf13/cobra"
)

func init() {
	// runCmd represents the run command
	runCmd := &cobra.Command{
		Use:   "run <server name>",
		Short: "Create a new server",
		Long:  "Defaults to a new default world. .mcworld file and optional server.properties may be provided.",
		// Require exactly one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		// Create a new docker container, copy files and run the mc server binary
		RunE: func(cmd *cobra.Command, args []string) error {
			backups, err := craft.BackupServerNames()
			if err != nil {
				log.Fatalf("Error getting backups: %s", err)
			}

			for _, b := range backups {
				if args[0] == b {
					fmt.Printf("Error: server name '%s' is in use by a backup, run 'craft list -a'", args[0])
					os.Exit(0)
				}
			}

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

			sb := &craft.ServerFiles{Docker: d}

			// Load server files if requested
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

			// Customise server properties
			props, err := cmd.Flags().GetStringSlice("prop")
			if err != nil {
				panic(err)
			}

			if len(props) > 0 {
				for _, p := range props {
					keyVal := strings.Split(p, "=")
					if len(keyVal) != 2 || keyVal[0] == "" || keyVal[1] == "" {
						log.Fatalf("Invalid property: '%s'", p)
					}

					if err := sb.UpdateServerProperties(keyVal[0], keyVal[1]); err != nil {
						log.Fatalf("Error changing property '%s' to '%s'", keyVal[0], keyVal[1])
					}
				}
			}

			// Add files
			files, err := cmd.Flags().GetStringSlice("files")
			if err != nil {
				panic(err)
			}

			if len(files) > 0 {
				for _, path := range files {
					if err = sb.LoadZippedFiles(path); err != nil {
						log.Fatalf("Error loading file at %s: %s", path, err)
					}
				}
			}

			// Restore all server files if needed
			if sb.Archive != nil && len(sb.Files) > 0 {
				err := sb.Restore()
				if err != nil {
					log.Fatalf("Error loading files to server: %s", err)
				}
			}

			go func() {
				logs, err := d.LogReader(20)
				if err != nil {
					log.Fatalf("Error reading logs from server: %s", err)
				}

				if _, err := io.Copy(os.Stdout, logs); err != nil {
					log.Fatalf("Error copying server output to stdout: %s", err)
				}
			}()

			// Run the bedrock_server process
			err = d.Command(strings.Split(craft.RunMCCommand, " "))
			if err != nil {
				return err
			}

			return nil
		},
	}

	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntP("port", "p", 0, "External port players connect to.")
	runCmd.Flags().String("world", "", "Path to a .mcworld file to be loaded.")
	runCmd.Flags().String("server-properties", "", "Path to a server.properties file to be loaded.")
	runCmd.Flags().StringSlice("prop", []string{}, "A server property name and value e.g. 'gamemode=creative'.")
	runCmd.Flags().StringSlice("files", []string{},
		"Full local path to a zip file containing files which will be added to the mc server directory.")
}

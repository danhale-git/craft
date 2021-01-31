package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

const (
	RunMCCommand = "cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server"
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
			backups, err := backupServerNames()
			if err != nil {
				log.Fatalf("Error getting backups: %s", err)
			}

			// Check the server doesn't already exist
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
			c, err := docker.RunContainer(port, args[0])
			if err != nil {
				log.Fatalf("Error creating new container: %s", err)
			}

			// Run the server process
			if err = runServer(c); err != nil {
				log.Fatalf("Error starting server process: %s", err)
			}

			return nil
		},
	}

	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntP("port", "p", 0, "External port players connect to.")
	runCmd.Flags().String("world", "", "Path to a .mcworld file to be loaded.")
}

// runServer executes the server binary and waits for the server to be ready before returning.
func runServer(c *docker.Container) error {
	// Run the bedrock_server process
	if err := c.Command(strings.Split(RunMCCommand, " ")); err != nil {
		return err
	}

	logs, err := c.LogReader(-1) // Negative number results in all logs
	if err != nil {
		return err
	}

	s := bufio.NewScanner(logs)
	s.Split(bufio.ScanLines)

	for s.Scan() {
		if s.Text() == "[INFO] Server started." {
			// Server has finished starting
			return nil
		}
	}

	return fmt.Errorf("reached end of log reader without finding the 'Server started' message")
}

package cmd

import (
	"archive/zip"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/backup"

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
		Run: RunCommand,
	}

	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntP("port", "p", 0, "External port players connect to.")
	runCmd.Flags().String("world", "", "Path to a .mcworld file to be loaded.")
}

func RunCommand(cmd *cobra.Command, args []string) {
	// Check the server doesn't already exist
	for _, b := range backupServerNames() {
		if args[0] == b {
			logger.Error.Fatalf("server name '%s' is in use by a backup, run 'craft list -a'", args[0])
		}
	}

	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		logger.Panic(err)
	}

	// Create a container for the server
	c, err := docker.RunContainer(port, args[0])
	if err != nil {
		logger.Error.Fatalf("creating new container: %s", err)
	}

	mcworld, err := cmd.Flags().GetString("world")
	if err != nil {
		logger.Panic(err)
	}

	// Copy the world files to the server
	if mcworld != "" {
		if err := checkWorldFiles(mcworld); err != nil {
			logger.Error.Printf("invalid mcworld file: %s", err)

			if err := c.Stop(); err != nil {
				panic(err)
			}

			os.Exit(0)
		}

		// Open backup zip
		zr, err := zip.OpenReader(mcworld)
		if err != nil {
			logger.Panic(err)
		}

		if err = backup.RestoreMCWorld(&zr.Reader, c.CopyTo); err != nil {
			logger.Error.Printf("restoring backup: %s", err)

			if err := c.Stop(); err != nil {
				panic(err)
			}

			os.Exit(1)
		}

		if err = zr.Close(); err != nil {
			logger.Panicf("closing zip: %s", err)
		}
	}

	// Run the server process
	if err = runServer(c); err != nil {
		logger.Error.Fatalf("starting server process: %s", err)
	}
}

func checkWorldFiles(mcworld string) error {
	expected := map[string]bool{
		"db/CURRENT":    false,
		"level.dat":     false,
		"levelname.txt": false,
	}

	zr, err := zip.OpenReader(mcworld)
	if err != nil {
		return fmt.Errorf("failed to open zip: %s", err)
	}

	for _, f := range zr.File {
		expected[f.Name] = true
	}

	if !(expected["db/CURRENT"] &&
		expected["level.dat"] &&
		expected["levelname.txt"]) {
		return fmt.Errorf("missing one of: db, level.dat, levelname.txt")
	}

	if err = zr.Close(); err != nil {
		return fmt.Errorf("failed to close zip: %s", err)
	}

	return nil
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

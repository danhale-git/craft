package cmd

import (
	"archive/zip"
	"bufio"
	"fmt"
	"strings"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/danhale-git/craft/internal/backup"

	"github.com/danhale-git/craft/internal/docker"

	"github.com/spf13/cobra"
)

const (
	RunMCCommand = "cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server"
)

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

	// Copy world files to the server
	if mcworld != "" {
		if err := loadMCWorldFile(mcworld, c); err != nil {
			if err := c.Stop(); err != nil {
				panic(err)
			}

			logger.Error.Fatalf("loading world file")
		}
	}

	props, err := cmd.Flags().GetStringSlice("prop")
	if err != nil {
		panic(err)
	}

	if err := setServerProperties(props, c); err != nil {
		logger.Error.Fatalf("setting server properties: %s", err)
	}

	// Run the server process
	if err = runServer(c); err != nil {
		logger.Error.Fatalf("starting server process: %s", err)
	}
}

func loadMCWorldFile(mcworld string, c *docker.Container) error {
	if err := checkWorldFiles(mcworld); err != nil {
		return fmt.Errorf("invalid mcworld file: %s", err)
	}

	// Open backup zip
	zr, err := zip.OpenReader(mcworld)
	if err != nil {
		logger.Panic(err)
	}

	if err = backup.RestoreMCWorld(&zr.Reader, c.CopyTo); err != nil {
		return fmt.Errorf("restoring backup: %s", err)
	}

	if err = zr.Close(); err != nil {
		logger.Panicf("closing zip: %s", err)
	}

	return nil
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

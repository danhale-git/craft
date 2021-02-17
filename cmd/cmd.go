package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/danhale-git/craft/craft"

	"github.com/danhale-git/craft/internal/docker"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

func InitCobra() *cobra.Command {
	rootCmd := NewRootCmd()

	// Call all constructor functions
	for _, f := range commands() {
		rootCmd.AddCommand(f())
	}

	return rootCmd
}

func commands() []func() *cobra.Command {
	return []func() *cobra.Command{
		NewRootCmd,
		NewRunCmd,
		NewCommandCmd,
		NewBackupCmd,
		NewStartCmd,
		NewStopCmd,
		NewLogsCmd,
		NewListCmd,
		NewConfigureCmd,
		NewVersionCmd,
	}
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "craft",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logPath, err := cmd.Flags().GetString("log")
			if err != nil {
				panic(fmt.Sprintln(cmd.Name(), err))
			}

			logLevel, err := cmd.Flags().GetString("log-level")
			if err != nil {
				panic(err)
			}

			logger.Init(logPath, logLevel, fmt.Sprintf("[%s]", cmd.Name()))
		},
	}

	rootCmd.PersistentFlags().String("log", "",
		"Path to the file where logs are saved.")

	rootCmd.PersistentFlags().String("log-level", "info",
		"Minimum severity of logs to output. [info|warn|error].")

	return rootCmd
}

// NewRunCmd returns the run command which creates a new craft server container and runs the server process.
func NewRunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run <server name>",
		Short: "Create a new server",
		Long: `Runs a new docker container and runs the server process within it.
A .mcworld file and custom server.properties fields may be provided via command line flags.
When setting multiple properties, provide each one as a separate flag. Each flag should define only property field.
If no port flag is provided, the lowest available (unused by docker) port between 19132 and 19232 will be used.`,
		Example: `craft run mynewserver --world C:\Users\MyUser\Downloads\exported_world.mcworld --prop difficulty=hard`,
		// Takes exactly one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			port, err := cmd.Flags().GetInt("port")
			if err != nil {
				logger.Panic(err)
			}

			mcworld, err := cmd.Flags().GetString("world")
			if err != nil {
				logger.Panic(err)
			}

			props, err := cmd.Flags().GetStringSlice("prop")
			if err != nil {
				logger.Panic(err)
			}

			err = craft.CreateServer(args[0], mcworld, port, props)
			if err != nil {
				logger.Error.Fatalf("creating server: %s", err)
			}
		},
	}

	runCmd.Flags().Int("port", 0,
		"External port for players connect to. Default (0 value) will auto-assign a port.")

	runCmd.Flags().String("world", "",
		"Path to a .mcworld file to be loaded.")

	runCmd.Flags().StringSlice("prop", nil,
		"A server.properties field e.g. --prop gamemode=survival")

	return runCmd
}

// NewCommandCmd returns the command command which executes a mc server command on the given server.
func NewCommandCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "command <server> <mc command>",
		Aliases: []string{"cmd"},
		Short:   "Run a command on a server",
		Long:    `The first argument is the serer name. The following arguments will be executed in the server CLI.`,
		Example: "craft command myserver give PlayerName stone 1",
		// Takes 2 or more arguments
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(2)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := docker.NewContainerOrExit(args[0]).Command(args[1:])
			if err != nil {
				logger.Error.Fatalf("running command '%s': %s", strings.Join(args[1:], " "), err)
			}
		},
	}
}

// NewBackupCmd returns the backup command which saves a local backup of the server and world.
func NewBackupCmd() *cobra.Command {
	backupCmd := &cobra.Command{
		Use:   "backup <server names...>",
		Short: "Take a backup",
		Long: `
Save the current world and server.properties to a zip file in the backup directory.
If two backups are taken in the same minute, the second will overwrite the first.
Backups are saved to a default directory under the user's home directory.
The backed up world is usually a few seconds behind the world state at the time of backup.
Use the trim and skip-trim-file-removal-check flags with linux cron or windows task scheduler to automate backups.`,
		Example: `craft backup myserver
craft backup myserver -l

Linux cron (hourly):
0 * * * * ~/craft_backups/backup.sh
	
	#!/usr/bin/env bash
	~/go/bin/craft backup myserver myotherserver \ # path to craft executable and one or more servers
	--skip-trim-file-removal-check --trim 3 \ # skip cmdline prompts and delete all except 3 newest files
	--log ~/craft_backups/backup.log --log-level info # log to file with log level info
`,
		// Takes one or more arguments
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			trim, err := cmd.Flags().GetInt("trim")
			if err != nil {
				logger.Panic(err)
			}

			list, err := cmd.Flags().GetBool("list")
			if err != nil {
				logger.Panic(err)
			}

			skip, err := cmd.Flags().GetBool("skip-trim-file-removal-check")
			if err != nil {
				logger.Panic(err)
			}

			craft.BackupCommand(trim, list, skip, args)
		},
	}

	backupCmd.Flags().IntP("trim", "t", 0,
		"Delete the oldest backup files, leaving the given count of newest files in place.")

	backupCmd.Flags().BoolP("list", "l", false,
		"List backup files and take no other action.")

	backupCmd.Flags().Bool("skip-trim-file-removal-check", false,
		"Don't prompt the user before removing files. Useful for automating backups.")

	return backupCmd
}

// NewStartCmd returns the start command which starts a server from the most recent backup.
func NewStartCmd() *cobra.Command {
	// startCmd represents the start command
	startCmd := &cobra.Command{
		Use:   "start <servers...>",
		Short: "Start a stopped server.",
		Long: `Start creates a new server from the latest backup for the given server name(s).

If no port flag is provided, the lowest available (unused by docker) port between 19132 and 19232 will be used.
If multiple arguments are provided, the --port flag is ignored and ports are assigned automatically.`,
		// Takes one or more arguments
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			started := make([]string, 0)

			var port int
			var err error

			if len(args) > 1 {
				port = 0
			} else {
				port, err = cmd.Flags().GetInt("port")
				if err != nil {
					panic(err)
				}
			}

			for _, name := range args {
				_, err := craft.RunLatestBackup(name, port)
				if err != nil {
					logger.Error.Fatal(err)
				}

				started = append(started, name)
			}

			logger.Info.Println("started:", strings.Join(started, " "))
		},
	}

	startCmd.Flags().IntP("port", "p", 0,
		"External port for players connect to. Default (0 value) will auto-assign a port.")

	return startCmd
}

// NewStopCmd returns the stop command which takes a backup and stops the server.
func NewStopCmd() *cobra.Command {
	// stopCmd represents the stop command
	stopCmd := &cobra.Command{
		Use:   "stop <servers...>",
		Short: "Back up and stop a running server.",
		Long:  `Back up the server then stop it. If the backup process fails, the server will not be stopped. `,
		// Takes at least one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			stopped := make([]string, 0)

			for _, name := range args {
				// TODO: Should skip here if ContainerNotFoundError, not exit
				c := docker.NewContainerOrExit(name)
				if _, err := craft.CopyBackup(c); err != nil {
					logger.Error.Printf("%s: error while taking backup: %s", c.ContainerName, err)
					continue
				}

				if err := craft.Stop(c); err != nil {
					logger.Error.Printf("%s: stopping server: %s", c.ContainerName, err)
				}
				stopped = append(stopped, c.ContainerName)
			}

			logger.Info.Println("stopped:", strings.Join(stopped, " "))
		},
	}

	return stopCmd
}

func NewLogsCmd() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs <server>",
		Short: "Output server logs",
		// Accept only one argument
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		// Read logs from a container and copy them to stdout
		Run: func(cmd *cobra.Command, args []string) {
			c := docker.NewContainerOrExit(args[0])

			tail, err := cmd.Flags().GetInt("tail")
			if err != nil {
				panic(err)
			}

			logs, err := c.LogReader(tail)
			if err != nil {
				log.Fatalf("Error reading logs from server: %s", err)
			}

			if _, err := io.Copy(os.Stdout, logs); err != nil {
				log.Fatalf("Error copying server output to stdout: %s", err)
			}
		},
	}

	logsCmd.Flags().IntP("tail", "t", 20,
		"The number of previous log lines to print immediately.")

	return logsCmd
}

func NewListCmd() *cobra.Command {
	// listCmd represents the list command
	listCmd := &cobra.Command{
		Use:   "list <server>",
		Short: "List servers",
		Run:   craft.ListCommand,
	}

	listCmd.Flags().BoolP("all", "a", false, "Show all servers. Defaults to only running servers.")

	return listCmd
}

func NewConfigureCmd() *cobra.Command {
	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure server properties, whitelist and mods.",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		Run: craft.ConfigureCommand,
	}

	configureCmd.Flags().StringSlice("prop", []string{}, "A server property name and value e.g. 'gamemode=creative'.")

	return configureCmd
}

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the current craft version.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("craft version 0.1.0")
		},
	}
}

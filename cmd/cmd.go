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
		NewExportCommand,
		NewBuildCommand,
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

			ok, err := docker.CheckImage()
			if err != nil {
				log.Fatalf("Error checking docker images: %s", err)
			}
			if !ok && cmd.Name() != "build" {
				fmt.Println("server image doesn't exist, run 'craft build' to build it")
				os.Exit(0)
			}
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

			var mcwFile craft.ZipOpener
			if mcworld != "" {
				mcwFile = craft.MCWorld{Path: mcworld}
			}

			err = craft.CreateServer(args[0], port, props, mcwFile)
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
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(2)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := docker.GetContainerOrExit(args[0])

			logs, err := c.LogReader(0)
			if err != nil {
				logger.Error.Fatalf("retrieving log reader: %s", err)
			}

			if err = c.Command(args[1:]); err != nil {
				logger.Error.Fatalf("running command '%s': %s", strings.Join(args[1:], " "), err)
			}

			// Wait for command to return before exiting
			for i := 0; i < 2; i++ {
				response, err := logs.ReadString('\n')
				if err != nil {
					logger.Error.Fatalf("reading response %d from logs: %s", i, err)
				}

				if strings.HasPrefix(response, "Syntax error:") {
					logger.Error.Printf("error reported from server cli: %s", response)
				}
			}
		},
	}
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
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
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
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			stopped := make([]string, 0)
			c := docker.GetContainerOrExit(args[0])

			if _, err := craft.CopyBackup(c); err != nil {
				logger.Error.Fatalf("%s: error while taking backup: %s", c.ContainerName, err)
			}

			if err := craft.Stop(c); err != nil {
				logger.Error.Fatalf("%s: stopping server: %s", c.ContainerName, err)
			}
			stopped = append(stopped, c.ContainerName)

			logger.Info.Println("stopped:", strings.Join(stopped, " "))
		},
	}

	return stopCmd
}

// NewLogsCmd returns the logs command which tails the server cli output.
func NewLogsCmd() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs <server>",
		Short: "Output server logs",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := docker.GetContainerOrExit(args[0])

			tail, err := cmd.Flags().GetInt("tail")
			if err != nil {
				panic(err)
			}

			logs, err := c.LogReader(tail)
			if err != nil {
				logger.Error.Fatalf("reading logs from server: %s", err)
			}

			if _, err := io.Copy(os.Stdout, logs); err != nil {
				logger.Error.Fatalf("copying server output to stdout: %s", err)
			}
		},
	}

	logsCmd.Flags().IntP("tail", "t", 20,
		"The number of previous log lines to print immediately.")

	return logsCmd
}

// NewListCmd returns the list command which lists running and backed up servers.
func NewListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list <server>",
		Short: "List servers",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			all, err := cmd.Flags().GetBool("all")
			if err != nil {
				panic(err)
			}

			if err := craft.PrintServers(all); err != nil {
				logger.Error.Fatal(err)
			}
		},
	}

	listCmd.Flags().BoolP("all", "a", false,
		"Show all servers. The Default is to show only running servers.")

	return listCmd
}

// NewConfigureCmd returns the configure command which operates on server files.
func NewConfigureCmd() *cobra.Command {
	configureCmd := &cobra.Command{
		Use:   "configure <server>",
		Short: "Configure server properties.",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.RangeArgs(1, 1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := docker.GetContainerOrExit(args[0])

			props, err := cmd.Flags().GetStringSlice("prop")
			if err != nil {
				logger.Panic(err)
			}

			if err := craft.SetServerProperties(props, c); err != nil {
				logger.Error.Fatalf("setting server properties: %s", err)
			}
		},
	}

	configureCmd.Flags().StringSlice("prop", []string{},
		"A server property name and value e.g. 'gamemode=creative'.")
	_ = configureCmd.MarkFlagRequired("prop")

	return configureCmd
}

// NewExportCommand returns the version command which prints the current craft version
func NewExportCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "export",
		Short: "Export the current world to a .mcworld file.",
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.ExactArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			dir, err := cmd.Flags().GetString("destination")
			if err != nil {
				panic(err)
			}

			err = craft.ExportMCWorld(
				docker.GetContainerOrExit(args[0]),
				dir,
			)
			if err != nil {
				logger.Error.Fatal(err)
			}
		},
	}

	command.Flags().StringP("destination", "d", "",
		"Directory to save the .mcworld file.")

	return command
}

// NewExportCommand returns the version command which prints the current craft version
func NewBuildCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "build",
		Short: "Build the server image.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := docker.BuildImage(); err != nil {
				log.Fatalf("Error building image: %s", err)
			}
		},
	}

	return command
}

// NewVersionCmd returns the version command which prints the current craft version
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the current craft version.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("craft version 0.1.0")
		},
	}
}

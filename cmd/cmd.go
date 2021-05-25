package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/danhale-git/craft/craft"

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

// NewRootCmd returns the root command which always runs before all other commands
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

			ok, err := craft.ImageExists(craft.NewClient())
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

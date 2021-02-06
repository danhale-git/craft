package cmd

import (
	"fmt"
	"os"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("log", "",
		"Path to the file where logs are saved.")

	rootCmd.PersistentFlags().String("log-level", "error",
		"Minimum severity of logs to output. [info|warn|error].")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	/*// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Search config in home directory with name ".craft" (without extension).
	viper.AddConfigPath(home)
	viper.SetConfigName(".craft")

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.Println("No config file ('.craft') found in", home)
	}*/
}

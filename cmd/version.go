package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current craft version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("craft version 0.1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

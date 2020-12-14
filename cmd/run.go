package cmd

import (
	"log"

	"github.com/danhale-git/craft/internal/server"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use: "run",
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.RangeArgs(1, 1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		worldPath, _ := cmd.Flags().GetString("world")

		err := server.Run(19133, name)
		if err != nil {
			return err
		}

		c, ok := server.ContainerFromName(name)
		if !ok {
			log.Fatal("container doesn't exist")
		}

		if worldPath != "" {
			err = server.LoadWorld(c.ID, worldPath)
			if err != nil {
				return err
			}
		}

		err = server.RunMC(c.ID)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("world", "w", "", "Path to a .mcworld file to be loaded.")
}

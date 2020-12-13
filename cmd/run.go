package cmd

import (
	"log"

	"github.com/danhale-git/craft/internal/server"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		worldPath, err := cmd.Flags().GetString("world")
		if err != nil {
			return err
		}

		err = server.Run(19133, "mc")
		if err != nil {
			return err
		}

		c, ok := server.ContainerFromName("mc")
		if !ok {
			log.Fatal("container doesn't exist")
		}

		err = server.LoadWorld(c.ID, worldPath)
		if err != nil {
			return err
		}

		_, err = server.RunMC(c.ID)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("world", "w", "", "Path to a .mcworld file to be loaded.")
	_ = runCmd.MarkFlagRequired("world")
}

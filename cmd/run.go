package cmd

import (
	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

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

			c, err := craft.NewServer(args[0], port, props, mcwFile)
			if err != nil {
				logger.Error.Fatalf("creating server: %s", err)
			}

			// Run the server process
			if err = craft.RunBedrock(c); err != nil {
				logger.Error.Fatalf("starting server process: %s", err)
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

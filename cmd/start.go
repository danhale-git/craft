package cmd

import (
	"strings"

	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

// NewStartCmd returns the start command which starts a server from the most recent backup.
func NewStartCmd() *cobra.Command {
	// startCmd represents the start command
	startCmd := &cobra.Command{
		Use:   "start <servers...>",
		Short: "Start a stopped server.",
		Long: `Start creates a new server from the latest backup for the given server name(s).

If no port flag is provided, the lowest available (unused by docker) port between 19132 and 19232 will be used.
If multiple arguments are provided, the --port flag is ignored and ports are assigned automatically.`,
		Args: cobra.MinimumNArgs(1),
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
					logger.Error.Println(err)
					continue
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

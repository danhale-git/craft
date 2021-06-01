package cmd

import (
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/danhale-git/craft/internal/server"

	"github.com/danhale-git/craft/internal/logger"

	"github.com/spf13/cobra"
)

const (
	webURL            = "https://www.minecraft.net/en-us/download/server/bedrock"
	downloadURLRegexp = "https\\:\\/\\/.*\\/bin-linux\\/bedrock-server-.*zip"
	webHelp           = `Copy the URL from https://www.minecraft.net/en-us/download/server/bedrock > UBUNTU SERVER > DOWNLOAD
ending bedrock-server-<version>.zip`
)

// NewBuildCommand returns the build command which builds the server container image
func NewBuildCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "build",
		Short: "Build the server image.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			urlString, err := cmd.Flags().GetString("url")
			if err != nil {
				logger.Error.Panic(err)
			}

			if urlString == "" {
				logger.Error.Fatalf("Error: value of 'url' flag is an empty string. %s", webHelp)
			}

			u, err := url.Parse(urlString)
			if err != nil {
				logger.Error.Fatalf("error parsing url: %s", err)
			}

			if err := server.BuildDockerImage(u.String()); err != nil {
				logger.Error.Fatalf("Error building image: %s", err)
			}
		},
	}

	downloadURL := make([]byte, 0)

	// Try to get the download URL. If something goes wrong the user will be asked to get it.
	res, err := http.Get(webURL)
	if err == nil {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			logger.Error.Fatalln(err)
		}

		downloadURLRegexp := regexp.MustCompile(downloadURLRegexp)
		downloadURL = downloadURLRegexp.Find(data)
	}

	command.Flags().String("url", string(downloadURL), webHelp)

	return command
}

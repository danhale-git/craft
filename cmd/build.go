package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/danhale-git/craft/server"

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

			noCache, err := cmd.Flags().GetBool("no-cache")
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

			if err := server.BuildDockerImage(u.String(), noCache); err != nil {
				logger.Error.Fatalf("Error building image: %s", err)
			}
		},
	}

	downloadURL, err := getServerDownloadURL()

	if err != nil {
		log.Printf("unable to query %s for latest version to auto populate --url flag: %s\n",
			webURL, err)
	}

	command.Flags().String("url", string(downloadURL), webHelp)
	command.Flags().Bool("no-cache", false,
		"If this flag is passed the server image will be built from scratch without using cached layers.")

	return command
}

func getServerDownloadURL() ([]byte, error) {
	downloadURL := make([]byte, 0)

	request, err := http.NewRequest("GET", webURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_1) AppleWebKit/537.36 (K HTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36")

	c := &http.Client{}

	res, err := c.Do(request)
	if err != nil {
		return nil, fmt.Errorf("sending http request: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("recieved status %s", res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %s", err)
	}

	downloadURLRegexp := regexp.MustCompile(downloadURLRegexp)
	downloadURL = downloadURLRegexp.Find(data)

	return downloadURL, nil
}

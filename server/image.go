package server

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/moby/term"

	"github.com/docker/docker/pkg/jsonmessage"

	_ "embed" // use embed package in this script

	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// DockerImageExists returns true if the craft server image exists.
func DockerImageExists(c client.ImageAPIClient) (bool, error) {
	images, err := c.ImageList(context.Background(), docker.ImageListOptions{})
	if err != nil {
		return false, err
	}

	for _, img := range images {
		if len(img.RepoTags) > 0 && img.RepoTags[0] == ImageName {
			return true, nil
		}
	}

	return false, nil
}

//go:embed Dockerfile
var dockerfile []byte //nolint:gochecknoglobals // embed needs a global

// BuildDockerImage builds the server image.
func BuildDockerImage(serverURL string, noCache bool) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Tar embedded dockerfile
	hdr := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing header: %s", err)
	}

	if _, err := tw.Write(dockerfile); err != nil {
		return fmt.Errorf("writing body: %s", err)
	}

	if err := tw.Close(); err != nil {
		return err
	}

	// Build image
	response, err := dockerClient().ImageBuild(
		context.Background(),
		&buf,
		docker.ImageBuildOptions{
			Dockerfile: "Dockerfile",
			Tags:       []string{ImageName},
			BuildArgs: map[string]*string{
				"URL": &serverURL,
			},
			NoCache: noCache,
		},
	)

	if err != nil {
		return err
	}

	// Output from build process
	termFd, isTerm := term.GetFdInfo(os.Stderr)

	err = jsonmessage.DisplayJSONMessagesStream(
		response.Body,
		os.Stderr,
		termFd,
		isTerm,
		nil,
	)
	if err != nil {
		return err
	}

	return response.Body.Close()
}

package craft

import (
	"context"
	"testing"

	"github.com/danhale-git/craft/internal/server"

	"github.com/danhale-git/craft/internal/mock"
	"github.com/docker/docker/api/types"
)

type ImageListClient struct {
	mock.DockerImageClient
	imageSummaries []types.ImageSummary
}

func (i ImageListClient) ImageList(_ context.Context, _ types.ImageListOptions) ([]types.ImageSummary, error) {
	return i.imageSummaries, nil
}

func TestServerClients(t *testing.T) {
	c := &mock.DockerContainerClient{ImageLabel: server.CraftLabel}

	servers, err := AllServers(c)
	if err != nil {
		t.Errorf("error returned after valid input: %s", err)
	}

	for i, want := range []string{"mc1_ID", "mc2_ID", "mc3_ID"} {
		got := servers[i].ContainerID
		if got != want {
			t.Errorf("unexpected server id: want %s: got %s", want, got)
		}
	}

	for i, want := range []string{"mc1", "mc2", "mc3"} {
		got := servers[i].ContainerName
		if got != want {
			t.Errorf("unexpected server name: want %s: got %s", want, got)
		}
	}
}

func TestCheckImage(t *testing.T) {
	client := ImageListClient{
		imageSummaries: []types.ImageSummary{
			{
				RepoTags: []string{"image"},
			},
			{
				RepoTags: []string{"otherimage"},
			},
			{
				RepoTags: []string{"anotherimage"},
			},
		},
	}

	got, err := ImageExists(client)
	if err != nil {
		t.Logf("%+v", client.imageSummaries)
		t.Errorf("error thrown with valid input: %s", err)
	}

	if got != false {
		t.Logf("%+v", client.imageSummaries)
		t.Errorf("unexpected return value when image doesn't exist: got %t: want false", got)
	}

	client.imageSummaries = append(client.imageSummaries, types.ImageSummary{RepoTags: []string{imageName}})

	got, err = ImageExists(client)
	if err != nil {
		t.Logf("%+v", client.imageSummaries)
		t.Errorf("error thrown with valid input: %s", err)
	}

	if got != true {
		t.Logf("%+v", client.imageSummaries)
		t.Errorf("unexpected return value when image exists: got %t: want true", got)
	}
}

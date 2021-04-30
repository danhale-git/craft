package mock

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

type DockerImageClient interface {
	client.ImageAPIClient
}

func ImageList(_ context.Context, _ types.ImageListOptions) ([]types.ImageSummary, error) {
	panic("mock.DockerImageClient.ImageList() is not implemented")
}

func ImageBuild(_ context.Context, _ io.Reader, _ types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	panic("mock.DockerImageClient.ImageBuild() is not implemented")
}

func BuildCachePrune(_ context.Context, _ types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error) {
	panic("mock.DockerImageClient.BuildCachePrune() is not implemented")
}

func BuildCancel(_ context.Context, _ string) error {
	panic("mock.DockerImageClient.BuildCancel() is not implemented")
}

func ImageCreate(_ context.Context, _ string, _ types.ImageCreateOptions) (io.ReadCloser, error) {
	panic("mock.DockerImageClient.ImageCreate() is not implemented")
}

func ImageHistory(_ context.Context, _ string) ([]image.HistoryResponseItem, error) {
	panic("mock.DockerImageClient.ImageHistory() is not implemented")
}

func ImageImport(_ context.Context, _ types.ImageImportSource, _ string, _ types.ImageImportOptions) (io.ReadCloser, error) {
	panic("mock.DockerImageClient.ImageImport() is not implemented")
}

func ImageInspectWithRaw(_ context.Context, _ string) (types.ImageInspect, []byte, error) {
	panic("mock.DockerImageClient.ImageInspectWithRaw() is not implemented")
}

func ImageLoad(_ context.Context, _ io.Reader, _ bool) (types.ImageLoadResponse, error) {
	panic("mock.DockerImageClient.ImageLoad() is not implemented")
}

func ImagePull(_ context.Context, _ string, _ types.ImagePullOptions) (io.ReadCloser, error) {
	panic("mock.DockerImageClient.ImagePull() is not implemented")
}

func ImagePush(_ context.Context, _ string, _ types.ImagePushOptions) (io.ReadCloser, error) {
	panic("mock.DockerImageClient.ImagePush() is not implemented")
}

func ImageRemove(_ context.Context, _ string, _ types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	panic("mock.DockerImageClient.ImageRemove() is not implemented")
}

func ImageSearch(_ context.Context, _ string, _ types.ImageSearchOptions) ([]registry.SearchResult, error) {
	panic("mock.DockerImageClient.ImageSearch() is not implemented")
}

func ImageSave(_ context.Context, _ []string) (io.ReadCloser, error) {
	panic("mock.DockerImageClient.ImageSave() is not implemented")
}

func ImageTag(_ context.Context, _ string, _ string) error {
	panic("mock.DockerImageClient.ImageTag() is not implemented")
}

func ImagesPrune(_ context.Context, _ filters.Args) (types.ImagesPruneReport, error) {
	panic("mock.DockerImageClient.ImagesPrune() is not implemented")
}

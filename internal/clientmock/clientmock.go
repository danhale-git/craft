package clientmock

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path"
	"time"

	"github.com/danhale-git/craft/internal/logger"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ContainerAPIDockerClientMock struct {
	Conn   net.Conn
	Reader *bufio.Reader

	CopyToFileNames []string
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerAttach(_ context.Context, _ string, _ types.ContainerAttachOptions) (types.HijackedResponse, error) {
	rw := types.HijackedResponse{
		Conn:   m.Conn,
		Reader: m.Reader,
	}

	fmt.Println("container attach returned")

	return rw, nil
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerLogs(_ context.Context, _ string, _ types.ContainerLogsOptions) (io.ReadCloser, error) {
	r := ioutil.NopCloser(bytes.NewReader([]byte(`NO LOG FILE! - setting up server logging...
[2020-12-22 20:24:38 INFO] Starting Server
[2020-12-22 20:24:38 INFO] Version 1.16.200.2
[2020-12-22 20:24:38 INFO] Session ID c20875b8-bc46-44e0-b862-2b7fb9563d14
[2020-12-22 20:24:38 INFO] Level Name: Bedrock level
[2020-12-22 20:24:38 INFO] Game mode: 1 Creative
[2020-12-22 20:24:38 INFO] Difficulty: 1 EASY
[INFO] opening worlds/Bedrock level/db
[INFO] IPv4 supported, port: 19132
[INFO] IPv6 not supported
[INFO] IPv4 supported, port: 33290
[INFO] IPv6 not supported
[INFO] Server started.`)))

	return r, nil
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerList(_ context.Context, _ types.ContainerListOptions) ([]types.Container, error) {
	containers := []types.Container{
		{ID: "mc1_ID", Names: []string{"/mc1"}},
		{ID: "mc2_ID", Names: []string{"/mc2"}},
		{ID: "mc3_ID", Names: []string{"/mc3"}},
	}

	return containers, nil
}

// func (cli *Client) CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options types.CopyToContainerOptions) error {
//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) CopyToContainer(_ context.Context, _ string, dstPath string, r io.Reader, _ types.CopyToContainerOptions) error {
	if m.CopyToFileNames == nil {
		panic("ContainerAPIDockerClientMock.CopyToFileNames must be assigned")
	}

	// Open and iterate through the files in the tar archive
	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}

		if err != nil {
			logger.Error.Fatal(err)
		}

		m.CopyToFileNames = append(m.CopyToFileNames, path.Join(dstPath, hdr.Name))
	}

	return nil
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerCommit(_ context.Context, _ string, _ types.ContainerCommitOptions) (types.IDResponse, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerCreate(_ context.Context, _ *container.Config, _ *container.HostConfig, _ *network.NetworkingConfig, _ *v1.Platform, _ string) (container.ContainerCreateCreatedBody, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerDiff(_ context.Context, _ string) ([]container.ContainerChangeResponseItem, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerExecAttach(_ context.Context, _ string, _ types.ExecStartCheck) (types.HijackedResponse, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerExecCreate(_ context.Context, _ string, _ types.ExecConfig) (types.IDResponse, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerExecInspect(_ context.Context, _ string) (types.ContainerExecInspect, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerExecResize(_ context.Context, _ string, _ types.ResizeOptions) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerExecStart(_ context.Context, _ string, _ types.ExecStartCheck) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerExport(_ context.Context, _ string) (io.ReadCloser, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerInspect(_ context.Context, _ string) (types.ContainerJSON, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerInspectWithRaw(_ context.Context, _ string, _ bool) (types.ContainerJSON, []byte, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerKill(_ context.Context, _ string, _ string) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerPause(_ context.Context, _ string) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerRemove(_ context.Context, _ string, _ types.ContainerRemoveOptions) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerRename(_ context.Context, _ string, _ string) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerResize(_ context.Context, _ string, _ types.ResizeOptions) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerRestart(_ context.Context, _ string, _ *time.Duration) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerStatPath(_ context.Context, _ string, _ string) (types.ContainerPathStat, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerStats(_ context.Context, _ string, _ bool) (types.ContainerStats, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerStatsOneShot(_ context.Context, _ string) (types.ContainerStats, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerStart(_ context.Context, _ string, _ types.ContainerStartOptions) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerStop(_ context.Context, _ string, _ *time.Duration) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerTop(_ context.Context, _ string, _ []string) (container.ContainerTopOKBody, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerUnpause(_ context.Context, _ string) error {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerUpdate(_ context.Context, _ string, _ container.UpdateConfig) (container.ContainerUpdateOKBody, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainerWait(_ context.Context, _ string, _ container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) CopyFromContainer(_ context.Context, _ string, _ string) (io.ReadCloser, types.ContainerPathStat, error) {
	panic("not implemented!")
}

//nolint:lll // mock method
func (m *ContainerAPIDockerClientMock) ContainersPrune(_ context.Context, _ filters.Args) (types.ContainersPruneReport, error) {
	panic("not implemented!")
}
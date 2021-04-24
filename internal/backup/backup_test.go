package backup

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/danhale-git/craft/internal/logger"
)

const mockTarContent = "some content"

func TestFileTime(t *testing.T) {
	valid := "test_18-43_01-02-2021.zip"

	var want int64 = 1612204980

	tme, err := FileTime(valid)
	if err != nil {
		t.Errorf("error returned for valid input: %s", err)
	}

	got := tme.Unix()

	if got != want {
		t.Errorf("unexpected value returned: want %d: got %d", want, got)
	}

	invalid := "18-43_01-02-2021.zip"

	_, err = FileTime(invalid)
	if err == nil {
		t.Error("no error returned for bad input", err)
	}

	if _, ok := err.(*time.ParseError); !ok {
		t.Errorf("unexpected error type: want time.ParseError: got %T", err)
	}
}

func TestRestore(t *testing.T) {
	// zip data and count of zipped files
	zippedBackup := mockZip(map[string]string{
		"worlds/Bedrock level/db/MANIFEST-000051": mockTarContent,
		"worlds/Bedrock level/db/000050.ldb":      mockTarContent,
		"worlds/Bedrock level/db/000053.log":      mockTarContent,
		"worlds/Bedrock level/db/000052.ldb":      mockTarContent,
		"worlds/Bedrock level/db/CURRENT":         mockTarContent,
		"worlds/Bedrock level/level.dat":          mockTarContent,
		"worlds/Bedrock level/level.dat_old":      mockTarContent,
		"worlds/Bedrock level/levelname.txt":      mockTarContent,
	})

	// zip data and count of zipped files
	zippedMCWorld := mockZip(map[string]string{
		"db/MANIFEST-000051": mockTarContent,
		"db/000050.ldb":      mockTarContent,
		"db/000053.log":      mockTarContent,
		"db/000052.ldb":      mockTarContent,
		"db/CURRENT":         mockTarContent,
		"level.dat":          mockTarContent,
		"level.dat_old":      mockTarContent,
		"levelname.txt":      mockTarContent,
	})

	backupNames, err := testRestoreFunc(zippedBackup, Restore)
	if err != nil {
		t.Error(err)
	}

	mcWorldNames, err := testRestoreFunc(zippedMCWorld, RestoreMCWorld)
	if err != nil {
		t.Error(err)
	}

	sort.Strings(backupNames)
	sort.Strings(mcWorldNames)

	// World files should be delivered consistently from mcworld and craft backup zips
	for i := 0; i < len(backupNames); i++ {
		if backupNames[i] != mcWorldNames[i] {
			t.Errorf(
				"mcworld destination path is different to equivalent backup destination path: '%s' != '%s'",
				mcWorldNames[i],
				backupNames[i],
			)
		}
	}
}

func testRestoreFunc(z *zip.Reader, restoreFunc func(*zip.Reader, string, client.ContainerAPIClient) error) ([]string, error) { //nolint:lll
	mockClient := &ContainerAPIDockerClientMock{}
	mockClient.CopyToFileNames = make([]string, 0)

	if err := restoreFunc(z, "", mockClient); err != nil {
		return nil, fmt.Errorf("error returned when calling with valid input: %s", err)
	}

	return mockClient.CopyToFileNames, nil
}

func mockZip(files map[string]string) *zip.Reader {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			logger.Error.Fatal(err)
		}

		_, err = f.Write([]byte(body))
		if err != nil {
			logger.Error.Fatal(err)
		}
	}

	err := w.Close()
	if err != nil {
		logger.Error.Fatal(err)
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		logger.Error.Fatal(err)
	}

	return r
}

func TestSaveHoldQuery(t *testing.T) {
	// command echo and responses are read from the CLI
	logs := bytes.NewReader(
		//nolint:lll // test
		[]byte(`save hold
Saving...
save query
Data saved. Files are now ready to be copied.
Bedrock level/db/MANIFEST-000051:258, Bedrock level/db/000050.ldb:1281520, Bedrock level/db/000053.log:0, Bedrock level/db/000052.ldb:150713, Bedrock level/db/CURRENT:16, Bedrock level/level.dat:2209, Bedrock level/level.dat_old:2209, Bedrock level/levelname.txt:13
`))

	bytes.NewBuffer([]byte{})

	_, err := SaveHoldQuery(
		bytes.NewBuffer([]byte{}),
		bufio.NewReader(logs),
	)
	if err != nil {
		t.Errorf("error returned when calling with valid input: %s", err)
	}
}

func TestSaveResume(t *testing.T) {
	// command echo and responses are read from the CLI
	logs := bytes.NewReader(
		//nolint:lll // test
		[]byte(`save resume
Changes to the level are resumed.
`))

	bytes.NewBuffer([]byte{})

	err := SaveResume(
		bytes.NewBuffer([]byte{}),
		bufio.NewReader(logs),
	)
	if err != nil {
		t.Errorf("error returned when calling with valid input: %s", err)
	}
}

/*func mockTar(path string) *tar.Reader {
	var buf bytes.Buffer

	tw := tar.NewWriter(&buf)

	var files = []struct {
		Name, Body string
	}{
		{path, "some content"},
	}

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			logger.Error.Fatal(err)
		}

		if _, err := tw.Write([]byte(file.Body)); err != nil {
			logger.Error.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		logger.Error.Fatal(err)
	}

	return tar.NewReader(bytes.NewReader(buf.Bytes()))
}*/

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

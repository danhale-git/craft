package backup

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/danhale-git/craft/internal/mock"

	"github.com/docker/docker/client"

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
	mockClient := &mock.ContainerAPIDockerClientMock{}
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

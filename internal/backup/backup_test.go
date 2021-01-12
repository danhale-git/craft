package backup

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"log"
	"testing"
)

func TestRestoreBackup(t *testing.T) {
	got := 0
	copyToFunc := func(string, *bytes.Buffer) error {
		// count the number of files copies
		got++
		return nil
	}

	// zip data and count of zipped files
	z, want := mockZip()

	if err := RestoreBackup(z, copyToFunc); err != nil {
		t.Errorf("error returned when calling with valid input: %s", err)
	}

	if got != want {
		t.Errorf("unexpected count of copyToFunc calls, got %d: want %d", got, want)
	}
}

func mockZip() (*zip.Reader, int) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	var files = []struct {
		Name, Body string
	}{
		{"server.properties", "some content"},
		{"worlds/Bedrock level/db/MANIFEST-000051", "some content"},
		{"worlds/Bedrock level/db/000050.ldb", "some content"},
		{"worlds/Bedrock level/db/000053.log", "some content"},
		{"worlds/Bedrock level/db/000052.ldb", "some content"},
		{"worlds/Bedrock level/db/CURRENT", "some content"},
		{"worlds/Bedrock level/level.dat", "some content"},
		{"worlds/Bedrock level/level.dat_old", "some content"},
		{"worlds/Bedrock level/levelname.txt", "some content"},
	}

	for _, file := range files {
		f, err := w.Create(file.Name)
		if err != nil {
			log.Fatal(err)
		}

		_, err = f.Write([]byte(file.Body))
		if err != nil {
			log.Fatal(err)
		}
	}

	err := w.Close()
	if err != nil {
		log.Fatal(err)
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		log.Fatal(err)
	}

	return r, len(files)
}

func TestTakeBackup(t *testing.T) {
	// file list in the string literal below has 8 paths
	want := 8 + len(serverFiles)
	got := 0

	copyFromFunc := func(p string) (*tar.Reader, error) {
		got++
		return mockTar(p), nil
	}

	// command echo and responses are read from the CLI
	logs := bytes.NewReader(
		[]byte(`save hold
Saving...
save query
Data saved. Files are now ready to be copied.
Bedrock level/db/MANIFEST-000051:258, Bedrock level/db/000050.ldb:1281520, Bedrock level/db/000053.log:0, Bedrock level/db/000052.ldb:150713, Bedrock level/db/CURRENT:16, Bedrock level/level.dat:2209, Bedrock level/level.dat_old:2209, Bedrock level/levelname.txt:13
save resume
Changes to the level are resumed.
`))

	bytes.NewBuffer([]byte{})

	err := TakeBackup(
		bytes.NewBuffer([]byte{}),
		bytes.NewBuffer([]byte{}),
		bufio.NewReader(logs),
		copyFromFunc,
	)
	if err != nil {
		t.Errorf("error returned when calling with valid input: %s", err)
	}

	if got != want {
		t.Errorf("unexpected count of copyFromFunc calls, got %d: want %d", got, want)
	}
}

func mockTar(path string) *tar.Reader {
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
			log.Fatal(err)
		}

		if _, err := tw.Write([]byte(file.Body)); err != nil {
			log.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}

	return tar.NewReader(bytes.NewReader(buf.Bytes()))
}

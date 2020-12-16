package server

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type (
	// ZipTar is some data that can be either a zip or a tar archive
	ZipTar interface {
		Zip() *zip.ReadCloser
		Tar() *bytes.Buffer
	}

	// WorldData is a .mcworld zip archive
	WorldData struct {
		*zip.ReadCloser
	}

	// ServerProperties is the body of a server.properties file
	ServerProperties struct {
		*os.File
	}
)

func (w *WorldData) Zip() *zip.ReadCloser {
	return w.ReadCloser
}

func (w *WorldData) Tar() *bytes.Buffer {
	t, err := zipToTar(w.ReadCloser)
	if err != nil {
		log.Fatalf("Failed converting world data to tar archive.")
	}

	return t
}

func (s *ServerProperties) Zip() *zip.ReadCloser {
	log.Fatal("(s *ServerProperties) Zip() NOT IMPLEMENTED")
	return nil
}

func (s *ServerProperties) Tar() *bytes.Buffer {
	files := make(map[string]*os.File)
	files["server.properties"] = s.File

	t, err := toTar(files)
	if err != nil {
		log.Fatalf("Failed converting world data to tar archive.")
	}

	return t
}

// zipToTar reads each file in a zip archive and writes it to a tar archive. A buffer of the tar archive data
// is returned, or an error.
func zipToTar(r *zip.ReadCloser) (*bytes.Buffer, error) {
	// Create and add files to the archive.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, f := range r.File {
		b, err := f.Open()
		if err != nil {
			return nil, err
		}

		// Read the file body
		body, err := ioutil.ReadAll(b)
		if err != nil {
			return nil, err
		}

		if err = b.Close(); err != nil {
			return nil, err
		}

		// Preserve the file names and permissions in file header
		hdr := &tar.Header{
			Name: f.Name,
			Mode: int64(f.FileInfo().Mode()),
			Size: int64(len(body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		// Write the file body
		if _, err := tw.Write(body); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}

// readZipToTarReader reads each file in a tar archive, writes it to a zip archive and returns the tar archive reader.
func tarToZip(r io.ReadCloser) (*bytes.Buffer, error) {
	tr := tar.NewReader(r)
	defer r.Close()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for {
		// Next file or end of archive
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("calling next() in tar archive: %s", err)
		}

		// The worlds/'Bedrock level' directory was copied. Strip that directory from the file paths to move everything
		// up one level resulting in a valid .mcworld zip.
		name := strings.Replace(hdr.Name, "Bedrock level/", "", 1)
		// Skip the file representing the 'Bedrock level' directory.
		if len(strings.TrimSpace(name)) == 0 {
			continue
		}

		f, err := w.Create(name)
		if err != nil {
			return nil, fmt.Errorf("creating file in zip archive: %s", err)
		}

		b, err := ioutil.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("reading tar data: %s", err)
		}

		if _, err = f.Write(b); err != nil {
			return nil, fmt.Errorf("writing zip data: %s", err)
		}
	}

	err := w.Close()
	if err != nil {
		return nil, fmt.Errorf("closing buffer: %s", err)
	}

	return buf, nil
}

func toTar(files map[string]*os.File) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for name, file := range files {
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}

		hdr := &tar.Header{
			Name: name,
			Mode: int64(info.Mode()),
			Size: info.Size(),
		}
		if err = tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		b, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}

		if _, err := tw.Write(b); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}

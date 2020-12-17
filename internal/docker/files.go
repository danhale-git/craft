package docker

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// Archive is a collection of file names and contents
type Archive struct {
	Files []File
}

// File is a file name, body and mode.
type File struct {
	Name string
	Mode os.FileMode
	Body []byte
}

// Zip returns a zip archive
func (a *Archive) Zip() (*bytes.Buffer, error) {
	return filesToZip(a.Files)
}

// Tar returns a tar archive.
func (a *Archive) Tar() (*bytes.Buffer, error) {
	return filesToTar(a.Files)
}

// FromZip converts a zip archive reader into a slice of File struct preserving the name and body of the file.
func FromZip(r *zip.ReadCloser) (*Archive, error) {
	f, err := zipToFiles(r)
	if err != nil {
		return nil, err
	}

	return &Archive{Files: f}, nil
}

// FromTar converts a tar archive reader into a slice of File struct preserving the name and body of the file.
func FromTar(r *io.Reader) (*Archive, error) {
	f, err := tarToFiles(r)
	if err != nil {
		return nil, err
	}

	return &Archive{Files: f}, nil
}

// FromFiles converts a slice of os.File structs into a slice of File struct preserving the name and body of the file.
func FromFiles(osFiles []*os.File) (*Archive, error) {
	files := make([]File, 0)

	for _, of := range osFiles {
		b, err := ioutil.ReadAll(of)

		if err != nil {
			return nil, err
		}

		files = append(files, File{
			Name: of.Name(),
			Body: b,
		})
	}

	return &Archive{Files: files}, nil
}

func filesToZip(files []File) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for _, f := range files {
		zf, err := w.Create(f.Name)
		if err != nil {
			return nil, err
		}

		if _, err = zf.Write(f.Body); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

func filesToTar(files []File) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	w := tar.NewWriter(buf)

	for _, f := range files {
		err := w.WriteHeader(&tar.Header{
			Name: f.Name,
			Mode: int64(f.Mode),
			Size: int64(len(f.Body)),
		})
		if err != nil {
			return nil, err
		}

		if _, err = w.Write(f.Body); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

func zipToFiles(r *zip.ReadCloser) ([]File, error) {
	files := make([]File, 0)

	for _, f := range r.File {
		b, err := f.Open()
		if err != nil {
			return nil, err
		}

		body, err := ioutil.ReadAll(b)
		if err != nil {
			return nil, err
		}

		if err = b.Close(); err != nil {
			return nil, err
		}

		files = append(files, File{
			Name: f.Name,
			Body: body,
		})
	}

	return files, nil
}

func tarToFiles(r *io.Reader) ([]File, error) {
	tr := tar.NewReader(*r)

	files := make([]File, 0)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("calling next() in tar archive: %s", err)
		}

		b, err := ioutil.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("reading tar data: %s", err)
		}

		files = append(files, File{
			Name: hdr.Name,
			Mode: os.FileMode(hdr.Mode),
			Body: b,
		})
	}

	return files, nil
}

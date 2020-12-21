package files

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
)

// Archive is a collection of file names and contents
type Archive struct {
	Files []File
}

// NewArchiveFromZip converts a zip archive reader into a slice of File struct preserving the name and body of the file.
func NewArchiveFromZip(r *zip.Reader) (*Archive, error) {
	f, err := zipToFiles(r)
	if err != nil {
		return nil, err
	}

	return &Archive{Files: f}, nil
}

// NewArchiveFromTar converts a tar archive reader into a slice of File struct preserving the name and body of the file.
func NewArchiveFromTar(r io.ReadCloser) (*Archive, error) {
	f, err := tarToFiles(r)
	if err != nil {
		return nil, err
	}

	err = r.Close()

	return &Archive{Files: f}, nil
}

// NewArchiveFromFiles converts a slice of os.File structs into an Archive.
func NewArchiveFromFiles(osFiles []*os.File) (*Archive, error) {
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

// File is a file name, body and mode.
type File struct {
	Name string
	Mode os.FileMode
	Body []byte
}

func (a *Archive) AddFile(f File) {
	a.Files = append(a.Files, f)
}

// Save writes all files to disk at the specified path.
func (a *Archive) Save(outDir string) error {
	fileInfo, err := os.Stat(outDir)
	if err != nil {
		return fmt.Errorf("getting file info for '%s': %s", outDir, err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("parameter must be a directory, got %s", outDir)
	}

	for _, f := range a.Files {
		err := save(outDir, f.Name, f.Body)
		if err != nil {
			return fmt.Errorf("writing file '%s': %s", f.Name, err)
		}
	}

	return nil
}

// Save writes all files to disk in a zip archive at the specified path.
func (a *Archive) SaveZip(dirPath, fileName string) error {
	z, err := a.Zip()
	if err != nil {
		return fmt.Errorf("creating zip archive: %s", err)
	}

	err = save(dirPath, fileName, z.Bytes())
	if err != nil {
		return fmt.Errorf("writing file '%s': %s", path.Join(dirPath, fileName), err)
	}

	return nil
}

// Zip returns a zip archive.
func (a *Archive) Zip() (*bytes.Buffer, error) {
	return filesToZip(a.Files)
}

// Tar returns a tar archive.
func (a *Archive) Tar() (*bytes.Buffer, error) {
	return filesToTar(a.Files)
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

func zipToFiles(r *zip.Reader) ([]File, error) {
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

func tarToFiles(r io.Reader) ([]File, error) {
	tr := tar.NewReader(r)

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

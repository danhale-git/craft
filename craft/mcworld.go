package craft

import (
	"archive/zip"
	"fmt"
	"path/filepath"
)

type ZipOpener interface {
	Open() (*zip.ReadCloser, error)
}

type MCWorld struct {
	Path string // The full path to a valid .mcworld zip file
}

func (w MCWorld) Open() (*zip.ReadCloser, error) {
	if err := w.check(); err != nil {
		return nil, fmt.Errorf("invalid .mcworld file: %s", err)
	}

	// Open backup zip
	zr, _ := zip.OpenReader(w.Path)

	return zr, nil
}

func (w MCWorld) check() error {
	expected := []string{
		filepath.Join("db", "CURRENT"),
		"level.dat",
		"levelname.txt",
	}

	results := make(map[string]bool)
	for _, n := range expected {
		results[n] = false
	}

	zr, err := zip.OpenReader(w.Path)
	if err != nil {
		return fmt.Errorf("failed to open zip: %s", err)
	}

	for _, f := range zr.File {
		results[f.Name] = true
	}

	for _, n := range expected {
		if !results[n] {
			return fmt.Errorf("missing expected file '%s'", n)
		}
	}

	if err = zr.Close(); err != nil {
		return fmt.Errorf("failed to close zip: %s", err)
	}

	return nil
}

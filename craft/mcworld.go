package craft

import (
	"archive/zip"
	"fmt"
)

type File struct {
	Name string
	Body []byte
}

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
	expected := map[string]bool{
		"db/CURRENT":    false,
		"level.dat":     false,
		"levelname.txt": false,
	}

	zr, err := zip.OpenReader(w.Path)
	if err != nil {
		return fmt.Errorf("failed to open zip: %s", err)
	}

	for _, f := range zr.File {
		expected[f.Name] = true
	}

	if !(expected["db/CURRENT"] &&
		expected["level.dat"] &&
		expected["levelname.txt"]) {
		return fmt.Errorf("missing one of: db, level.dat, levelname.txt")
	}

	if err = zr.Close(); err != nil {
		return fmt.Errorf("failed to close zip: %s", err)
	}

	return nil
}

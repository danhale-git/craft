package craft

import (
	"archive/zip"
	"fmt"

	"github.com/danhale-git/craft/internal/backup"
	"github.com/danhale-git/craft/internal/docker"
	"github.com/danhale-git/craft/internal/logger"
)

func LoadMCWorldFile(mcworld string, c *docker.Container) error {
	if err := checkWorldFiles(mcworld); err != nil {
		return fmt.Errorf("invalid mcworld file: %s", err)
	}

	// Open backup zip
	zr, err := zip.OpenReader(mcworld)
	if err != nil {
		logger.Panic(err)
	}

	if err = backup.RestoreMCWorld(&zr.Reader, c.CopyTo); err != nil {
		return fmt.Errorf("restoring backup: %s", err)
	}

	if err = zr.Close(); err != nil {
		logger.Panicf("closing zip: %s", err)
	}

	return nil
}

func checkWorldFiles(mcworld string) error {
	expected := map[string]bool{
		"db/CURRENT":    false,
		"level.dat":     false,
		"levelname.txt": false,
	}

	zr, err := zip.OpenReader(mcworld)
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

package craft

import (
	"io/ioutil"

	"github.com/danhale-git/craft/internal/logger"
)

const (
	timeFormat = "02 Jan 2006 3:04PM"
)

// backupServerNames returns a slice with the names of all backed up servers.
func backupServerNames() []string {
	backupDir := backupDirectory()
	infos, err := ioutil.ReadDir(backupDir)

	if err != nil {
		logger.Panicf("reading directory '%s': %s", backupDir, err)
	}

	names := make([]string, 0)

	for _, f := range infos {
		if !f.IsDir() {
			continue
		}

		names = append(names, f.Name())
	}

	return names
}

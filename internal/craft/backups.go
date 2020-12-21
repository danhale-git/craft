package craft

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"
)

func BackupServerNames(backupDir string) ([]string, error) {
	infos, err := ioutil.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory '%s': %s", backupDir, err)
	}

	names := make([]string, len(infos))
	for i, f := range infos {
		names[i] = f.Name()
	}

	return names, nil
}

func LatestServerBackup(serverName, backupDir string) (string, error) {
	infos, err := ioutil.ReadDir(path.Join(backupDir, serverName))
	if err != nil {
		return "", fmt.Errorf("reading directory '%s': %s", backupDir, err)
	}

	var mostRecent time.Time

	var mostRecentFileName string

	for _, f := range infos {
		name := f.Name()

		prefix := fmt.Sprintf("%s_", serverName)
		if strings.HasPrefix(name, prefix) {
			backupTime := strings.Replace(name, prefix, "", 1)
			backupTime = strings.Split(backupTime, ".")[0]

			t, err := time.Parse(backupFilenameTimeLayout, backupTime)
			if err != nil {
				return "", fmt.Errorf("parsing time from file name '%s': %s", name, err)
			}

			if t.After(mostRecent) {
				mostRecent = t
				mostRecentFileName = name
			}
		}
	}

	return mostRecentFileName, nil
}

package files

import (
	"os"
	"path"
	"sort"
)

func save(dirPath, fileName string, body []byte) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(path.Join(dirPath, fileName))
	if err != nil {
		return err
	}

	_, err = f.Write(body)
	if err != nil {
		return err
	}

	return nil
}

type filesByName []os.FileInfo

func (s filesByName) Len() int {
	return len(s)
}
func (s filesByName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s filesByName) Less(i, j int) bool {
	// Put i first
	names := []string{s[i].Name(), s[j].Name()}
	sort.Strings(names)

	// i is still first
	return names[0] == s[i].Name()
}

func SortFilesByName(files []os.FileInfo) {
	sort.Sort(filesByName(files))
}

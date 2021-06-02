package backup

import (
	"os"
	"sort"
)

type filesByName []os.FileInfo

func (s filesByName) Len() int {
	return len(s)
}
func (s filesByName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s filesByName) Less(i, j int) bool {
	it, err := FileTime(s[i].Name())
	if err != nil {
		panic(err)
	}

	jt, err := FileTime(s[j].Name())
	if err != nil {
		panic(err)
	}

	return it.Before(jt)
}

// SortFilesByDate returns a list of all os.FileInfo with valid backup names, sorted by the date the backup was taken.
// The sort order is older files have lower indices.
func SortFilesByDate(files []os.FileInfo) []os.FileInfo {
	// Less can't return an error so ignore invalid files here
	cleanedFiles := make([]os.FileInfo, 0)

	for _, f := range files {
		_, err := FileTime(f.Name())
		if err == nil {
			cleanedFiles = append(cleanedFiles, f)
		}
	}

	sortedFiles := filesByName(cleanedFiles)
	sort.Sort(sortedFiles)

	return sortedFiles
}

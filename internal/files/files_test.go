package files

import (
	"os"
	"testing"
	"time"
)

func TestFilesByName(t *testing.T) {
	files := []os.FileInfo{
		MockFileInfo{
			FileName: "egflksmdglskmd",
		},
		MockFileInfo{
			FileName: "caksajfnalkf",
		},
		MockFileInfo{
			FileName: "daksdmalskmd",
		},
		MockFileInfo{
			FileName: "fglkmrlgkmslfk",
		},
		MockFileInfo{
			FileName: "beatkjnfs",
		},
		MockFileInfo{
			FileName: "asfkgnsfgnj",
		},
	}

	SortFilesByName(files)

	got := ""
	for _, f := range files {
		got += string(f.Name()[0])
	}

	want := "abcdef"

	if got != want {
		t.Errorf("sort strings: want: %s: got %s", want, got)
	}
}

type MockFileInfo struct {
	FileName string
}

// base name of the file
func (mf MockFileInfo) Name() string {
	return mf.FileName
}

// length in bytes for regular files; system-dependent for others
func (mf MockFileInfo) Size() int64 {
	return 0
}

// file mode bits
func (mf MockFileInfo) Mode() os.FileMode {
	return 0
}

// modification time
func (mf MockFileInfo) ModTime() time.Time {
	return time.Time{}
}

// abbreviation for Mode().IsDir()
func (mf MockFileInfo) IsDir() bool {
	return false
}

// underlying data source (can return nil)
func (mf MockFileInfo) Sys() interface{} {
	return nil
}

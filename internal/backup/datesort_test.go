package backup

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestSortFilesByDate(t *testing.T) {
	files := []os.FileInfo{
		MockFileInfo{
			FileName: "test_01-02-2021_21-31.zip",
		},
		MockFileInfo{
			FileName: "test_01-02-2021_21-30.zip",
		},
		MockFileInfo{
			FileName: "test_16-01-2021_11-11.zip",
		},
		MockFileInfo{
			FileName: "test_16-01-2021_11-10.zip",
		},
		MockFileInfo{
			FileName: "test_26-01-2021_21-35.zip",
		},
		MockFileInfo{
			FileName: "test_26-01-2021_21-52.zip",
		},
		MockFileInfo{
			FileName: "test_28-01-2021_20-32.zip",
		},
		MockFileInfo{
			FileName: "invalid28-01-2021_20-32.zip",
		},
	}

	want := []string{
		"test_16-01-2021_11-10.zip",
		"test_16-01-2021_11-11.zip",
		"test_26-01-2021_21-35.zip",
		"test_26-01-2021_21-52.zip",
		"test_28-01-2021_20-32.zip",
		"test_01-02-2021_21-30.zip",
		"test_01-02-2021_21-31.zip",
	}

	got := SortFilesByDate(files)

	for i := range got {
		if got[i].Name() != want[i] {
			t.Fatalf("unexpected value at index %d:\n%s", i, strings.Join(want, "\n"))
		}
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

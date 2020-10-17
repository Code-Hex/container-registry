package registry

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/h2non/filetype"
)

// BasePath represents base path for this application.
var BasePath = "testdata"

// PathJoinWithBase joins any number of path elements with base path into a single path,
// separating them with an OS specific Separator.
func PathJoinWithBase(name string, p ...string) string {
	return filepath.Join(
		append(
			[]string{
				BasePath,
				name,
			},
			p...,
		)...,
	)
}

// CreateLayer creates layer a file which will be json or gz extension on specified path.
func CreateLayer(r io.Reader, path string) (int64, error) {
	// see filetype.MatchReader
	buffer := make([]byte, 8192)
	n, err := r.Read(buffer)
	if err != nil && err != io.EOF {
		return 0, err
	}

	filePath := filepath.Join(path, "layer"+detectExt(buffer))
	f, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, io.MultiReader(bytes.NewReader(buffer[:n]), r))
}

func detectExt(buf []byte) string {
	if filetype.IsArchive(buf) {
		return ".tar.gz"
	}
	return ".json"
}

// PickupFileinfo picks up one file in the specified directory.
// This function is expected to use if there's only one file in the directory.
func PickupFileinfo(dir string) (os.FileInfo, error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(fis) == 0 {
		return nil, fmt.Errorf("there is no file in %q directory", dir)
	}
	return fis[0], nil
}

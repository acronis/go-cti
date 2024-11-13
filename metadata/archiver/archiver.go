package archiver

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

var (
	SkipFile error = errors.New("skip this file")
	SkipDir  error = filepath.SkipDir
)

type Archiver interface {
	Init(dst string) (io.Closer, error)
	WriteBytes(fName string, buf []byte) error
	WriteFile(baseDir string, fName string) error
	WriteDirectory(baseDir string, excludeFn func(fsPath string, d os.DirEntry) error) error
}

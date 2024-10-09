package packer

import (
	"io"
)

type Writer interface {
	Init(dst string) (io.Closer, error)
	WriteBytes(fName string, buf []byte) error
	WriteFile(baseDir string, fName string) error
	WriteDirectory(baseDir string, excludeFn func(dirName string, fName string) bool) error
}

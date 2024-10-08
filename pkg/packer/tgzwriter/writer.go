package tgzwriter

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

type tarWriter struct {
	archive *os.File
	gw      *gzip.Writer
	tw      *tar.Writer
}

func New() *tarWriter {
	return &tarWriter{}
}

func (wr *tarWriter) Close() error {
	wr.gw.Close()
	wr.tw.Close()
	wr.archive.Close()
	return nil
}

func (wr *tarWriter) Init(destination string) (io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(destination), os.ModePerm); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	archive, err := os.Create(destination)
	if err != nil {
		return nil, fmt.Errorf("create archive: %w", err)
	}
	wr.gw = gzip.NewWriter(archive)
	wr.tw = tar.NewWriter(wr.gw)

	return wr, nil
}

func (wr *tarWriter) WriteFile(baseDir string, fName string) error {
	f, err := os.OpenFile(filepath.Join(baseDir, fName), os.O_RDONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open serialized metadata %s: %w", fName, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("get file info: %w", err)
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return fmt.Errorf("create file info header: %w", err)
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory structure would not be preserved
	// https://golang.org/src/archive/tar/common.go?#L626
	header.Name = fName

	// Write file header to the tar archive
	if err := wr.tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write file header: %w", err)
	}

	// Copy file content to tar archive
	_, err = io.Copy(wr.tw, f)
	if err != nil {
		return err
	}

	return nil
}

func (wr *tarWriter) WriteBytes(fName string, buf []byte) error {
	// Create a new file header
	tarHeader := &tar.Header{
		Name:     fName,
		Size:     int64(len(buf)),
		Mode:     0o644,
		Typeflag: tar.TypeReg,
	}

	// Write file header to the tar archive
	if err := wr.tw.WriteHeader(tarHeader); err != nil {
		return fmt.Errorf("write file header: %w", err)
	}

	// Copy file content to tar archive
	if _, err := wr.tw.Write(buf); err != nil {
		return fmt.Errorf("write file content: %w", err)
	}
	return nil
}

func (wr *tarWriter) WriteDirectory(baseDir string, excludeFn func(dirName string, fName string) bool) error {
	if err := filepath.WalkDir(baseDir, func(fsPath string, d os.DirEntry, err error) error {
		rel, err := filepath.Rel(baseDir, fsPath)
		if err != nil {
			return fmt.Errorf("walk directory: %w", err)
		}

		if rel == "." || rel == "" || d.IsDir() {
			return nil
		}

		if excludeFn != nil && excludeFn(path.Dir(rel), path.Base(rel)) {
			return nil
		}

		return wr.WriteFile(baseDir, rel)
	}); err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}
	return nil
}

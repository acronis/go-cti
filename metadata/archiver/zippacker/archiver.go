package zippacker

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/metadata/archiver"
)

type zipWriter struct {
	zip.Writer
	archive *os.File
}

func New() *zipWriter {
	return &zipWriter{}
}

func (zipWriter *zipWriter) Close() error {
	zipWriter.Writer.Close()
	zipWriter.archive.Close()
	return nil
}

func (zipWriter *zipWriter) Init(destination string) (io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(destination), os.ModePerm); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	archive, err := os.Create(destination)
	if err != nil {
		return nil, fmt.Errorf("create archive: %w", err)
	}
	zipWriter.Writer = *zip.NewWriter(archive)

	return zipWriter, nil
}

func (zipWriter *zipWriter) WriteFile(baseDir string, metadata string) error {
	f, err := os.OpenFile(filepath.Join(baseDir, metadata), os.O_RDONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open serialized metadata %s: %w", metadata, err)
	}
	defer f.Close()

	w, err := zipWriter.Create(metadata)
	if err != nil {
		return fmt.Errorf("create serialized metadata %s in package: %w", metadata, err)
	}
	if _, err = io.Copy(w, f); err != nil {
		return fmt.Errorf("write serialized metadata %s to package: %w", metadata, err)
	}
	return nil
}

func (zipWriter *zipWriter) WriteBytes(fName string, buf []byte) error {
	w, err := zipWriter.Create(fName)
	if err != nil {
		return fmt.Errorf("file in archive: %w", err)
	}

	if _, err = w.Write(buf); err != nil {
		return fmt.Errorf("write bytes into file: %w", err)
	}

	return nil
}

func (zipWriter *zipWriter) WriteDirectory(baseDir string, excludeFn func(fsPath string, d os.DirEntry) error) error {
	baseDir = filepath.ToSlash(baseDir)
	if !strings.HasSuffix(baseDir, "/") {
		baseDir += "/"
	}

	if err := filepath.WalkDir(baseDir, func(fsPath string, d os.DirEntry, err error) error {
		rel, err := filepath.Rel(baseDir, fsPath)
		if err != nil {
			return fmt.Errorf("walk directory: %w", err)
		}

		if rel == "." || rel == "" || d.IsDir() {
			return nil
		}

		if excludeFn != nil {
			switch excludeFn(fsPath, d) {
			case archiver.SkipDir:
				return filepath.SkipDir
			case archiver.SkipFile:
				return nil
			}
		}

		f, err := os.OpenFile(fsPath, os.O_RDONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open index: %w", err)
		}
		w, err := zipWriter.Writer.Create(rel)
		if err != nil {
			return fmt.Errorf("create file in archive: %w", err)
		}
		if _, err = io.Copy(w, f); err != nil {
			return fmt.Errorf("copy file into archive: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}
	return nil
}

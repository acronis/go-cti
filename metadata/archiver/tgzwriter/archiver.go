package tgzwriter

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/metadata/archiver"
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
	if err := wr.tw.Close(); err != nil {
		return err
	}
	if err := wr.gw.Close(); err != nil {
		return err
	}
	return wr.archive.Close()
}

func (wr *tarWriter) Init(destination string) (io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(destination), os.ModePerm); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	archive, err := os.Create(destination)
	if err != nil {
		return nil, fmt.Errorf("create archive: %w", err)
	}
	wr.archive = archive
	wr.gw = gzip.NewWriter(wr.archive)
	wr.tw = tar.NewWriter(wr.gw)

	return wr, nil
}

func (wr *tarWriter) WriteFile(baseDir string, fName string) error {
	filePath := filepath.Join(baseDir, fName)
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("get file info: %w", err)
	}

	var link string
	if info.Mode()&os.ModeSymlink != 0 {
		if link, err = os.Readlink(filePath); err != nil {
			return fmt.Errorf("read link: %w", err)
		}
	}

	if link != "" {
		slog.Warn("Symlink found", slog.String("link", link), slog.String("path", filePath))
	}

	header, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return fmt.Errorf("create file info header: %w", err)
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory structure would not be preserved
	// https://golang.org/src/archive/tar/common.go?#L626
	header.Name = filepath.ToSlash(fName)

	// Write file header to the tar archive
	if err := wr.tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write file header: %w", err)
	}

	f, err := os.OpenFile(filePath, os.O_RDONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open serialized metadata %s: %w", fName, err)
	}
	defer f.Close()

	if _, err = io.Copy(wr.tw, f); err != nil {
		slog.Error("Failed to write file content into archive",
			slog.String("file", fName),
			slog.Any("info", info),
			slog.Any("header", header))

		return fmt.Errorf("write file content into archive: %w", err)
	}

	return nil
}

func (wr *tarWriter) WriteBytes(fName string, buf []byte) error {
	// Create a new file header
	tarHeader := &tar.Header{
		Name:     filepath.ToSlash(fName),
		Size:     int64(len(buf)),
		Mode:     0600,
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

func (wr *tarWriter) WriteDirectory(baseDir string, excludeFn func(fsPath string, d os.DirEntry) error) error {
	baseDir = filepath.ToSlash(baseDir)
	if !strings.HasSuffix(baseDir, "/") {
		baseDir += "/"
	}
	destinationInfo, err := wr.archive.Stat()
	if err != nil {
		return fmt.Errorf("get archive info: %w", err)
	}

	if err := filepath.WalkDir(baseDir, func(fsPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip archive file itself, avoid recursive archiving
		if !d.IsDir() {
			fInfo, err := d.Info()
			if err != nil {
				return fmt.Errorf("get file info: %w", err)
			}

			if os.SameFile(fInfo, destinationInfo) {
				slog.Debug("Skip archive file to avoid recursion", slog.String("path", fsPath))
				return nil
			}
		}

		rel, err := filepath.Rel(baseDir, fsPath)
		if err != nil {
			return fmt.Errorf("get relative path: %w", err)
		}

		// skip the base directory itself
		if rel == "." {
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

		if !d.IsDir() {
			return wr.WriteFile(baseDir, rel)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}
	return nil
}

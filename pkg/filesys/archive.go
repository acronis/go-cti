package filesys

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func OpenTarFile(source string, fpath string) ([]byte, error) {
	reader, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("open tar file: %w", err)
	}
	defer reader.Close()

	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar header: %w", err)
		}

		if header.Name != fpath {
			continue
		}

		if header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("specified file path %s is not a regular file", fpath)
		}

		return io.ReadAll(tr)
	}

	return nil, fmt.Errorf("failed to find %s in archive", fpath)
}

func OpenZipFile(source string, fpath string) ([]byte, error) {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return nil, fmt.Errorf("open zip file: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != fpath {
			continue
		}

		if file.FileInfo().IsDir() {
			return nil, fmt.Errorf("specified file path %s is a directory", fpath)
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open file in archive: %w", err)
		}
		defer rc.Close()

		return io.ReadAll(rc)
	}

	return nil, fmt.Errorf("failed to find %s in archive", fpath)
}

func sanitizeAndValidatePath(dest string, src string) (string, error) {
	// Sanitize the file name and remove any dangerous characters
	filePath := filepath.Join(dest, filepath.Clean(src))

	// Ensure file paths don't escape the target directory using filepath.Rel for strict comparison
	relPath, err := filepath.Rel(dest, filePath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("invalid file path: %s", filePath)
	}
	return filePath, nil
}

// Secure unzip function
func SecureUnzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open zip file: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Sanitize the file name and remove any dangerous characters
		filePath, err := sanitizeAndValidatePath(dest, f.Name)
		if err != nil {
			return fmt.Errorf("sanitize file path: %w", err)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
			continue
		}

		// Limit file size to prevent DoS attacks
		const maxFileSize = 100 << 20 // 100 MB
		if f.UncompressedSize64 > maxFileSize {
			return fmt.Errorf("file too large: %s", f.Name)
		}

		// Create destination file securely
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		destFile, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		defer destFile.Close()

		srcFile, err := f.Open()
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer srcFile.Close()

		if _, err := io.CopyN(destFile, srcFile, maxFileSize); err != nil {
			return fmt.Errorf("copy file: %w", err)
		}
	}
	return nil
}

// Secure untar function
func SecureUntar(src string, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open tar file: %w", err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		// Sanitize the file name and remove any dangerous characters
		fPath, err := sanitizeAndValidatePath(dest, header.Name)
		if err != nil {
			return fmt.Errorf("sanitize file path: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fPath, os.ModePerm); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
		case tar.TypeReg:
			// Limit file size to prevent DoS attacks
			const maxFileSize = 100 << 20 // 100 MB
			if header.Size > maxFileSize {
				return fmt.Errorf("file too large: %s", header.Name)
			}

			// Create destination file securely
			if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}

			destFile, err := os.Create(fPath)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			defer destFile.Close()

			if _, err := io.CopyN(destFile, tr, maxFileSize); err != nil {
				return fmt.Errorf("copy file: %w", err)
			}
		}
	}
	return nil
}

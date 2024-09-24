package filesys

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type CopyConfig struct {
	RootDir      string // Root directory to copy
	Overwrite    bool   // Override existing files
	PreservePerm bool   // Preserve file permissions
}

type CopyOption func(*CopyConfig)

func WithRoot(root string) CopyOption {
	return func(cfg *CopyConfig) {
		cfg.RootDir = root
	}
}

func WithOverwrite(overwrite bool) CopyOption {
	return func(cfg *CopyConfig) {
		cfg.Overwrite = overwrite
	}
}

func WithPreservePermissions(preserve bool) CopyOption {
	return func(cfg *CopyConfig) {
		cfg.PreservePerm = preserve
	}
}

// copyFS copies files and directories from srcFS to destRoot securely.
func CopyFS(srcFS fs.FS, destRoot string, options ...CopyOption) error {
	config := &CopyConfig{
		RootDir:      ".",
		Overwrite:    false,
		PreservePerm: false,
	}

	for _, opt := range options {
		opt(config)
	}

	if !fs.ValidPath(config.RootDir) {
		return fmt.Errorf("invalid root path: %s", config.RootDir)
	}

	// Ensure the destination root is absolute and clean
	absDestRoot, err := filepath.Abs(destRoot)
	if err != nil {
		return fmt.Errorf("get absolute path of destination root: %w", err)
	}

	if config.Overwrite {
		if err := os.RemoveAll(absDestRoot); err != nil {
			return fmt.Errorf("remove destination root: %w", err)
		}
	}

	// Walk the source filesystem
	return fs.WalkDir(srcFS, config.RootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing %s: %w", path, err)
		}

		// Skip symbolic links
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		relPath, err := filepath.Rel(config.RootDir, path)
		if err != nil {
			return fmt.Errorf("get relative path: %w", err)
		}

		// Validate the path to prevent path traversal
		if !fs.ValidPath(path) {
			return fmt.Errorf("invalid path: %s", path)
		}

		// Compute the destination path
		destPath := filepath.Join(absDestRoot, filepath.FromSlash(relPath))

		// Ensure the destination path is within the destination root
		absDestPath, err := filepath.Abs(destPath)
		if err != nil {
			return fmt.Errorf("get absolute path of destination: %w", err)
		}

		if !strings.HasPrefix(absDestPath, absDestRoot+string(os.PathSeparator)) && absDestPath != absDestRoot {
			return fmt.Errorf("path traversal detected: %s is outside of %s", absDestPath, absDestRoot)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("get filesystem entry info for %s: %w", path, err)
		}

		if info.IsDir() {
			if err := os.MkdirAll(destPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("create destination directory %s: %w", destPath, err)
			}
		} else {
			if err := CopyFile(srcFS, path, destPath, info.Mode()); err != nil {
				return fmt.Errorf("copy file: %w", err)
			}
		}

		return nil
	})
}

// copyFile copies a single file from srcFS to destPath, preserving permissions.
func CopyFile(srcFS fs.FS, srcPath, destPath string, perm fs.FileMode) error {
	// Open the source file
	srcFile, err := srcFS.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	// Create the destination file with the same permissions
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm.Perm())
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy the file contents
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copy data to %s: %w", destPath, err)
	}

	return nil
}

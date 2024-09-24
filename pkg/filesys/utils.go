package filesys

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

const (
	AppEnvironVar = "CTIROOT"
	AppUserDir    = ".cti"
)

// GetBaseName Get filename without extension.
func GetBaseName(fileName string) string {
	filename := path.Base(fileName)

	return filename[:len(filename)-len(filepath.Ext(filename))]
}

func GetDirName(filePath string) string {
	return filepath.Base(filepath.Dir(filePath))
}

func GetRootDir() (string, error) {
	rootDir := os.Getenv(AppEnvironVar)
	if rootDir == "" {
		userDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		rootDir = filepath.Join(userDir, AppUserDir)
	}
	if _, err := os.Stat(rootDir); err != nil {
		err := os.Mkdir(rootDir, 0755)
		if err != nil {
			return "", fmt.Errorf("create root dir: %w", err)
		}
	}
	return rootDir, nil
}

func GetCtiBundlesCacheDir() (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", fmt.Errorf("get root dir: %w", err)
	}
	pkgCacheDir := filepath.Join(rootDir, "src")
	if _, err := os.Stat(pkgCacheDir); err != nil {
		if err := os.Mkdir(pkgCacheDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("create package cache dir: %w", err)
		}
	}
	return pkgCacheDir, nil
}

func CollectFilesWithExt(root, ext string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(root, func(file string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(dir.Name()) == ext {
			files = append(files, file)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}
	return files, nil
}

package pacman

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	AppEnvironVar = "CTIROOT"
	AppUserDir    = ".cti"
)

/*
  .cache/
		source/
			.cti-<random>/ - temporary cache directory for the downloader
			<name>/ - source cache directory (could include subdirectories, e.g. github.com/acronis/cti)
				@v/ - version cache directory
					<version>.info - integrity info
		package/
			<package id>/ - package cache directory
				@v/ - version cache directory
					<version>.index.json - index file
					<version>.info - integrity info
	<package id>/
		@<version>/ - package directory
*/

func (pm *packageManager) getSourceCacheDir() string {
	return filepath.Join(pm.PackagesDir, ".cache", "source")
}

func (pm *packageManager) getPackageCacheDir() string {
	return filepath.Join(pm.PackagesDir, ".cache", "package")
}

func (pm *packageManager) getPackageDir(pkgId string, version string) string {
	return filepath.Join(pm.PackagesDir, pkgId, "@"+version)
}

func (pm *packageManager) getSourceInfoPath(name string, version string) string {
	return filepath.Join(pm.getSourceCacheDir(), name, "@v", version+".info")
}

func (pm *packageManager) getPackageInfoPath(pkgId string, version string) string {
	return filepath.Join(pm.getPackageCacheDir(), pkgId, "@v", version+".info")
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

func GetCtiPackagesCacheDir() (string, error) {
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

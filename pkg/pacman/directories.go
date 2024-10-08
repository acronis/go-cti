package pacman

import "path/filepath"

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

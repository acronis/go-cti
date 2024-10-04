package depman

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

func (dm *dependencyManager) getSourceCacheDir() string {
	return filepath.Join(dm.PackagesDir, ".cache", "source")
}

func (dm *dependencyManager) getPackageCacheDir() string {
	return filepath.Join(dm.PackagesDir, ".cache", "package")
}

func (dm *dependencyManager) getPackageDir(pkgId string, version string) string {
	return filepath.Join(dm.PackagesDir, pkgId, "@"+version)
}

func (dm *dependencyManager) getSourceInfoPath(name string, version string) string {
	return filepath.Join(dm.getSourceCacheDir(), name, "@v", version+".info")
}

func (dm *dependencyManager) getPackageInfoPath(pkgId string, version string) string {
	return filepath.Join(dm.getPackageCacheDir(), pkgId, "@v", version+".info")
}

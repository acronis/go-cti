package depman

import "path/filepath"

/*
  .cache/
		source/
			.cti-<random>/ - temporary cache directory for the downloader
			<name>/ - source cache directory (could include subdirectories, e.g. github.com/acronis/cti)
				@v/ - version cache directory
					<version>.info - integrity info
		bundle/
			<app_code>/ - bundle cache directory
				@v/ - version cache directory
					<version>.index.json - index file
					<version>.info - integrity info
	<code>/
		@<version>/ - bundle directory
*/

func (dm *dependencyManager) getSourceCacheDir() string {
	return filepath.Join(dm.BundlesDir, ".cache", "source")
}

func (dm *dependencyManager) getBundleCacheDir() string {
	return filepath.Join(dm.BundlesDir, ".cache", "bundle")
}

func (dm *dependencyManager) getBundleDir(appCode string, version string) string {
	return filepath.Join(dm.BundlesDir, appCode, "@"+version)
}

func (dm *dependencyManager) getSourceInfoPath(name string, version string) string {
	return filepath.Join(dm.getSourceCacheDir(), name, "@v", version+".info")
}

func (dm *dependencyManager) getBundleInfoPath(code string, version string) string {
	return filepath.Join(dm.getBundleCacheDir(), code, "@v", version+".info")
}

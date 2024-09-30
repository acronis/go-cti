package depman

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/filesys"
)

func (dm *dependencyManager) downloadDependency(source, version string) (CachedDependencyInfo, error) {
	downloadFn, info, err := dm.Downloader.Discover(source, version)
	if err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("discover source %s version %s: %w", source, version, err)
	}

	slog.Info("Discovered dependency", slog.String("bundle", source))

	// Pre-download integrity check
	if err := dm.validateSourceInformation(source, version, info); err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("check integrity: %w", err)
	}

	// Download into temporary directory
	sourceCacheDir := dm.getSourceCacheDir()
	if err := os.MkdirAll(sourceCacheDir, os.ModePerm); err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("create source cache dir: %w", err)
	}

	cacheDir, err := os.MkdirTemp(sourceCacheDir, ".cti-")
	if err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(cacheDir)

	depDir, err := downloadFn(cacheDir)
	if err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("download bundle: %w", err)
	}

	depIdx, err := bundle.ReadIndex(depDir)
	if err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("read index.json: %w", err)
	}

	// Check bundle integrity and register bundle
	if err := dm.updateDependencyCache(source, version, info, depDir, depIdx); err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("update dependency cache: %w", err)
	}

	// Move bundle to the final destination
	targetDir := dm.getBundleDir(depIdx.AppCode, version)
	if err := filesys.ReplaceWithMove(depDir, targetDir); err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("move bundle %s from source %s: %w", depIdx.AppCode, source, err)
	}

	// Patch links
	if err := patchRelativeLinks(targetDir); err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("patch dependency links: %w", err)
	}

	// TODO hmm... probably do not parse it again, just patch the index
	movedIndex, err := bundle.ReadIndex(targetDir)
	if err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("read index.json: %w", err)
	}

	hash, err := filesys.ComputeDirectoryHash(targetDir)
	if err != nil {
		return CachedDependencyInfo{}, fmt.Errorf("compute directory hash: %w", err)
	}

	return CachedDependencyInfo{
		Path:      targetDir,
		Source:    source,
		Version:   version,
		Integrity: hash,
		Index:     *movedIndex,
	}, nil
}

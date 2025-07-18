package pacman

import (
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/acronis/go-cti/metadata/storage"
	"github.com/acronis/go-cti/metadata/storage/gitstorage"
	"github.com/blang/semver/v4"
)

type PackageManager interface {
	// Add new dependencies to index-lock
	Add(pkg *ctipackage.Package, depends map[string]string) error
	// Install dependencies from index or index-lock, force will ignore index-lock and install dependencies from index
	Install(pkg *ctipackage.Package, force bool) error
	// Download dependencies and their sub-dependencies
	Download(depends map[string]string, recursive bool) ([]CachedDependencyInfo, error)
}

type Option func(*packageManager)

type packageManager struct {
	PackagesDir string
	Storage     storage.Storage
}

func New(options ...Option) (PackageManager, error) {
	pm := &packageManager{}

	for _, o := range options {
		o(pm)
	}

	if pm.Storage == nil {
		pm.Storage = gitstorage.New()
	}
	if pm.PackagesDir == "" {
		cacheDir, err := GetCtiPackagesCacheDir()
		if err != nil {
			return nil, fmt.Errorf("get cache dir: %w", err)
		}
		pm.PackagesDir = cacheDir
	}

	return pm, nil
}

func WithStorage(st storage.Storage) Option {
	return func(pm *packageManager) {
		pm.Storage = st
	}
}

func WithPackagesCache(cacheDir string) Option {
	return func(pm *packageManager) {
		pm.PackagesDir = cacheDir
	}
}

func addInstalledDepends(lock *ctipackage.IndexLock, depends []CachedDependencyInfo) *ctipackage.IndexLock {
	for _, info := range depends {
		lock.Depends[info.Index.PackageID] = info.Source
		lock.DependsInfo[info.Source] = ctipackage.Info{
			PackageID: info.Index.PackageID,
			Version:   info.Version.String(),
			Integrity: info.Integrity,
			Source:    info.Source,
			Depends:   info.Index.Depends,
		}
	}
	return lock
}

func (pm *packageManager) Add(pkg *ctipackage.Package, depends map[string]string) error {
	if err := pkg.Sync(); err != nil {
		return fmt.Errorf("sync package: %w", err)
	}

	// Validate dependencies
	newDeps, err := pm.installDependencies(pkg.BaseDir, depends)
	if err != nil {
		return fmt.Errorf("install dependencies: %w", err)
	}

	// update index
	for source, version := range depends {
		if oldVersion, ok := pkg.Index.Depends[source]; ok {
			slog.Info("Dependency already exists in index, updating version",
				slog.String("package", source),
				slog.String("version", oldVersion+" -> "+version),
			)
		} else {
			slog.Info("Adding new dependency to index",
				slog.String("package", source),
				slog.String("version", version),
			)
		}

		pkg.Index.Depends[source] = version
	}

	lock := addInstalledDepends(pkg.IndexLock, newDeps)
	if err := pkg.SaveIndex(); err != nil {
		return fmt.Errorf("save index: %w", err)
	}

	if err := pkg.SaveIndexLock(lock); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}

	return nil
}

func (pm *packageManager) Install(pkg *ctipackage.Package, force bool) error {
	if err := pkg.Sync(); err != nil {
		return fmt.Errorf("sync package: %w", err)
	}

	if force {
		slog.Info("Installing dependencies from index, ignoring index-lock",
			slog.String("package_id", pkg.Index.PackageID),
			slog.Any("depends", pkg.Index.Depends),
		)

		installed, err := pm.installDependencies(pkg.BaseDir, pkg.Index.Depends)
		if err != nil {
			return fmt.Errorf("install index dependencies: %w", err)
		}

		// Create new index-lock from installed dependencies
		lock := addInstalledDepends(ctipackage.NewIndexLock(), installed)

		if err := pkg.SaveIndexLock(lock); err != nil {
			return fmt.Errorf("save index lock: %w", err)
		}
		return nil
	}

	// Check index-lock
	if pkg.IndexLock == nil {
		slog.Error("Index-lock is not found, please run 'pkg tidy' to update index-lock")
		return fmt.Errorf("package index lock missing")
	}

	if !pkg.Index.CompareHash(pkg.IndexLock.Hash) {
		slog.Error("Package index hash mismatch, please run 'pkg tidy' to update index-lock",
			slog.String("package_id", pkg.Index.PackageID),
		)
		return fmt.Errorf("package index hash mismatch")
	}

	slog.Info("Installing dependencies from index-lock",
		slog.String("package_id", pkg.Index.PackageID),
		slog.Any("depends", pkg.IndexLock.DependsInfo),
	)

	// install dependencies from index-lock but do not update it
	if _, err := pm.installDependenciesInfo(pkg.BaseDir, pkg.IndexLock.DependsInfo); err != nil {
		return fmt.Errorf("install index-lock dependencies: %w", err)
	}
	return nil
}

func (pm *packageManager) download(depends map[string]string, recursive bool, installed map[string]CachedDependencyInfo) ([]CachedDependencyInfo, error) {
	subDepends := map[string]string{}

	current := make(map[string]CachedDependencyInfo)
	for source, ver := range depends {
		info, err := pm.downloadDependency(source, ver)
		if err != nil {
			return nil, fmt.Errorf("download dependency %s %s: %w", source, ver, err)
		}

		installed[source] = info
		current[source] = info
	}

	// Add sub-dependencies to resolve
	for source, info := range current {
		for s, v := range info.Index.Depends {
			version, err := semver.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("parse package %s version %s: %w", s, v, err)
			}

			if existing, ok := installed[s]; ok {
				// Compare versions and keep the latest one
				if version.LE(existing.Version) {
					slog.Info("Found lower or equal version of dependency",
						slog.String("_pkg", s),
						slog.String("_ver", existing.Version.String()),
						slog.String("new", v),
						slog.String("origin", source),
					)
					continue
				}

				// TODO: add major version check

				slog.Info("Found greater dependency version",
					slog.String("_pkg", s),
					slog.String("_ver", existing.Version.String()),
					slog.String("new", v),
					slog.String("origin", source),
				)

			} else {
				slog.Info("AFound new dependency",
					slog.String("_pkg", s),
					slog.String("_ver", v),
					slog.String("origin", source),
				)
			}

			subDepends[s] = v
		}
	}

	// Collect sub-dependencies
	if recursive && len(subDepends) > 0 {
		slog.Info("Found sub-dependencies",
			slog.Any("sub-depends", subDepends),
		)

		if _, err := pm.download(subDepends, recursive, installed); err != nil {
			return nil, fmt.Errorf("download sub-dependencies: %w", err)
		}
	}

	res := []CachedDependencyInfo{}
	for _, dep := range installed {
		res = append(res, dep)
	}
	return res, nil
}

func (pm *packageManager) Download(depends map[string]string, recursive bool) ([]CachedDependencyInfo, error) {
	return pm.download(depends, recursive, map[string]CachedDependencyInfo{})
}

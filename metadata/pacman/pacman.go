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
	Download(depends map[string]string) ([]CachedDependencyInfo, error)
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

func (pm *packageManager) Add(pkg *ctipackage.Package, depends map[string]string) error {
	// Validate dependencies
	if err := pm.installDependencies(pkg, depends, true); err != nil {
		return fmt.Errorf("install dependencies: %w", err)
	}

	for source, version := range depends {
		if _, ok := pkg.Index.Depends[source]; ok {
			slog.Info("Added direct dependency", slog.String("package", source), slog.String("version", version))
			pkg.Index.Depends[source] = version
		}
		// TODO check if depends version were updated
		// is possible?
	}

	if err := pkg.SaveIndex(); err != nil {
		return fmt.Errorf("save index: %w", err)
	}

	if err := pkg.SaveIndexLock(); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}

	return nil
}

func (pm *packageManager) Install(pkg *ctipackage.Package, force bool) error {
	useIndex := force

	// Check if index-lock exists and has the same hash as index
	if pkg.IndexLock != nil && pkg.Index.Hash() != pkg.IndexLock.Hash {
		if !force {
			slog.Error("Package index hash mismatch, please run 'pkg tidy' to update index-lock",
				slog.String("package_id", pkg.Index.PackageID),
			)
			return fmt.Errorf("package index hash mismatch")
		}

		slog.Warn("Package index hash mismatch, updating index-lock",
			slog.String("package_id", pkg.Index.PackageID),
		)
		useIndex = true
	}

	if useIndex {
		slog.Info("Installing dependencies from index",
			slog.String("package_id", pkg.Index.PackageID),
			slog.Any("depends", pkg.Index.Depends),
		)

		if err := pm.installDependencies(pkg, pkg.Index.Depends, true); err != nil {
			return fmt.Errorf("install index dependencies: %w", err)
		}

		if err := pkg.SaveIndexLock(); err != nil {
			return fmt.Errorf("save index lock: %w", err)
		}
	} else {
		// collect depends from index-lock
		depends := map[string]string{}
		for name, source := range pkg.IndexLock.DependsInfo {
			depends[name] = source.Version
		}

		slog.Info("Installing dependencies from index-lock",
			slog.String("package_id", pkg.Index.PackageID),
			slog.Any("depends", depends),
		)

		// TODO check if index-lock is correct
		return pm.installDependencies(pkg, depends, false)
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

func (pm *packageManager) Download(depends map[string]string) ([]CachedDependencyInfo, error) {
	return pm.download(depends, true, map[string]CachedDependencyInfo{})
}

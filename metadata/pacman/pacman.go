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
	// Add new dependencies to index.lock
	Add(pkg *ctipackage.Package, depends map[string]string) error
	// Install dependencies from index.lock
	Install(pkg *ctipackage.Package) error
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
	if err := pm.installDependencies(pkg, depends); err != nil {
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

func (pm *packageManager) Install(pkg *ctipackage.Package) error {
	if err := pm.installDependencies(pkg, pkg.Index.Depends); err != nil {
		return fmt.Errorf("install index dependencies: %w", err)
	}
	if err := pkg.SaveIndexLock(); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}
	return nil
}

func (pm *packageManager) download(depends map[string]string, installed []CachedDependencyInfo) ([]CachedDependencyInfo, error) {
	subDepends := map[string]string{}
	for source, version := range depends {
		info, err := pm.downloadDependency(source, version)
		if err != nil {
			return nil, fmt.Errorf("download dependency %s %s: %w", source, version, err)
		}

		installed = append(installed, info)
		// TODO check for cyclic dependencies or duplicates
		for subSource, subTag := range info.Index.Depends {
			installedDep := func() CachedDependencyInfo {
				for _, info := range installed {
					if info.Source == subSource {
						return info
					}
				}
				return CachedDependencyInfo{}
			}()
			if installedDep.Source != "" {
				slog.Info("Dependency already installed",
					slog.String("source", source),
					slog.String("package", subSource),
					slog.String("version", subTag))

				// compare versions
				installedVers, err := semver.Parse(installedDep.Version)
				if err != nil {
					return nil, fmt.Errorf("parse installed version %s: %w", installedDep.Version, err)
				}
				depVers, err := semver.Parse(subTag)
				if err != nil {
					return nil, fmt.Errorf("parse dependency version %s: %w", subTag, err)
				}

				if installedVers.LT(depVers) {
					slog.Info("Installed version is older, update",
						slog.String("source", source),
						slog.String("package", subSource),
						slog.String("installed", installedDep.Version),
						slog.String("dependency", subTag))
				} else {
					logText := func() string {
						if installedVers.GT(depVers) {
							return "newer"
						}
						return "the same"
					}()

					slog.Info(fmt.Sprintf("Installed version is %s, skip", logText),
						slog.String("source", source),
						slog.String("package", subSource),
						slog.String("installed", installedDep.Version),
						slog.String("dependency", subTag))
					continue
				}
			}

			subDepends[subSource] = subTag
		}
	}

	// Recursively download sub-dependencies
	if len(subDepends) != 0 {
		slog.Info("Download sub-dependencies")
		inst, err := pm.download(subDepends, installed)
		if err != nil {
			return nil, fmt.Errorf("download sub-dependencies: %w", err)
		}
		installed = inst
	}

	return installed, nil
}

func (pm *packageManager) Download(depends map[string]string) ([]CachedDependencyInfo, error) {
	return pm.download(depends, []CachedDependencyInfo{})
}

package depman

import (
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/pkg/ctipackage"
	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/storage"
	"github.com/acronis/go-cti/pkg/storage/gitstorage"
)

const (
	DependencyDirName = ".dep"
)

type DependencyManager interface {
	// Add new dependencies to index.lock
	Add(pkg *ctipackage.Package, depends map[string]string) error
	// Install dependencies from index.lock
	Install(pkg *ctipackage.Package) error
	// Download dependencies and their sub-dependencies
	Download(depends map[string]string) ([]CachedDependencyInfo, error)
}

type Option func(*dependencyManager)

type dependencyManager struct {
	PackagesDir string
	Storage     storage.Storage
}

func New(options ...Option) (DependencyManager, error) {
	depman := &dependencyManager{}

	for _, o := range options {
		o(depman)
	}

	if depman.Storage == nil {
		depman.Storage = gitstorage.New()
	}
	if depman.PackagesDir == "" {
		cacheDir, err := filesys.GetCtiPackagesCacheDir()
		if err != nil {
			return nil, fmt.Errorf("get cache dir: %w", err)
		}
		depman.PackagesDir = cacheDir
	}

	return depman, nil
}

func WithDownloader(st storage.Storage) Option {
	return func(dm *dependencyManager) {
		dm.Storage = st
	}
}

func WithPackagesCache(cacheDir string) Option {
	return func(dm *dependencyManager) {
		dm.PackagesDir = cacheDir
	}
}

func (dm *dependencyManager) Add(pkg *ctipackage.Package, depends map[string]string) error {
	// Validate dependencies
	if err := dm.installDependencies(pkg, depends); err != nil {
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

func (dm *dependencyManager) Install(pkg *ctipackage.Package) error {
	if err := dm.installDependencies(pkg, pkg.Index.Depends); err != nil {
		return fmt.Errorf("install index dependencies: %w", err)
	}
	if err := pkg.SaveIndexLock(); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}
	return nil
}

func (dm *dependencyManager) Download(depends map[string]string) ([]CachedDependencyInfo, error) {
	installed := []CachedDependencyInfo{}
	subDepends := map[string]string{}
	for source, version := range depends {
		info, err := dm.downloadDependency(source, version)
		if err != nil {
			return nil, fmt.Errorf("download dependency %s %s: %w", source, version, err)
		}

		installed = append(installed, info)
		// TODO check for cyclic dependencies or duplicates
		for subSource, subTag := range info.Index.Depends {
			subDepends[subSource] = subTag
		}
	}

	// Recursively download sub-dependencies
	if len(subDepends) != 0 {
		inst, err := dm.Download(subDepends)
		if err != nil {
			return nil, fmt.Errorf("download sub-dependencies: %w", err)
		}
		installed = append(installed, inst...)
	}

	return installed, nil
}

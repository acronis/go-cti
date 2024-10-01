package depman

import (
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/storage"
	"github.com/acronis/go-cti/pkg/storage/gitstorage"
)

const (
	DependencyDirName = ".dep"
)

type DependencyManager interface {
	// Add new dependencies to index.lock
	Add(bd *bundle.Bundle, depends map[string]string) error
	// Install dependencies from index.lock
	Install(bd *bundle.Bundle) error
	// Download dependencies and their sub-dependencies
	Download(depends map[string]string) ([]CachedDependencyInfo, error)
}

type Option func(*dependencyManager)

type dependencyManager struct {
	BundlesDir string
	Storage    storage.Storage
}

func New(options ...Option) (DependencyManager, error) {
	depman := &dependencyManager{}

	for _, o := range options {
		o(depman)
	}

	if depman.Storage == nil {
		depman.Storage = gitstorage.New()
	}
	if depman.BundlesDir == "" {
		cacheDir, err := filesys.GetCtiBundlesCacheDir()
		if err != nil {
			return nil, fmt.Errorf("get cache dir: %w", err)
		}
		depman.BundlesDir = cacheDir
	}

	return depman, nil
}

func WithDownloader(st storage.Storage) Option {
	return func(dm *dependencyManager) {
		dm.Storage = st
	}
}

func WithBundlesCache(cacheDir string) Option {
	return func(dm *dependencyManager) {
		dm.BundlesDir = cacheDir
	}
}

func (dm *dependencyManager) Add(bd *bundle.Bundle, depends map[string]string) error {
	// Validate dependencies
	if err := dm.installDependencies(bd, depends); err != nil {
		return fmt.Errorf("install dependencies: %w", err)
	}

	for source, version := range depends {
		if _, ok := bd.Index.Depends[source]; ok {
			slog.Info("Added direct dependency", slog.String("bundle", source), slog.String("version", version))
			bd.Index.Depends[source] = version
		}
		// TODO check if depends version were updated
		// is possible?
	}

	if err := bd.SaveIndex(); err != nil {
		return fmt.Errorf("save index: %w", err)
	}

	if err := bd.SaveIndexLock(); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}

	return nil
}

func (dm *dependencyManager) Install(bd *bundle.Bundle) error {
	if err := dm.installDependencies(bd, bd.Index.Depends); err != nil {
		return fmt.Errorf("install index dependencies: %w", err)
	}
	if err := bd.SaveIndexLock(); err != nil {
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

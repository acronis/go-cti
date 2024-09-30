package depman

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/filesys"
)

type CachedDependencyInfo struct {
	Path      string
	Source    string
	Version   string
	Integrity string
	Index     bundle.Index
}

func (dm *dependencyManager) installDependencies(bd *bundle.Bundle, depends map[string]string) error {
	installed, err := dm.Download(depends)
	if err != nil {
		return fmt.Errorf("download dependencies: %w", err)
	}

	if err := dm.installFromCache(bd, installed); err != nil {
		return fmt.Errorf("install from cache: %w", err)
	}
	return nil
}

func (dm *dependencyManager) installFromCache(target *bundle.Bundle, depends []CachedDependencyInfo) error {
	// put new dependencies from cache and replace links
	for _, info := range depends {
		// Validate integrity with installed bundle
		if source, ok := target.IndexLock.DependentBundles[info.Index.AppCode]; ok {
			// TODO check integrity
			if source != info.Source {
				slog.Error("Bundle from different source was already installed",
					slog.String("app_code", info.Index.AppCode),
					slog.String("known", source),
					slog.String("new", info.Source))
				return fmt.Errorf("bundle from different source was already installed")
			}

			// TODO if the same source was already installed, skip
		}

		// Replace the dependency in the root bundle
		depPath := filepath.Join(target.BaseDir, DependencyDirName, info.Index.AppCode)
		if err := filesys.ReplaceWithCopy(info.Path, depPath); err != nil {
			return fmt.Errorf("replace with copy: %w", err)
		}
	}

	// Install RAMLX spec

	// Pre-build dependencies and update target's index lock
	for _, info := range depends {
		depPath := filepath.Join(target.BaseDir, DependencyDirName, info.Index.AppCode)

		bd := bundle.New(depPath)
		if err := bd.Read(); err != nil {
			return fmt.Errorf("read bundle: %w", err)
		}

		if err := bd.Parse(); err != nil {
			return fmt.Errorf("parse bundle: %w", err)
		}

		if err := bd.DumpCache(); err != nil {
			return fmt.Errorf("build cache: %w", err)
		}

		checksum, err := filesys.ComputeDirectoryHash(depPath)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		target.IndexLock.DependentBundles[info.Index.AppCode] = info.Source
		target.IndexLock.SourceInfo[info.Source] = bundle.Info{
			AppCode:   info.Index.AppCode,
			Version:   info.Version,
			Integrity: checksum,
			Source:    info.Source,
			Depends:   info.Index.Depends,
		}
	}
	return nil
}

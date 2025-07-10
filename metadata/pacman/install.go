package pacman

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/acronis/go-cti/metadata/filesys"
	"github.com/blang/semver/v4"
)

type CachedDependencyInfo struct {
	Path      string
	Source    string
	Version   semver.Version
	Integrity string
	Index     ctipackage.Index
}

func (pm *packageManager) installDependencies(pkg *ctipackage.Package, depends map[string]string, transitive bool) error {
	// Make sure that package is valid i.e. ramlx spec is in place
	if err := pkg.Sync(); err != nil {
		return fmt.Errorf("sync package: %w", err)
	}

	installed, err := pm.Download(depends)
	if err != nil {
		return fmt.Errorf("download dependencies: %w", err)
	}

	if err := pm.installFromCache(pkg, installed); err != nil {
		return fmt.Errorf("install from cache: %w", err)
	}
	return nil
}

func (pm *packageManager) installFromCache(target *ctipackage.Package, depends []CachedDependencyInfo) error {
	// put new dependencies from cache and replace links
	for _, info := range depends {
		// Validate integrity with installed package
		if source, ok := target.IndexLock.Depends[info.Index.PackageID]; ok {
			// TODO check integrity
			if source != info.Source {
				slog.Error("Package from different source was already installed",
					slog.String("id", info.Index.PackageID),
					slog.String("known", source),
					slog.String("new", info.Source))
				return fmt.Errorf("package from different source was already installed")
			}

			// TODO if the same source was already installed, skip
		}

		// Replace the dependency in the root package
		depPath := filepath.Join(target.BaseDir, ctipackage.DependencyDirName, info.Index.PackageID)
		if err := filesys.ReplaceWithCopy(info.Path, depPath); err != nil {
			return fmt.Errorf("replace with copy: %w", err)
		}
	}

	// Install RAMLX spec

	// Pre-build dependencies and update target's index lock
	for _, info := range depends {
		depPath := filepath.Join(target.BaseDir, ctipackage.DependencyDirName, info.Index.PackageID)

		pkg, err := ctipackage.New(depPath)
		if err != nil {
			return fmt.Errorf("new package: %w", err)
		}

		if err := pkg.Read(); err != nil {
			return fmt.Errorf("read package: %w", err)
		}

		if err := pkg.Parse(); err != nil {
			return fmt.Errorf("parse package: %w", err)
		}

		checksum, err := filesys.ComputeDirectoryHash(depPath)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		target.IndexLock.Depends[info.Index.PackageID] = info.Source
		target.IndexLock.DependsInfo[info.Source] = ctipackage.Info{
			PackageID: info.Index.PackageID,
			Version:   info.Version.String(),
			Integrity: checksum,
			Source:    info.Source,
			Depends:   info.Index.Depends,
		}
	}
	return nil
}

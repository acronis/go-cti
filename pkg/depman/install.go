package depman

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/ctipackage"
	"github.com/acronis/go-cti/pkg/filesys"
)

type CachedDependencyInfo struct {
	Path      string
	Source    string
	Version   string
	Integrity string
	Index     ctipackage.Index
}

func (dm *dependencyManager) installDependencies(pkg *ctipackage.Package, depends map[string]string) error {
	installed, err := dm.Download(depends)
	if err != nil {
		return fmt.Errorf("download dependencies: %w", err)
	}

	if err := dm.installFromCache(pkg, installed); err != nil {
		return fmt.Errorf("install from cache: %w", err)
	}
	return nil
}

func (dm *dependencyManager) installFromCache(target *ctipackage.Package, depends []CachedDependencyInfo) error {
	// put new dependencies from cache and replace links
	for _, info := range depends {
		// Validate integrity with installed package
		if source, ok := target.IndexLock.DependentPackages[info.Index.PackageID]; ok {
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
		depPath := filepath.Join(target.BaseDir, DependencyDirName, info.Index.PackageID)
		if err := filesys.ReplaceWithCopy(info.Path, depPath); err != nil {
			return fmt.Errorf("replace with copy: %w", err)
		}
	}

	// Install RAMLX spec

	// Pre-build dependencies and update target's index lock
	for _, info := range depends {
		depPath := filepath.Join(target.BaseDir, DependencyDirName, info.Index.PackageID)

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

		if err := pkg.DumpCache(); err != nil {
			return fmt.Errorf("build cache: %w", err)
		}

		checksum, err := filesys.ComputeDirectoryHash(depPath)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		target.IndexLock.DependentPackages[info.Index.PackageID] = info.Source
		target.IndexLock.SourceInfo[info.Source] = ctipackage.Info{
			PackageID: info.Index.PackageID,
			Version:   info.Version,
			Integrity: checksum,
			Source:    info.Source,
			Depends:   info.Index.Depends,
		}
	}
	return nil
}

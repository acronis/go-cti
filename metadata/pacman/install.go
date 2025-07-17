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

func (pm *packageManager) installDependencies(targetDir string, depends map[string]string) ([]CachedDependencyInfo, error) {
	installed, err := pm.Download(depends, true)
	if err != nil {
		return nil, fmt.Errorf("download dependencies: %w", err)
	}

	return pm.installFromCache(targetDir, installed)
}

// TODO add tests
func validateDependencies(expected map[string]ctipackage.Info, installed []CachedDependencyInfo) bool {
	valid := true
	for _, info := range installed {
		if existingInfo, ok := expected[info.Source]; ok {
			if info.Index.PackageID != existingInfo.PackageID {
				slog.Error("Package ID mismatch",
					slog.String("source", info.Source),
					slog.String("expected", existingInfo.PackageID),
					slog.String("got", info.Index.PackageID),
				)
				valid = false
			}
			// if info.Integrity != existingInfo.Integrity {
			// 	slog.Error("Package hash mismatch",
			// 		slog.String("source", info.Source),
			// 		slog.String("package_id", info.Index.PackageID),
			// 		slog.String("expected", existingInfo.Integrity),
			// 		slog.String("got", info.Integrity),
			// 	)
			// 	valid = false
			// }
		} else {
			slog.Error("Package downloaded but not found in index-lock",
				slog.String("source", info.Source),
				slog.String("package_id", info.Index.PackageID),
				slog.String("version", info.Version.String()),
				slog.String("integrity", info.Integrity),
			)
			valid = false
		}
	}
	for source, info := range expected {
		found := func() bool {
			for _, inst := range installed {
				if inst.Source == source {
					return true
				}
			}
			return false
		}()
		if !found {
			slog.Error("Package from index-lock was not downloaded",
				slog.String("source", source),
				slog.String("package_id", info.PackageID),
				slog.String("version", info.Version),
				slog.String("integrity", info.Integrity),
			)
			valid = false
		}
	}
	return valid
}

func (pm *packageManager) installDependenciesInfo(targetDir string, infos map[string]ctipackage.Info) ([]CachedDependencyInfo, error) {
	depends := make(map[string]string, len(infos))
	for source, info := range infos {
		depends[source] = info.Version
	}

	// Download only direct dependencies
	installed, err := pm.Download(depends, false)
	if err != nil {
		return nil, fmt.Errorf("download dependencies: %w", err)
	}

	// check package integrity
	if !validateDependencies(infos, installed) {
		return nil, fmt.Errorf("depends integrity mismatch")
	}

	return pm.installFromCache(targetDir, installed)
}

func (pm *packageManager) installFromCache(targetDir string, depends []CachedDependencyInfo) ([]CachedDependencyInfo, error) {
	// put new dependencies from cache and replace links
	for _, info := range depends {
		// Replace the dependency in the root package
		depPath := filepath.Join(targetDir, ctipackage.DependencyDirName, info.Index.PackageID)
		if err := filesys.ReplaceWithCopy(info.Path, depPath); err != nil {
			return nil, fmt.Errorf("replace with copy: %w", err)
		}
	}

	// Pre-build dependencies
	result := make([]CachedDependencyInfo, 0, len(depends))
	for _, info := range depends {
		depPath := filepath.Join(targetDir, ctipackage.DependencyDirName, info.Index.PackageID)

		pkg, err := ctipackage.New(depPath)
		if err != nil {
			return nil, fmt.Errorf("new package: %w", err)
		}

		if err := pkg.Read(); err != nil {
			return nil, fmt.Errorf("read package: %w", err)
		}

		if err := pkg.Parse(); err != nil {
			return nil, fmt.Errorf("parse package: %w", err)
		}

		checksum, err := filesys.ComputeDirectoryHash(depPath)
		if err != nil {
			return nil, fmt.Errorf("compute directory hash: %w", err)
		}

		result = append(result, CachedDependencyInfo{
			Path:      depPath,
			Source:    info.Source,
			Version:   info.Version,
			Integrity: checksum,
			Index:     info.Index,
		})
	}
	return result, nil
}

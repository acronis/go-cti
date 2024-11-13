package pacman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/acronis/go-cti/metadata/filesys"
	"github.com/acronis/go-cti/metadata/storage"
)

/*
{
    "Version": "v1.0.0",
    "Time": "2023-06-20T06:39:01Z",
    "Origin": {
        "VCS": "git",
        "URL": "https://github.com/imdario/mergo",
        "Hash": "131de815afc35a77c41ae99da6c8f4288b6cb513",
        "Ref": "refs/tags/v1.0.0"
    }
}
*/

type SourceIntegrityInfo struct {
	Version string         `json:"Version"`
	Time    string         `json:"Time"`
	Origin  storage.Origin `json:"Origin"`
}

func (inf *SourceIntegrityInfo) Read(pm *packageManager, source string, version string) error {
	infoPath := pm.getSourceInfoPath(source, version)
	if _, err := os.Stat(infoPath); err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("stat %s: %w", infoPath, err)
	}

	if err := filesys.ReadJSON(infoPath, inf); err != nil {
		return fmt.Errorf("read origin info %s: %w", infoPath, err)
	}

	return nil
}

func (inf *SourceIntegrityInfo) Write(pm *packageManager, source string, version string) error {
	infoPath := pm.getSourceInfoPath(source, version)

	if err := os.MkdirAll(filepath.Dir(infoPath), os.ModePerm); err != nil {
		return fmt.Errorf("create package info directory: %w", err)
	}

	if err := filesys.WriteJSON(infoPath, inf); err != nil {
		return fmt.Errorf("write %s: %w", infoPath, err)
	}

	return nil
}

type PackageIntegrityInfo struct {
	Source  string `json:"Source"`
	Version string `json:"Version"`
	Hash    string `json:"Hash"`
}

func (inf *PackageIntegrityInfo) Read(pm *packageManager, pkgId string, version string) error {
	infoPath := pm.getPackageInfoPath(pkgId, version)
	if _, err := os.Stat(infoPath); err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("stat %s: %w", infoPath, err)
	}

	if err := filesys.ReadJSON(infoPath, inf); err != nil {
		return fmt.Errorf("read %s: %w", infoPath, err)
	}

	return nil
}

func (inf *PackageIntegrityInfo) Write(pm *packageManager, pkgId string, version string) error {
	infoPath := pm.getPackageInfoPath(pkgId, version)

	if err := os.MkdirAll(filepath.Dir(infoPath), os.ModePerm); err != nil {
		return fmt.Errorf("create package info directory: %w", err)
	}

	if err := filesys.WriteJSON(infoPath, inf); err != nil {
		return fmt.Errorf("write %s: %w", infoPath, err)
	}

	return nil
}

func (pm *packageManager) validateSourceInformation(source string, version string, info storage.Origin) error {
	sourceInfo := SourceIntegrityInfo{
		Origin: pm.Storage.Origin(), // required for proper parsing
	}
	if err := sourceInfo.Read(pm, source, version); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read source info: %w", err)
	}

	if err := sourceInfo.Origin.Validate(info); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	return nil
}

// Check source and package integrity cache and update both
func (pm *packageManager) updateDependencyCache(source string, version string, info storage.Origin, depDir string, depIdx *ctipackage.Index) error {
	sourceInfo := SourceIntegrityInfo{
		Origin: pm.Storage.Origin(), // required for proper parsing
	}

	if err := sourceInfo.Read(pm, source, version); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read source info: %w", err)
		}

		sourceInfo = SourceIntegrityInfo{
			Version: version,
			Time:    "TODO",
			Origin:  info,
		}

		if err := sourceInfo.Write(pm, source, version); err != nil {
			return fmt.Errorf("write integrity info: %w", err)
		}
	} else {
		// source information already exists
		// TODO validate the information
	}

	// move dependency from cache to the dependencies directory, calculate directory integrity information
	// TODO save additional storage specific information

	packageInfo := PackageIntegrityInfo{}
	if err := packageInfo.Read(pm, depIdx.PackageID, version); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read package info: %w", err)
		}

		hash, err := filesys.ComputeDirectoryHash(depDir)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		packageInfo = PackageIntegrityInfo{
			Source:  source,
			Version: version,
			Hash:    hash,
		}

		if err := packageInfo.Write(pm, depIdx.PackageID, version); err != nil {
			return fmt.Errorf("write package integrity info: %w", err)
		}
	} else {
		hash, err := filesys.ComputeDirectoryHash(depDir)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		if hash != packageInfo.Hash {
			return fmt.Errorf("package integrity check failed")
		}
	}

	return nil
}

package depman

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/downloader"
	"github.com/acronis/go-cti/pkg/filesys"
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
	Version string          `json:"Version"`
	Time    string          `json:"Time"`
	Origin  downloader.Info `json:"Origin"`
}

func (inf *SourceIntegrityInfo) Read(dm *dependencyManager, source string, version string) error {
	infoPath := dm.getSourceInfoPath(source, version)
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

func (inf *SourceIntegrityInfo) Write(dm *dependencyManager, source string, version string) error {
	infoPath := dm.getSourceInfoPath(source, version)

	if err := os.MkdirAll(filepath.Dir(infoPath), os.ModePerm); err != nil {
		return fmt.Errorf("create bundle info directory: %w", err)
	}

	if err := filesys.WriteJSON(infoPath, inf); err != nil {
		return fmt.Errorf("write %s: %w", infoPath, err)
	}

	return nil
}

type BundleIntegrityInfo struct {
	Source  string `json:"Source"`
	Version string `json:"Version"`
	Hash    string `json:"Hash"`
}

func (inf *BundleIntegrityInfo) Read(dm *dependencyManager, appCode string, version string) error {
	infoPath := dm.getBundleInfoPath(appCode, version)
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

func (inf *BundleIntegrityInfo) Write(dm *dependencyManager, appCode string, version string) error {
	infoPath := dm.getBundleInfoPath(appCode, version)

	if err := os.MkdirAll(filepath.Dir(infoPath), os.ModePerm); err != nil {
		return fmt.Errorf("create bundle info directory: %w", err)
	}

	if err := filesys.WriteJSON(infoPath, inf); err != nil {
		return fmt.Errorf("write %s: %w", infoPath, err)
	}

	return nil
}

func (dm *dependencyManager) validateSourceInformation(source string, version string, info downloader.Info) error {
	sourceInfo := SourceIntegrityInfo{}
	if err := sourceInfo.Read(dm, source, version); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read source info: %w", err)
	}

	valid := true
	if sourceInfo.Origin.VCS != info.VCS {
		slog.Error("vcs mismatch", slog.String("known", sourceInfo.Origin.VCS), slog.String("new", info.VCS))
		valid = false
	}
	if sourceInfo.Origin.URL != info.URL {
		slog.Error("url mismatch", slog.String("known", sourceInfo.Origin.URL), slog.String("new", info.URL))
		valid = false
	}
	if sourceInfo.Origin.Hash != info.Hash {
		slog.Error("hash mismatch", slog.String("known", sourceInfo.Origin.Hash), slog.String("new", info.Hash))
		valid = false
	}
	if sourceInfo.Origin.Ref != info.Ref {
		slog.Error("ref mismatch", slog.String("known", sourceInfo.Origin.Ref), slog.String("new", info.Ref))
		valid = false
	}
	if !valid {
		return fmt.Errorf("integrity check failed, please see logs for details")
	}

	return nil
}

// Check source and bundle integrity cache and update both
func (dm *dependencyManager) updateDependencyCache(source string, version string, info downloader.Info, depDir string, depIdx *bundle.Index) error {
	sourceInfo := SourceIntegrityInfo{}
	if err := sourceInfo.Read(dm, source, version); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read source info: %w", err)
		}

		sourceInfo = SourceIntegrityInfo{
			Version: version,
			Time:    "TODO",
			Origin:  info,
		}

		if err := sourceInfo.Write(dm, source, version); err != nil {
			return fmt.Errorf("write integrity info: %w", err)
		}
	} else {
		// source information already exists
		// TODO validate the information
	}

	// move dependency from cache to the dependencies directory, calculate directory integrity information
	// TODO save additional downloader specific information

	bundleInfo := BundleIntegrityInfo{}
	if err := bundleInfo.Read(dm, depIdx.AppCode, version); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read bundle info: %w", err)
		}

		hash, err := filesys.ComputeDirectoryHash(depDir)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		bundleInfo = BundleIntegrityInfo{
			Source:  source,
			Version: version,
			Hash:    hash,
		}

		if err := bundleInfo.Write(dm, depIdx.AppCode, version); err != nil {
			return fmt.Errorf("write bundle integrity info: %w", err)
		}
	} else {
		hash, err := filesys.ComputeDirectoryHash(depDir)
		if err != nil {
			return fmt.Errorf("compute directory hash: %w", err)
		}

		if hash != bundleInfo.Hash {
			return fmt.Errorf("bundle integrity check failed")
		}
	}

	return nil
}

package depman

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/collector"
	"github.com/acronis/go-cti/pkg/cti"
	"github.com/acronis/go-cti/pkg/downloader"
	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/parser"
	"github.com/acronis/go-cti/pkg/validator"
)

const (
	DependencyDirName = ".dep"
	BundleName        = "bundle.zip"
)

type DependencyManager interface {
	InstallNewDependencies(depends []string, replace bool) ([]string, error)
	InstallIndexDependencies() ([]string, error)
	// TODO strip to separate interfaces
	ParseWithCache() (parser.Parser, *collector.CtiRegistry, error)
	LoadEntitiesFromCache(cacheFile string) (cti.Entities, error)
	Validate() []error
	Pack(includeSource bool) (string, error)
}

type dependencyManager struct {
	RootBundle      *bundle.Bundle
	BundlesCacheDir string
	DependenciesDir string
	Downloader      downloader.Downloader

	BaseDir string
}

func New(idxFile string) (DependencyManager, error) {
	bundle, err := bundle.New(idxFile)
	if err != nil {
		return nil, fmt.Errorf("create bundle: %w", err)
	}
	cacheDir, err := filesys.GetCtiBundlesCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get cache dir: %w", err)
	}

	depman := &dependencyManager{
		RootBundle:      bundle,
		BundlesCacheDir: cacheDir,
		DependenciesDir: filepath.Join(bundle.BaseDir, DependencyDirName),
		BaseDir:         bundle.BaseDir,
	}

	depman.Downloader = downloader.New(depman.RootBundle.IndexLock, depman.BundlesCacheDir, depman.DependenciesDir)

	return depman, nil
}

func NewWithDownloader(idxFile string, dl downloader.Downloader) (DependencyManager, error) {
	bundle, err := bundle.New(idxFile)
	if err != nil {
		return nil, fmt.Errorf("create bundle: %w", err)
	}
	cacheDir, err := filesys.GetCtiBundlesCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get cache dir: %w", err)
	}

	depman := &dependencyManager{
		RootBundle:      bundle,
		BundlesCacheDir: cacheDir,
		DependenciesDir: filepath.Join(bundle.BaseDir, DependencyDirName),
		BaseDir:         bundle.BaseDir,
	}

	depman.Downloader = dl

	return depman, nil
}

func (depman *dependencyManager) InstallNewDependencies(depends []string, replace bool) ([]string, error) {
	installed, replaced, err := depman.installDependencies(depends, replace)
	if err != nil {
		return nil, fmt.Errorf("install dependencies: %w", err)
	}

	// TODO: Possibly needs refactor
	if len(replaced) != 0 {
		var depends []string
		for _, idxDepName := range depman.RootBundle.Index.Depends {
			depName, _ := bundle.ParseIndexDependency(idxDepName)
			if _, ok := replaced[depName]; ok {
				continue
			}
			depends = append(depends, idxDepName)
		}
		depman.RootBundle.Index.Depends = depends
	}

	for _, depName := range depends {
		found := false
		for _, idxDepName := range depman.RootBundle.Index.Depends {
			if idxDepName == depName {
				found = true
				break
			}
		}
		if !found {
			depman.RootBundle.Index.Depends = append(depman.RootBundle.Index.Depends, depName)
			slog.Info(fmt.Sprintf("Added %s as direct dependency", depName))
		}
	}

	if err = depman.RootBundle.SaveIndex(); err != nil {
		return nil, fmt.Errorf("save index: %w", err)
	}

	if err = depman.RootBundle.SaveIndexLock(); err != nil {
		return nil, fmt.Errorf("save index lock: %w", err)
	}

	return installed, nil
}

func (depman *dependencyManager) InstallIndexDependencies() ([]string, error) {
	installed, _, err := depman.installDependencies(depman.RootBundle.Index.Depends, false)
	if err != nil {
		return nil, fmt.Errorf("install index dependencies: %w", err)
	}
	if err = depman.RootBundle.SaveIndexLock(); err != nil {
		return nil, fmt.Errorf("save index lock: %w", err)
	}
	return installed, nil
}

func (depman *dependencyManager) installDependencies(depends []string, replace bool) ([]string, map[string]struct{}, error) {
	installed, replaced, err := depman.Downloader.Download(depends, replace)
	if err != nil {
		return nil, nil, fmt.Errorf("download dependencies: %w", err)
	}
	if err = depman.processInstalledDependencies(installed); err != nil {
		return nil, nil, fmt.Errorf("process installed dependencies: %w", err)
	}
	return installed, replaced, nil
}

func (depman *dependencyManager) ParseWithCache() (parser.Parser, *collector.CtiRegistry, error) {
	// TODO: Always build current bundle?
	p, err := parser.ParsePackage(depman.RootBundle.Index.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("parse bundle: %w", err)
	}
	if err := p.DumpCache(); err != nil {
		return nil, nil, fmt.Errorf("dump cache: %w", err)
	}
	// Make a shallow clone of the resulting registry to make an enriched registry
	r := p.GetRegistry().Clone()
	for _, dep := range depman.RootBundle.IndexLock.Bundles {
		cacheFile := filepath.Join(depman.DependenciesDir, dep.AppCode, parser.MetadataCacheFile)
		// TODO: Automatically rebuild cache if missing?
		entities, err := depman.LoadEntitiesFromCache(cacheFile)
		if err != nil {
			return nil, nil, fmt.Errorf("load cache file %s: %w", cacheFile, err)
		}
		for _, entity := range entities {
			if entity.Values != nil {
				r.Instances[entity.Cti] = entity
			} else if entity.Schema != nil {
				r.Types[entity.Cti] = entity
			} else {
				return nil, nil, fmt.Errorf("invalid entity: %s", entity.Cti)
			}
			// TODO: Check for duplicates?
			r.TotalIndex[entity.Cti] = entity
			r.Total = append(r.Total, entity)
		}
	}
	return p, r, nil
}

func (depman *dependencyManager) LoadEntitiesFromCache(cacheFile string) (cti.Entities, error) {
	f, err := os.OpenFile(cacheFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open cache file %s: %w", cacheFile, err)
	}
	defer f.Close()

	d := json.NewDecoder(f)
	var entities cti.Entities
	if err := d.Decode(&entities); err != nil {
		return nil, fmt.Errorf("decode cache file %s: %w", cacheFile, err)
	}
	return entities, nil
}

func (depman *dependencyManager) Validate() []error {
	_, r, err := depman.ParseWithCache()
	if err != nil {
		return []error{fmt.Errorf("parse with cache: %w", err)}
	}
	validator := validator.MakeCtiValidator()
	validator.LoadFromRegistry(r)
	// TODO: Validation for usage of indirect dependencies
	return validator.ValidateAll()
}

func (depman *dependencyManager) Pack(includeSource bool) (string, error) {
	p, r, err := depman.ParseWithCache()
	if err != nil {
		return "", fmt.Errorf("parse with cache: %w", err)
	}
	fileName := filepath.Join(depman.BaseDir, BundleName)
	archive, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	w, err := zipWriter.Create("index.json")
	if err != nil {
		return "", fmt.Errorf("create index in bundle: %w", err)
	}

	idx := depman.RootBundle.Index.Clone()
	idx.PutSerialized(parser.MetadataCacheFile)

	if _, err = w.Write(idx.ToBytes()); err != nil {
		return "", fmt.Errorf("write index to bundle: %w", err)
	}

	for _, metadata := range idx.Serialized {
		f, err := os.OpenFile(filepath.Join(depman.BaseDir, metadata), os.O_RDONLY, 0o644)
		if err != nil {
			return "", fmt.Errorf("open serialized metadata %s: %w", metadata, err)
		}
		defer f.Close()

		w, err := zipWriter.Create(metadata)
		if err != nil {
			return "", fmt.Errorf("create serialized metadata %s in bundle: %w", metadata, err)
		}
		if _, err = io.Copy(w, f); err != nil {
			return "", fmt.Errorf("write serialized metadata %s to bundle: %w", metadata, err)
		}
	}

	if includeSource {
		err := filepath.WalkDir(depman.BaseDir, func(path string, d os.DirEntry, err error) error {
			rel := strings.TrimPrefix(path, depman.BaseDir)
			if rel == "" || d.IsDir() {
				return nil
			}
			rel = filepath.ToSlash(rel[1:])
			if err != nil {
				return fmt.Errorf("walk directory: %w", err)
			}
			if rel[0] == '.' || rel == BundleName || rel == bundle.IndexFileName {
				return nil
			}
			f, err := os.OpenFile(path, os.O_RDONLY, 0o644)
			if err != nil {
				return fmt.Errorf("open index: %w", err)
			}
			w, err := zipWriter.Create(rel)
			if err != nil {
				return fmt.Errorf("create file in bundle: %w", err)
			}
			if _, err = io.Copy(w, f); err != nil {
				return fmt.Errorf("copy file in bundle: %w", err)
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("walk directory: %w", err)
		}
		return "", nil
	}

	for _, entity := range p.GetRegistry().Instances {
		tID := cti.GetParentCti(entity.Cti)
		typ, ok := r.Types[tID]
		if !ok {
			return "", fmt.Errorf("type %s not found", tID)
		}
		// TODO: Collect annotations from the entire chain of CTI types
		for key, annotation := range typ.Annotations {
			if annotation.Asset == nil {
				continue
			}
			value := key.GetValue(entity.Values)
			assetPath := value.String()
			if assetPath == "" {
				break
			}
			err := func() error {
				asset, err := os.OpenFile(filepath.Join(depman.BaseDir, assetPath), os.O_RDONLY, 0o644)
				if err != nil {
					return fmt.Errorf("open asset %s: %w", assetPath, err)
				}
				defer asset.Close()

				w, err := zipWriter.Create(assetPath)
				if err != nil {
					return fmt.Errorf("create asset %s in bundle: %w", assetPath, err)
				}
				if _, err = io.Copy(w, asset); err != nil {
					return fmt.Errorf("write asset %s to bundle: %w", assetPath, err)
				}
				return nil
			}()
			if err != nil {
				return "", fmt.Errorf("bundle asset %s: %w", assetPath, err)
			}
		}
	}

	return fileName, nil
}

func (depman *dependencyManager) processInstalledDependencies(installed []string) error {
	for _, sourceName := range installed {
		pkgLock := depman.RootBundle.IndexLock.Bundles[sourceName]
		pkgPath := filepath.Join(depman.DependenciesDir, pkgLock.AppCode)
		for _, dep := range pkgLock.Depends {
			depSourceName, _ := bundle.ParseIndexDependency(dep)
			depBundleLock := depman.RootBundle.IndexLock.Bundles[depSourceName]

			if err := depman.rewriteDepLinks(pkgPath, depBundleLock.AppCode); err != nil {
				return fmt.Errorf("rewrite dependency links: %w", err)
			}
		}
		if err := parser.BuildPackageCache(filepath.Join(pkgPath, bundle.IndexFileName)); err != nil {
			return fmt.Errorf("build cache: %w", err)
		}
	}
	return nil
}

func (depman *dependencyManager) rewriteDepLinks(pkgPath, depName string) error {
	relPath, err := filepath.Rel(pkgPath, depman.RootBundle.BaseDir)
	if err != nil {
		return err
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	orig := fmt.Sprintf("%s/%s", DependencyDirName, depName)
	repl := fmt.Sprintf("%s/%s/%s", relPath, DependencyDirName, depName)

	for _, file := range filesys.WalkDir(pkgPath, ".raml") {
		// TODO: Maybe read file line by line?
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		contents := strings.ReplaceAll(string(raw), orig, repl)
		err = os.WriteFile(file, []byte(contents), 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

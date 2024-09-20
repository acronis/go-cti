package pacman

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/pkg/collector"
	"github.com/acronis/go-cti/pkg/cti"
	"github.com/acronis/go-cti/pkg/filesys"
	_package "github.com/acronis/go-cti/pkg/package"
	"github.com/acronis/go-cti/pkg/parser"
	"github.com/acronis/go-cti/pkg/validator"
)

const (
	DependencyDirName = ".dep"
	BundleName        = "bundle.zip"
)

type PackageManager struct {
	Package         *_package.Package
	PackageCacheDir string
	DependenciesDir string
	Downloader      Downloader

	BaseDir string
}

func New(idxFile string) (*PackageManager, error) {
	pkg, err := _package.New(idxFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}
	pkgCacheDir, err := filesys.GetPkgCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get package cache dir: %w", err)
	}

	pacman := &PackageManager{
		Package:         pkg,
		PackageCacheDir: pkgCacheDir,
		DependenciesDir: filepath.Join(pkg.BaseDir, DependencyDirName),
		BaseDir:         pkg.BaseDir,
	}

	pacman.Downloader = NewGoLikeDownloader(pacman)

	return pacman, nil
}

func NewWithDownloader(idxFile string, dl Downloader) (*PackageManager, error) {
	pkg, err := _package.New(idxFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}
	pkgCacheDir, err := filesys.GetPkgCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get package cache dir: %w", err)
	}

	pacman := &PackageManager{
		Package:         pkg,
		PackageCacheDir: pkgCacheDir,
		DependenciesDir: filepath.Join(pkg.BaseDir, DependencyDirName),
		BaseDir:         pkg.BaseDir,
	}

	pacman.Downloader = dl

	return pacman, nil
}

func (pacman *PackageManager) InstallNewDependencies(depends []string, replace bool) ([]string, error) {
	installed, replaced, err := pacman.installDependencies(depends, replace)
	if err != nil {
		return nil, fmt.Errorf("failed to install dependencies: %w", err)
	}

	// TODO: Possibly needs refactor
	if len(replaced) != 0 {
		var depends []string
		for _, idxDepName := range pacman.Package.Index.Depends {
			depName, _ := ParseIndexDependency(idxDepName)
			if _, ok := replaced[depName]; ok {
				continue
			}
			depends = append(depends, idxDepName)
		}
		pacman.Package.Index.Depends = depends
	}

	for _, depName := range depends {
		found := false
		for _, idxDepName := range pacman.Package.Index.Depends {
			if idxDepName == depName {
				found = true
				break
			}
		}
		if !found {
			pacman.Package.Index.Depends = append(pacman.Package.Index.Depends, depName)
			slog.Info(fmt.Sprintf("Added %s as direct dependency", depName))
		}
	}

	if err = pacman.Package.SaveIndex(); err != nil {
		return nil, fmt.Errorf("failed to save index: %w", err)
	}

	if err = pacman.Package.SaveIndexLock(); err != nil {
		return nil, fmt.Errorf("failed to save index lock: %w", err)
	}

	return installed, nil
}

func (pacman *PackageManager) InstallIndexDependencies() ([]string, error) {
	installed, _, err := pacman.installDependencies(pacman.Package.Index.Depends, false)
	if err != nil {
		return nil, fmt.Errorf("failed to install index dependencies: %w", err)
	}
	if err = pacman.Package.SaveIndexLock(); err != nil {
		return nil, fmt.Errorf("failed to save index lock: %w", err)
	}
	return installed, nil
}

func (pacman *PackageManager) installDependencies(depends []string, replace bool) ([]string, map[string]struct{}, error) {
	installed, replaced, err := pacman.Downloader.Download(depends, replace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download dependencies: %w", err)
	}
	if err = pacman.processInstalledDependencies(installed); err != nil {
		return nil, nil, fmt.Errorf("failed to process installed dependencies: %w", err)
	}
	return installed, replaced, nil
}

func (pacman *PackageManager) ParseWithCache() (*parser.Parser, *collector.CtiRegistry, error) {
	// TODO: Always build current package?
	p, err := parser.ParsePackage(pacman.Package.Index.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse package: %w", err)
	}
	if err := p.DumpCache(); err != nil {
		return nil, nil, fmt.Errorf("failed to dump cache: %w", err)
	}
	// Make a shallow clone of the resulting registry to make an enriched registry
	r := p.Registry.Clone()
	for _, dep := range pacman.Package.IndexLock.Packages {
		cacheFile := filepath.Join(pacman.DependenciesDir, dep.AppCode, parser.MetadataCacheFile)
		// TODO: Automatically rebuild cache if missing?
		entities, err := pacman.LoadEntitiesFromCache(cacheFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load cache file %s: %w", cacheFile, err)
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

func (pacman *PackageManager) LoadEntitiesFromCache(cacheFile string) (cti.Entities, error) {
	f, err := os.OpenFile(cacheFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache file %s: %w", cacheFile, err)
	}
	defer f.Close()

	d := json.NewDecoder(f)
	var entities cti.Entities
	if err := d.Decode(&entities); err != nil {
		return nil, fmt.Errorf("failed to decode cache file %s: %w", cacheFile, err)
	}
	return entities, nil
}

func (pacman *PackageManager) Validate() []error {
	_, r, err := pacman.ParseWithCache()
	if err != nil {
		return []error{fmt.Errorf("failed to parse with cache: %w", err)}
	}
	validator := validator.MakeCtiValidator()
	validator.LoadFromRegistry(r)
	// TODO: Validation for usage of indirect dependencies
	return validator.ValidateAll()
}

func (pacman *PackageManager) Pack(includeSource bool) error {
	p, r, err := pacman.ParseWithCache()
	if err != nil {
		return fmt.Errorf("failed to parse with cache: %w", err)
	}
	archive, err := os.Create(filepath.Join(pacman.BaseDir, BundleName))
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	w, err := zipWriter.Create("index.json")
	if err != nil {
		return fmt.Errorf("failed to create index in bundle: %w", err)
	}

	idx := pacman.Package.Index.Clone()
	idx.PutSerialized(parser.MetadataCacheFile)

	if _, err = w.Write(idx.ToBytes()); err != nil {
		return fmt.Errorf("failed to write index to bundle: %w", err)
	}

	for _, metadata := range idx.Serialized {
		f, err := os.OpenFile(filepath.Join(p.BaseDir, metadata), os.O_RDONLY, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open serialized metadata %s: %w", metadata, err)
		}
		defer f.Close()

		w, err := zipWriter.Create(metadata)
		if err != nil {
			return fmt.Errorf("failed to create serialized metadata %s in bundle: %w", metadata, err)
		}
		if _, err = io.Copy(w, f); err != nil {
			return fmt.Errorf("failed to write serialized metadata %s to bundle: %w", metadata, err)
		}
	}

	if includeSource {
		err := filepath.WalkDir(p.BaseDir, func(path string, d os.DirEntry, err error) error {
			rel := strings.TrimPrefix(path, p.BaseDir)
			if rel == "" || d.IsDir() {
				return nil
			}
			rel = filepath.ToSlash(rel[1:])
			if err != nil {
				return fmt.Errorf("failed to walk directory: %w", err)
			}
			if rel[0] == '.' || rel == BundleName || rel == _package.IndexFileName {
				return nil
			}
			f, err := os.OpenFile(path, os.O_RDONLY, 0o644)
			if err != nil {
				return fmt.Errorf("failed to open index: %w", err)
			}
			w, err := zipWriter.Create(rel)
			if err != nil {
				return fmt.Errorf("failed to create file in bundle: %w", err)
			}
			if _, err = io.Copy(w, f); err != nil {
				return fmt.Errorf("failed to copy file in bundle: %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
		return nil
	}

	for _, entity := range p.Registry.Instances {
		tId := cti.GetParentCti(entity.Cti)
		typ, ok := r.Types[tId]
		if !ok {
			return fmt.Errorf("type %s not found", tId)
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
				asset, err := os.OpenFile(filepath.Join(p.BaseDir, assetPath), os.O_RDONLY, 0o644)
				if err != nil {
					return fmt.Errorf("failed to open asset %s: %w", assetPath, err)
				}
				defer asset.Close()

				w, err := zipWriter.Create(assetPath)
				if err != nil {
					return fmt.Errorf("failed to create asset %s in bundle: %w", assetPath, err)
				}
				if _, err = io.Copy(w, asset); err != nil {
					return fmt.Errorf("failed to write asset %s to bundle: %w", assetPath, err)
				}
				return nil
			}()
			if err != nil {
				return fmt.Errorf("failed to bundle asset %s: %w", assetPath, err)
			}
		}
	}

	return nil
}

func (pacman *PackageManager) processInstalledDependencies(installed []string) error {
	for _, sourceName := range installed {
		pkgLock := pacman.Package.IndexLock.Packages[sourceName]
		pkgPath := filepath.Join(pacman.DependenciesDir, pkgLock.AppCode)
		for _, dep := range pkgLock.Depends {
			depSourceName, _ := ParseIndexDependency(dep)
			depPkgLock := pacman.Package.IndexLock.Packages[depSourceName]
			err := pacman.rewriteDepLinks(pkgPath, depPkgLock.AppCode)
			if err != nil {
				return fmt.Errorf("failed to rewrite dependency links: %w", err)
			}
		}
		if err := parser.BuildPackageCache(filepath.Join(pkgPath, _package.IndexFileName)); err != nil {
			return fmt.Errorf("failed to build cache: %w", err)
		}
	}
	return nil
}

func (pacman *PackageManager) rewriteDepLinks(pkgPath, depName string) error {
	relPath, err := filepath.Rel(pkgPath, pacman.Package.BaseDir)
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
		err = os.WriteFile(file, []byte(contents), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

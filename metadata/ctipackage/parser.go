package ctipackage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-raml"
)

const (
	MetadataCacheFile = ".cache.json"
)

func (pkg *Package) Parse() error {
	// TODO: Probably this can be combined into a single step during dependency resolution.
	deps, err := pkg.resolveDependencyOrder()
	if err != nil {
		return fmt.Errorf("resolve dependency order: %w", err)
	}
	c := collector.New()
	// This ensures that duplicate dependencies are parsed only once.
	processed := map[string]struct{}{}
	for _, dep := range deps {
		if _, ok := processed[dep]; ok {
			continue
		}
		processed[dep] = struct{}{}
		depIndexFile := filepath.Join(pkg.BaseDir, DependencyDirName, dep)
		// FIXME: Need a proper detection of the package type.
		if strings.Contains(pkg.BaseDir, "/.dep/") {
			depIndexFile = filepath.Join(pkg.BaseDir, "..", dep)
		}
		depPkg, err := New(depIndexFile)
		if err != nil {
			return fmt.Errorf("new package: %w", err)
		}
		if err = depPkg.Read(); err != nil {
			return fmt.Errorf("read package: %w", err)
		}
		err = depPkg.parse(c, false)
		if err != nil {
			return fmt.Errorf("parse dependent package: %w", err)
		}
	}

	err = pkg.parse(c, true)
	if err != nil {
		return fmt.Errorf("parse main package: %w", err)
	}
	pkg.LocalRegistry = c.LocalRegistry
	pkg.GlobalRegistry = c.GlobalRegistry

	// TODO: Maybe need an option to parse without dumping cache?
	if err := pkg.DumpCache(); err != nil {
		return fmt.Errorf("dump cache: %w", err)
	}

	return nil
}

func (pkg *Package) resolveDependencyOrder() ([]string, error) {
	var deps []string
	for _, dep := range pkg.IndexLock.SourceInfo {
		depIndexFile := filepath.Join(pkg.BaseDir, DependencyDirName, dep.PackageID)
		// FIXME: Need a proper detection of the package type.
		if strings.Contains(pkg.BaseDir, "/.dep/") {
			depIndexFile = filepath.Join(pkg.BaseDir, "..", dep.PackageID)
		}
		depPkg, err := New(depIndexFile)
		if err != nil {
			return nil, fmt.Errorf("new package: %w", err)
		}
		if err = depPkg.Read(); err != nil {
			return nil, fmt.Errorf("read package: %w", err)
		}
		depNames, err := depPkg.resolveDependencyOrder()
		if err != nil {
			return nil, fmt.Errorf("resolve dependency order: %w", err)
		}
		deps = append(deps, depNames...)
		deps = append(deps, dep.PackageID)
	}
	return deps, nil
}

func (pkg *Package) parse(c *collector.Collector, isLocal bool) error {
	// NOTE: Sync is mandatory before parse. Otherwise, parse may fail due to missing ramlx folder.
	if err := pkg.Sync(); err != nil {
		return fmt.Errorf("sync package: %w", err)
	}

	r, err := raml.ParseFromString(pkg.Index.GenerateIndexRaml(false), "index.raml", pkg.BaseDir, raml.OptWithValidate())
	if err != nil {
		return fmt.Errorf("parse index.raml: %w", err)
	}

	c.SetRaml(r)
	if err := c.Collect(isLocal); err != nil {
		return fmt.Errorf("collect from package: %w", err)
	}
	pkg.Parsed = true

	return nil
}

func (pkg *Package) DumpCache() error {
	var items []metadata.Entity
	for _, v := range pkg.LocalRegistry.Index {
		items = append(items, v)
	}
	// Sort entities by CTI to make the cache deterministic
	sort.Slice(items, func(a, b int) bool {
		return items[a].GetCti() < items[b].GetCti()
	})

	bytes, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("serialize entities: %w", err)
	}
	return os.WriteFile(filepath.Join(pkg.BaseDir, MetadataCacheFile), bytes, 0600)
}

// FIXME: Fix caching.
// Currently it may not work in cases when extraneous cti.schema is used by the package
// func (pkg *Package) ParseWithCache() (*collector.MetadataRegistry, error) {
// 	if err := pkg.Parse(); err != nil {
// 		return nil, fmt.Errorf("parse package: %w", err)
// 	}

// 	// Make a shallow clone of the resulting registry to make an enriched registry
// 	r := pkg.LocalRegistry.Clone()

// 	for _, dep := range pkg.IndexLock.SourceInfo {
// 		cacheFile := filepath.Join(pkg.BaseDir, DependencyDirName, dep.PackageID, MetadataCacheFile)
// 		// TODO: Automatically rebuild cache if missing?
// 		entities, err := loadEntitiesFromCache(cacheFile)
// 		if err != nil {
// 			return nil, fmt.Errorf("load cache file %s: %w", cacheFile, err)
// 		}
// 		for _, entity := range entities {
// 			err = r.Add(pkg.BaseDir, entity)
// 			if err != nil {
// 				return nil, fmt.Errorf("add entity %s: %w", entity.Cti, err)
// 			}
// 		}
// 	}
// 	return r, nil
// }

// func loadEntitiesFromCache(cacheFile string) (metadata.Entities, error) {
// 	f, err := os.OpenFile(cacheFile, os.O_RDONLY, 0644)
// 	if err != nil {
// 		return nil, fmt.Errorf("open cache file %s: %w", cacheFile, err)
// 	}
// 	defer f.Close()

// 	d := json.NewDecoder(f)
// 	var entities metadata.Entities
// 	if err := d.Decode(&entities); err != nil {
// 		return nil, fmt.Errorf("decode cache file %s: %w", cacheFile, err)
// 	}
// 	return entities, nil
// }

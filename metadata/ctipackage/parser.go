package ctipackage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-raml"
)

const (
	MetadataCacheFile = ".cache.json"
)

func (pkg *Package) Parse() error {
	// NOTE: Sync is mandatory before parse. Otherwise, parse may fail due to missing ramlx folder.
	if err := pkg.Sync(); err != nil {
		return fmt.Errorf("sync package: %w", err)
	}

	baseDir := pkg.BaseDir

	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}

	r, err := raml.ParseFromString(pkg.Index.GenerateIndexRaml(false), "index.raml", absPath, raml.OptWithValidate())
	if err != nil {
		return fmt.Errorf("parse index.raml: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return fmt.Errorf("collect from package: %w", err)
	}

	pkg.Registry = c.Registry

	// TODO: Maybe need an option to parse without dumping cache?
	if err := pkg.DumpCache(); err != nil {
		return fmt.Errorf("dump cache: %w", err)
	}

	return nil
}

func (pkg *Package) DumpCache() error {
	bytes, err := json.Marshal(pkg.Registry.Total)
	if err != nil {
		return fmt.Errorf("serialize entities: %w", err)
	}
	return os.WriteFile(filepath.Join(pkg.BaseDir, MetadataCacheFile), bytes, 0600)
}

func (pkg *Package) ParseWithCache() (*collector.MetadataRegistry, error) {
	if err := pkg.Parse(); err != nil {
		return nil, fmt.Errorf("parse package: %w", err)
	}

	// Make a shallow clone of the resulting registry to make an enriched registry
	r := pkg.Registry.Clone()

	for _, dep := range pkg.IndexLock.SourceInfo {
		cacheFile := filepath.Join(pkg.BaseDir, DependencyDirName, dep.PackageID, MetadataCacheFile)
		// TODO: Automatically rebuild cache if missing?
		entities, err := loadEntitiesFromCache(cacheFile)
		if err != nil {
			return nil, fmt.Errorf("load cache file %s: %w", cacheFile, err)
		}
		for _, entity := range entities {
			switch {
			case entity.Values != nil:
				r.Instances[entity.Cti] = entity
			case entity.Schema != nil:
				r.Types[entity.Cti] = entity
			default:
				return nil, fmt.Errorf("invalid entity: %s", entity.Cti)
			}

			// TODO: Check for duplicates?
			r.TotalIndex[entity.Cti] = entity
			r.Total = append(r.Total, entity)
		}
	}
	return r, nil
}

func loadEntitiesFromCache(cacheFile string) (metadata.Entities, error) {
	f, err := os.OpenFile(cacheFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open cache file %s: %w", cacheFile, err)
	}
	defer f.Close()

	d := json.NewDecoder(f)
	var entities metadata.Entities
	if err := d.Decode(&entities); err != nil {
		return nil, fmt.Errorf("decode cache file %s: %w", cacheFile, err)
	}
	return entities, nil
}

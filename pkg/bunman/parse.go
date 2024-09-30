package bunman

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/collector"
	"github.com/acronis/go-cti/pkg/cti"
	"github.com/acronis/go-cti/pkg/depman"
)

func ParseWithCache(bd *bundle.Bundle) (*collector.CtiRegistry, error) {
	if err := bd.Parse(); err != nil {
		return nil, fmt.Errorf("parse bundle: %w", err)
	}

	if err := bd.DumpCache(); err != nil {
		return nil, fmt.Errorf("dump cache: %w", err)
	}

	// Make a shallow clone of the resulting registry to make an enriched registry
	r := bd.Registry.Clone()

	for _, dep := range bd.IndexLock.SourceInfo {
		cacheFile := filepath.Join(bd.BaseDir, depman.DependencyDirName, dep.AppCode, bundle.MetadataCacheFile)
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

func loadEntitiesFromCache(cacheFile string) (cti.Entities, error) {
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

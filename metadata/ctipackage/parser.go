package ctipackage

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/acronis/go-cti/metadata"
	cmetadata "github.com/acronis/go-cti/metadata/collector/ctimetadata"
	cramlx "github.com/acronis/go-cti/metadata/collector/ramlx"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/acronis/go-cti/metadata/registry"
	"github.com/acronis/go-cti/metadata/transformer"
	"github.com/acronis/go-raml/v2"
)

const (
	MetadataCacheFile = ".cache.json"
)

func (pkg *Package) Parse() error {
	if err := pkg.Sync(); err != nil {
		return fmt.Errorf("sync package: %w", err)
	}

	// TODO: Probably this can be combined into a single step during dependency resolution.
	deps, err := pkg.resolveDependencyOrder()
	if err != nil {
		return fmt.Errorf("resolve dependency order: %w", err)
	}

	// Initialize the global registry for the parsed package.
	pkg.GlobalRegistry = registry.New()

	// This ensures that duplicate dependencies are parsed only once.
	processed := map[string]struct{}{}
	for _, dep := range deps {
		if _, ok := processed[dep]; ok {
			continue
		}
		processed[dep] = struct{}{}
		var depIndexFile string
		// FIXME: Need a proper detection of the package type.
		if strings.Contains(pkg.BaseDir, "/.dep/") {
			depIndexFile = filepath.Join(pkg.BaseDir, "..", dep)
		} else {
			depIndexFile = filepath.Join(pkg.BaseDir, DependencyDirName, dep)
		}
		depPkg, err := New(depIndexFile)
		if err != nil {
			return fmt.Errorf("new package: %w", err)
		}
		if err = depPkg.Read(); err != nil {
			return fmt.Errorf("read package: %w", err)
		}
		// Dependent packages are safe to parse with cache since they are not modified
		// by the user. Cache is updated when the package is installed.
		// FIXME: Temporarily disabled since requires better cache management.
		// if err = depPkg.parseWithCache(); err != nil {
		// 	return fmt.Errorf("parse dependent package: %w", err)
		// }
		if err = depPkg.parse(); err != nil {
			return fmt.Errorf("parse dependent package: %w", err)
		}
		if err = pkg.GlobalRegistry.CopyFrom(depPkg.LocalRegistry); err != nil {
			return fmt.Errorf("copy entities from dependent package: %w", err)
		}
	}

	// Always parse the main package without cache, since it may contain user modifications.
	if err = pkg.parse(); err != nil {
		return fmt.Errorf("parse main package: %w", err)
	}
	if err = pkg.GlobalRegistry.CopyFrom(pkg.LocalRegistry); err != nil {
		return fmt.Errorf("copy entities from root package: %w", err)
	}

	t := transformer.New(pkg.GlobalRegistry)
	if err = t.Transform(); err != nil {
		return fmt.Errorf("transform entities: %w", err)
	}

	// TODO: Maybe need an option to parse without dumping cache?
	if err = pkg.DumpCache(); err != nil {
		return fmt.Errorf("dump cache: %w", err)
	}

	return nil
}

// resolveDependencyOrder recursively resolves the order of dependencies for the package
// and returns a slice of package IDs in the order they should be processed.
func (pkg *Package) resolveDependencyOrder() ([]string, error) {
	var deps []string
	for _, dep := range pkg.IndexLock.DependsInfo {
		// FIXME: Need a proper detection of the package type.
		var depIndexFile string
		if strings.Contains(pkg.BaseDir, "/.dep/") {
			depIndexFile = filepath.Join(pkg.BaseDir, "..", dep.PackageID)
		} else {
			depIndexFile = filepath.Join(pkg.BaseDir, DependencyDirName, dep.PackageID)
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

func (pkg *Package) parseRAML() (*registry.MetadataRegistry, error) {
	r, err := raml.ParseFromString(pkg.generateRAML(false), "index.raml", pkg.BaseDir, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("parse index.raml: %w", err)
	}
	c, err := cramlx.NewRAMLXCollector(r)
	if err != nil {
		return nil, fmt.Errorf("create ramlx collector: %w", err)
	}
	return c.Collect()
}

func (pkg *Package) parseCTIMetadata() (*registry.MetadataRegistry, error) {
	fragments := make(map[string][]byte)
	for _, entity := range pkg.Index.Entities {
		if !strings.HasSuffix(entity, YAMLExt) {
			continue
		}
		b, err := os.ReadFile(path.Join(pkg.BaseDir, entity))
		if err != nil {
			return nil, fmt.Errorf("read entity %s: %w", entity, err)
		}
		fragments[entity] = b
	}
	return cmetadata.NewCTIMetadataCollector(fragments, pkg.BaseDir).Collect()
}

func (pkg *Package) parse() error {
	ramlRegistry, err := pkg.parseRAML()
	if err != nil {
		return fmt.Errorf("parse RAML: %w", err)
	}
	ctiMetadataRegistry, err := pkg.parseCTIMetadata()
	if err != nil {
		return fmt.Errorf("collect from package: %w", err)
	}
	if err = ramlRegistry.CopyFrom(ctiMetadataRegistry); err != nil {
		return fmt.Errorf("copy entities from metadata registry: %w", err)
	}
	pkg.LocalRegistry = ramlRegistry
	pkg.Parsed = true
	return nil
}

func (pkg *Package) generateRAML(includeExamples bool) string {
	// TODO: Maybe it is possible to avoid index.raml generation and reuse RAML parser instance to parse each entity file instead.
	// Could have something like PackageParser.Initialize(path string) (maybe even in go-raml itself).
	// This would also allow employing per-fragment cache strategy based on project configuration.
	var sb strings.Builder
	sb.WriteString("#%RAML 1.0 Library\nuses:")
	for i, entity := range pkg.Index.Entities {
		if strings.HasSuffix(entity, RAMLExt) {
			sb.WriteString(fmt.Sprintf("\n  e%d: %s", i+1, entity))
		}
	}
	if includeExamples {
		for i, example := range pkg.Index.Examples {
			if strings.HasSuffix(example, RAMLExt) {
				sb.WriteString(fmt.Sprintf("\n  x%d: %s", i+1, example))
			}
		}
	}
	return sb.String()
}

func (pkg *Package) DumpCache() error {
	var items []metadata.Entity
	for _, v := range pkg.LocalRegistry.Index {
		items = append(items, v)
	}
	// Sort entities by CTI to make the cache deterministic
	sort.Slice(items, func(a, b int) bool {
		return items[a].GetCTI() < items[b].GetCTI()
	})

	bytes, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("serialize entities: %w", err)
	}
	return os.WriteFile(filepath.Join(pkg.BaseDir, MetadataCacheFile), bytes, 0600)
}

func (pkg *Package) parseWithCache() error {
	cacheFile := filepath.Join(pkg.BaseDir, MetadataCacheFile)
	if _, err := os.Stat(cacheFile); err == nil {
		// Cache file exists, load entities from cache.
		entities, err := pkg.loadEntitiesFromCache(cacheFile)
		if err != nil {
			return fmt.Errorf("load entities from cache: %w", err)
		}
		pkg.LocalRegistry = registry.New()
		for _, entity := range entities {
			if err = pkg.LocalRegistry.Add(entity); err != nil {
				return fmt.Errorf("add entity from cache: %w", err)
			}
		}
		pkg.Parsed = true
		return nil
	}
	if err := pkg.parse(); err != nil {
		return fmt.Errorf("parse package: %w", err)
	}
	return nil
}

func (pkg *Package) loadEntitiesFromCache(cacheFile string) (metadata.Entities, error) {
	f, err := os.OpenFile(cacheFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open cache file %s: %w", cacheFile, err)
	}
	defer f.Close()

	d := json.NewDecoder(f)
	var legacyEntities metadata.UntypedEntities
	if err = d.Decode(&legacyEntities); err != nil {
		return nil, fmt.Errorf("decode cache file %s: %w", cacheFile, err)
	}

	entities := make(metadata.Entities, len(legacyEntities))
	for i, legacyEntity := range legacyEntities {
		entity, err := pkg.untypedEntityToTypedEntity(legacyEntity)
		if err != nil {
			return nil, fmt.Errorf("convert legacy entity to typed entity: %w", err)
		}
		entities[i] = entity
	}
	return entities, nil
}

func (pkg *Package) untypedEntityToTypedEntity(legacyEntity metadata.UntypedEntity) (metadata.Entity, error) {
	switch {
	case legacyEntity.Schema != nil:
		var schema jsonschema.JSONSchemaCTI
		if err := json.Unmarshal(legacyEntity.Schema, &schema); err != nil {
			return nil, fmt.Errorf("unmarshal schema for %s: %w", legacyEntity.CTI, err)
		}
		e, err := metadata.NewEntityType(legacyEntity.CTI, &schema, legacyEntity.Annotations)
		if err != nil {
			return nil, fmt.Errorf("make entity type: %w", err)
		}
		e.SetFinal(legacyEntity.Final)
		e.SetResilient(legacyEntity.Resilient)
		e.SetAccess(legacyEntity.Access)
		e.SetDisplayName(legacyEntity.DisplayName)
		e.SetDescription(legacyEntity.Description)
		if legacyEntity.TraitsSchema != nil {
			var traitsSchema jsonschema.JSONSchemaCTI
			if err = json.Unmarshal(legacyEntity.TraitsSchema, &traitsSchema); err != nil {
				return nil, fmt.Errorf("unmarshal traits schema for %s: %w", legacyEntity.CTI, err)
			}
			e.SetTraitsSchema(&traitsSchema, legacyEntity.TraitsAnnotations)
		}
		if legacyEntity.Traits != nil {
			var traits map[string]any
			if err = json.Unmarshal(legacyEntity.Traits, &traits); err != nil {
				return nil, fmt.Errorf("unmarshal traits for %s: %w", legacyEntity.CTI, err)
			}
			e.SetTraits(traits)
		}
		e.SetSourceMap(metadata.EntityTypeSourceMap{
			Name: legacyEntity.SourceMap.Name,
			EntitySourceMap: metadata.EntitySourceMap{
				OriginalPath: legacyEntity.SourceMap.OriginalPath,
				SourcePath:   legacyEntity.SourceMap.SourcePath,
			},
		})
		return e, nil
	case legacyEntity.Values != nil:
		e, err := metadata.NewEntityInstance(legacyEntity.CTI, legacyEntity.Values)
		if err != nil {
			return nil, fmt.Errorf("make entity instance: %w", err)
		}
		e.SetFinal(true)
		e.SetResilient(legacyEntity.Resilient)
		e.SetAccess(legacyEntity.Access)
		e.SetDisplayName(legacyEntity.DisplayName)
		e.SetDescription(legacyEntity.Description)
		e.SetSourceMap(metadata.EntityInstanceSourceMap{
			AnnotationType: *legacyEntity.SourceMap.AnnotationType,
			EntitySourceMap: metadata.EntitySourceMap{
				OriginalPath: legacyEntity.SourceMap.OriginalPath,
				SourcePath:   legacyEntity.SourceMap.SourcePath,
			},
		})
		return e, nil
	default:
		return nil, fmt.Errorf("legacy entity %s has neither schema nor values", legacyEntity.CTI)
	}
}

package collector

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/registry"
	"gopkg.in/yaml.v3"
)

type CTIMetadataCollector struct {
	collector.BaseCollector

	// entryPoint is a map of YAML fragments, where each key is a fragment name
	// and value is JSON Schema which is expressed as raw bytes.
	Entry map[string][]byte
}

func NewCTIMetadataCollector(entry map[string][]byte, baseDir string) *CTIMetadataCollector {
	return &CTIMetadataCollector{
		BaseCollector: collector.BaseCollector{
			CTIParser: cti.NewParser(),
			Registry:  registry.New(),
			BaseDir:   baseDir,
		},
		Entry: entry,
	}
}

func (c *CTIMetadataCollector) Collect() (*registry.MetadataRegistry, error) {
	if c.Entry == nil {
		return nil, errors.New("entry point is not set")
	}

	for fragmentName, raw := range c.Entry {
		r := bytes.NewReader(raw)
		head, err := c.readHead(r)
		if err != nil {
			return nil, fmt.Errorf("read head from fragment %s: %w", fragmentName, err)
		}
		var entity metadata.Entity
		switch head {
		case "#%CTI Type v1.0":
			var typ metadata.EntityType
			if err = yaml.Unmarshal(raw, &typ); err != nil {
				return nil, fmt.Errorf("unmarshal type %s: %w", fragmentName, err)
			}
			entity = &typ
		case "#%CTI Instance v1.0":
			var instance metadata.EntityInstance
			if err = yaml.Unmarshal(raw, &instance); err != nil {
				return nil, fmt.Errorf("unmarshal instance %s: %w", fragmentName, err)
			}
			entity = &instance
		default:
			return nil, fmt.Errorf("unknown fragment kind: head: %s", head)
		}
		err = c.Registry.Add(entity)
		if err != nil {
			return nil, fmt.Errorf("add cti entity: %w", err)
		}
	}

	return c.Registry, nil
}

// func (c *CTIMetadataCollector) MakeMetadataType(typ *JSONSchemaCTIType) (*metadata.EntityType, error) {
// 	schema := typ.schema
// 	id := typ.cti
// 	location := typ.sourcePath

// 	// annotations := c.AnnotationsCollector.Collect(schema)

// 	delete(schema, "$schema")
// 	entitySchema := map[string]any{
// 		"$schema":     "http://json-schema.org/draft-07/schema#",
// 		"$ref":        "#/definitions/" + id,
// 		"definitions": map[string]any{id: schema},
// 	}

// 	entity, err := metadata.NewEntityType(
// 		id,
// 		entitySchema,
// 		map[metadata.GJsonPath]*metadata.Annotations{},
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("make entity type: %w", err)
// 	}

// 	originalPath, _ := filepath.Rel(c.BaseDir, location)
// 	// FIXME: sourcePath points to itself or to next parent, if present.
// 	// However, this looks like a workaround rather than a proper solution.
// 	sourcePath, _ := filepath.Rel(c.BaseDir, location)

// 	if title, ok := schema["title"]; ok {
// 		entity.SetDisplayName(title.(string))
// 	} else {
// 		entity.SetDisplayName(id)
// 	}
// 	if description, ok := schema["description"]; ok {
// 		entity.SetDescription(description.(string))
// 	}
// 	if val, ok := schema["x-"+metadata.Final]; ok {
// 		entity.SetFinal(val.(bool))
// 	}
// 	if val, ok := schema["x-"+metadata.Resilient]; ok {
// 		entity.SetResilient(val.(bool))
// 	}
// 	if val, ok := schema["x-"+metadata.Access]; ok {
// 		entity.SetAccess(val.(metadata.AccessModifier))
// 	}
// 	if val, ok := schema["x-"+metadata.Traits+"values"]; ok {
// 		entity.SetTraits(val)
// 	}
// 	if val, ok := schema["x-"+metadata.Traits+"schema"]; ok {
// 		traitsSchema := val.(map[string]any)
// 		// Annotations will be collected later during the transformation phase.
// 		entity.SetTraitsSchema(traitsSchema, map[metadata.GJsonPath]*metadata.Annotations{})
// 	}

// 	entity.SetSourceMap(metadata.EntityTypeSourceMap{
// 		// Name: shape.Name,
// 		EntitySourceMap: metadata.EntitySourceMap{
// 			OriginalPath: filepath.ToSlash(originalPath),
// 			SourcePath:   filepath.ToSlash(sourcePath),
// 		},
// 	})

// 	return entity, nil
// }

// func (c *CTIMetadataCollector) MakeMetadataInstance(instance *JSONSchemaCTIInstance) (*metadata.EntityInstance, error) {
// 	values := instance.values.(map[string]any)
// 	id := instance.cti
// 	valuesLocation := instance.sourcePath

// 	delete(values, "$schema")
// 	entity, err := metadata.NewEntityInstance(id, values)
// 	if err != nil {
// 		return nil, fmt.Errorf("make entity instance: %w", err)
// 	}

// 	if val, ok := values["x-"+metadata.Resilient]; ok {
// 		delete(values, "x-"+metadata.Resilient)
// 		entity.SetResilient(val.(bool))
// 	}
// 	if val, ok := values["x-"+metadata.DisplayName]; ok {
// 		delete(values, "x-"+metadata.DisplayName)
// 		entity.SetDisplayName(val.(string))
// 	}
// 	if val, ok := values["x-"+metadata.Description]; ok {
// 		delete(values, "x-"+metadata.Description)
// 		entity.SetDescription(val.(string))
// 	}
// 	if val, ok := values["x-"+metadata.Access]; ok {
// 		delete(values, "x-"+metadata.Access)
// 		entity.SetAccess(val.(metadata.AccessModifier))
// 	}

// 	originalPath, _ := filepath.Rel(c.BaseDir, valuesLocation)
// 	// reference, _ := filepath.Rel(c.BaseDir, definedBy.Location)

// 	entity.SetSourceMap(metadata.EntityInstanceSourceMap{
// 		AnnotationType: metadata.AnnotationType{
// 			// Name:      definedBy.Name,
// 			// Type:      definedBy.Type,
// 			// Reference: filepath.ToSlash(reference),
// 		},
// 		EntitySourceMap: metadata.EntitySourceMap{
// 			OriginalPath: filepath.ToSlash(originalPath),
// 			SourcePath:   filepath.ToSlash(originalPath),
// 		},
// 	})

// 	return entity, nil
// }

// ReadHead reads, reset file and returns the trimmed first line of a file.
func (c *CTIMetadataCollector) readHead(f io.ReadSeeker) (string, error) {
	r := bufio.NewReader(f)
	head, err := r.ReadBytes('\n')
	if err != nil {
		return "", fmt.Errorf("read fragment head: %w", err)
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("seek to start: %w", err)
	}

	head = bytes.TrimRight(head, "\r\n ")
	return string(head), nil
}

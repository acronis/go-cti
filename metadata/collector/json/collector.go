package collector

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/registry"
)

type JSONSchemaCTIType struct {
	cti        string
	schema     map[string]any
	sourcePath string
}

type JSONSchemaCTIInstance struct {
	cti        string
	values     any
	sourcePath string
}

type JSONCollector struct {
	collector.BaseCollector

	// entryPoint is a map of JSON fragments, where each key is a fragment name
	// and value is JSON Schema which is expressed as map[string]any.
	Entry map[string]map[string]any

	localJSONCTITypes     map[string]*JSONSchemaCTIType
	localJSONCTIInstances map[string]*JSONSchemaCTIInstance
}

func NewJSONSchemaCollector(entry map[string]map[string]any, baseDir string) *JSONCollector {
	return &JSONCollector{
		BaseCollector: collector.BaseCollector{
			CTIParser: cti.NewParser(),
			Registry:  registry.New(),
			BaseDir:   baseDir,
		},
		localJSONCTITypes:     make(map[string]*JSONSchemaCTIType),
		localJSONCTIInstances: make(map[string]*JSONSchemaCTIInstance),
		Entry:                 entry,
	}
}

func (c *JSONCollector) Collect() (*registry.MetadataRegistry, error) {
	if c.Entry == nil {
		return nil, errors.New("entry point is not set")
	}

	for fragmentName, schema := range c.Entry {
		if err := c.readMetadataCti(schema, fragmentName); err != nil {
			return nil, fmt.Errorf("read metadata cti: %w", err)
		}
	}

	for _, typ := range c.localJSONCTITypes {
		// shape, err = c.findAndInsertCtiSchema(shape, make([]string, 0))
		// if err != nil {
		// 	return fmt.Errorf("find and insert cti schema: %w", err)
		// }
		entity, err := c.MakeMetadataType(typ)
		if err != nil {
			return nil, fmt.Errorf("make cti type: %w", err)
		}
		err = c.Registry.Add(entity)
		if err != nil {
			return nil, fmt.Errorf("add cti entity: %w", err)
		}
	}

	for _, instance := range c.localJSONCTIInstances {
		entity, err := c.MakeMetadataInstance(instance)
		if err != nil {
			return nil, fmt.Errorf("make cti type: %w", err)
		}
		err = c.Registry.Add(entity)
		if err != nil {
			return nil, fmt.Errorf("add cti entity: %w", err)
		}
	}

	return c.Registry, nil
}

func (c *JSONCollector) MakeMetadataType(typ *JSONSchemaCTIType) (*metadata.EntityType, error) {
	schema := typ.schema
	id := typ.cti
	location := typ.sourcePath

	// annotations := c.AnnotationsCollector.Collect(schema)

	delete(schema, "$schema")
	entitySchema := map[string]any{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"$ref":        "#/definitions/" + id,
		"definitions": map[string]any{id: schema},
	}

	entity, err := metadata.NewEntityType(
		id,
		entitySchema,
		map[metadata.GJsonPath]*metadata.Annotations{},
	)
	if err != nil {
		return nil, fmt.Errorf("make entity type: %w", err)
	}

	originalPath, _ := filepath.Rel(c.BaseDir, location)
	// FIXME: sourcePath points to itself or to next parent, if present.
	// However, this looks like a workaround rather than a proper solution.
	sourcePath, _ := filepath.Rel(c.BaseDir, location)

	if title, ok := schema["title"]; ok {
		entity.SetDisplayName(title.(string))
	} else {
		entity.SetDisplayName(id)
	}
	if description, ok := schema["description"]; ok {
		entity.SetDescription(description.(string))
	}
	if val, ok := schema["x-"+metadata.Final]; ok {
		entity.SetFinal(val.(bool))
	}
	if val, ok := schema["x-"+metadata.Resilient]; ok {
		entity.SetResilient(val.(bool))
	}
	if val, ok := schema["x-"+metadata.Access]; ok {
		entity.SetAccess(val.(metadata.AccessModifier))
	}
	if val, ok := schema["x-"+metadata.Traits+"values"]; ok {
		entity.SetTraits(val)
	}
	if val, ok := schema["x-"+metadata.Traits+"schema"]; ok {
		traitsSchema := val.(map[string]any)
		// Annotations will be collected later during the transformation phase.
		entity.SetTraitsSchema(traitsSchema, map[metadata.GJsonPath]*metadata.Annotations{})
	}

	entity.SetSourceMap(metadata.EntityTypeSourceMap{
		// Name: shape.Name,
		EntitySourceMap: metadata.EntitySourceMap{
			OriginalPath: filepath.ToSlash(originalPath),
			SourcePath:   filepath.ToSlash(sourcePath),
		},
	})

	return entity, nil
}

func (c *JSONCollector) MakeMetadataInstance(instance *JSONSchemaCTIInstance) (*metadata.EntityInstance, error) {
	values := instance.values.(map[string]any)
	id := instance.cti
	valuesLocation := instance.sourcePath

	delete(values, "$schema")
	entity, err := metadata.NewEntityInstance(id, values)
	if err != nil {
		return nil, fmt.Errorf("make entity instance: %w", err)
	}

	if val, ok := values["x-"+metadata.Resilient]; ok {
		delete(values, "x-"+metadata.Resilient)
		entity.SetResilient(val.(bool))
	}
	if val, ok := values["x-"+metadata.DisplayName]; ok {
		delete(values, "x-"+metadata.DisplayName)
		entity.SetDisplayName(val.(string))
	}
	if val, ok := values["x-"+metadata.Description]; ok {
		delete(values, "x-"+metadata.Description)
		entity.SetDescription(val.(string))
	}
	if val, ok := values["x-"+metadata.Access]; ok {
		delete(values, "x-"+metadata.Access)
		entity.SetAccess(val.(metadata.AccessModifier))
	}

	originalPath, _ := filepath.Rel(c.BaseDir, valuesLocation)
	// reference, _ := filepath.Rel(c.BaseDir, definedBy.Location)

	entity.SetSourceMap(metadata.EntityInstanceSourceMap{
		AnnotationType: metadata.AnnotationType{
			// Name:      definedBy.Name,
			// Type:      definedBy.Type,
			// Reference: filepath.ToSlash(reference),
		},
		EntitySourceMap: metadata.EntitySourceMap{
			OriginalPath: filepath.ToSlash(originalPath),
			SourcePath:   filepath.ToSlash(originalPath),
		},
	})

	return entity, nil
}

func (c *JSONCollector) readMetadataCti(schema map[string]any, fragmentName string) error {
	if val, ok := schema["x-"+metadata.Cti]; ok {
		if err := c.readCtiType(schema, val, fragmentName); err != nil {
			return fmt.Errorf("read cti type: %w", err)
		}
		return nil
	} else if val, ok := schema["x-"+metadata.ID]; ok {
		delete(schema, "x-"+metadata.ID)
		if err := c.readCtiInstance(schema, val, fragmentName); err != nil {
			return fmt.Errorf("read cti instance: %w", err)
		}
		return nil
	}
	return fmt.Errorf("cti.cti or cti.id not found in entity schema in fragment %s", fragmentName)
}

func (c *JSONCollector) readCtiInstance(values map[string]any, annotation any, fragmentName string) error {
	// If cti.cti is not specified, but id is, use it as cti.cti.
	switch v := annotation.(type) {
	case string:
		if _, ok := c.localJSONCTITypes[v]; ok {
			return fmt.Errorf("duplicate cti.id: %s", v)
		}
		if _, err := c.CTIParser.ParseIdentifier(v); err != nil {
			return fmt.Errorf("parse cti.cti: %w", err)
		}
		c.localJSONCTIInstances[v] = &JSONSchemaCTIInstance{
			cti:        v,
			values:     values,
			sourcePath: fragmentName,
		}
		return nil
	default:
		return fmt.Errorf("cti.id must be string, got %T", v)
	}
}

func (c *JSONCollector) readCtiType(schema map[string]any, annotation any, fragmentName string) error {
	var ctis []string
	switch v := annotation.(type) {
	case string:
		ctis = []string{v}
	case []any:
		ctis = make([]string, len(v))
		for i, vv := range v {
			ctis[i] = vv.(string)
		}
	default:
		return fmt.Errorf("cti.cti must be string or array of strings, got %T", v)
	}

	for _, cti := range ctis {
		if _, ok := c.localJSONCTITypes[cti]; ok {
			return fmt.Errorf("duplicate cti.cti: %s", cti)
		}
		if _, err := c.CTIParser.ParseIdentifier(cti); err != nil {
			return fmt.Errorf("parse cti.cti: %w", err)
		}
		c.localJSONCTITypes[cti] = &JSONSchemaCTIType{
			cti:        cti,
			schema:     schema,
			sourcePath: fragmentName,
		}
	}
	return nil
}

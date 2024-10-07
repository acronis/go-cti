package collector

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/identifier"
	"github.com/acronis/go-raml"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/acronis/go-cti/pkg/cti"
)

var annotationsToMove = []string{cti.Reference, cti.Schema}

type CtiRegistry struct {
	Types            cti.EntitiesMap
	Instances        cti.EntitiesMap
	FragmentEntities map[string]cti.Entities
	Total            cti.Entities
	TotalIndex       cti.EntitiesMap
}

func (r *CtiRegistry) Clone() *CtiRegistry {
	c := *r
	return &c
}

type Collector struct {
	Registry             *CtiRegistry
	baseDir              string
	ramlCtiTypes         map[string]*raml.Shape
	raml                 *raml.RAML
	ctiParser            *identifier.Parser
	jsonSchemaConverter  *raml.JSONSchemaConverter
	annotationsCollector *AnnotationsCollector
}

func New(r *raml.RAML, baseDir string) *Collector {
	return &Collector{
		baseDir:              baseDir,
		raml:                 r,
		jsonSchemaConverter:  raml.NewJSONSchemaConverter(raml.WithOmitRefs(true)),
		annotationsCollector: NewAnnotationsCollector(),
		ctiParser:            identifier.NewParser(),
		Registry: &CtiRegistry{
			Types:            make(cti.EntitiesMap),
			Instances:        make(cti.EntitiesMap),
			TotalIndex:       make(cti.EntitiesMap),
			FragmentEntities: make(map[string]cti.Entities),
		},
		ramlCtiTypes: make(map[string]*raml.Shape),
	}
}

func (c *Collector) Collect() error {
	idx, ok := c.raml.EntryPoint().(*raml.Library)
	if !ok {
		return fmt.Errorf("entry point is not a library")
	}
	for pair := idx.Uses.Oldest(); pair != nil; pair = pair.Next() {
		ref := pair.Value
		for pair := ref.Link.Types.Oldest(); pair != nil; pair = pair.Next() {
			shape := pair.Value
			if err := c.readCtiType(shape); err != nil {
				return err
			}
		}
		for pair := ref.Link.CustomDomainProperties.Oldest(); pair != nil; pair = pair.Next() {
			annotation := pair.Value
			if err := c.readAndMakeCtiInstances(annotation); err != nil {
				return err
			}
		}
	}

	for k, shape := range c.ramlCtiTypes {
		err := c.preProcessCtiType(shape)
		if err != nil {
			return fmt.Errorf("preprocess cti type: %w", err)
		}
		entity, err := c.MakeCtiTypeFromShape(k, (*shape).(*raml.ObjectShape))
		if err != nil {
			return err
		}
		if _, ok := c.Registry.TotalIndex[k]; ok {
			return fmt.Errorf("duplicate cti entity %s", k)
		}
		c.Registry.FragmentEntities[entity.SourceMap.OriginalPath] = append(c.Registry.FragmentEntities[entity.SourceMap.OriginalPath], entity)
		c.Registry.TotalIndex[k] = entity
		c.Registry.Types[k] = entity
		c.Registry.Total = append(c.Registry.Total, entity)
	}

	return nil
}

func (c *Collector) MakeCtiTypeFromShape(id string, shape *raml.ObjectShape) (*cti.Entity, error) {
	displayName := shape.Name
	if shape.DisplayName != nil {
		displayName = *shape.DisplayName
	}
	description := ""
	if shape.Description != nil {
		description = *shape.Description
	}
	final := true
	if val, ok := shape.CustomDomainProperties.Get(cti.Final); ok {
		final = val.Extension.Value.(bool)
	}
	var traitsBytes []byte
	if shape.CustomShapeFacets != nil {
		if t, ok := shape.CustomShapeFacets.Get(cti.Traits); ok {
			traitsBytes, _ = json.Marshal(t.Value)
		}
	}
	var traitsSchemaBytes []byte
	var traitsAnnotations map[cti.GJsonPath]cti.Annotations
	if t, ok := shape.CustomShapeFacetDefinitions.Get(cti.Traits); ok {
		traitsSchema := c.jsonSchemaConverter.Convert(*t.Shape)
		traitsSchemaBytes, _ = json.Marshal(traitsSchema)
		traitsAnnotations = c.annotationsCollector.Collect(*t.Shape)
	}
	s, err := c.unwrapCtiType(shape, make([]raml.Shape, 0))
	if err != nil {
		return nil, fmt.Errorf("unwrap cti type: %w", err)
	}
	schema := c.jsonSchemaConverter.Convert(s)
	schemaBytes, _ := json.Marshal(schema)
	annotations := c.annotationsCollector.Collect(s)

	originalPath, _ := filepath.Rel(c.baseDir, shape.Location)
	// FIXME: sourcePath points to itself or to next parent, if present.
	// However, this looks like a workaround rather than a proper solution.
	sourcePath, _ := filepath.Rel(c.baseDir, shape.Location)
	if shape.Inherits != nil {
		sourcePath, _ = filepath.Rel(c.baseDir, (*shape.Inherits[0]).Base().Location)
	}

	entity := &cti.Entity{
		Cti:               id,
		Final:             final,
		DisplayName:       displayName,
		Description:       description,
		Schema:            schemaBytes,
		Traits:            traitsBytes,
		TraitsSchema:      traitsSchemaBytes,
		TraitsAnnotations: traitsAnnotations,
		SourceMap: cti.SourceMap{
			TypeAnnotationReference: cti.TypeAnnotationReference{
				Name: shape.Name,
			},
			OriginalPath: filepath.ToSlash(originalPath),
			SourcePath:   filepath.ToSlash(sourcePath),
		},
		Annotations: annotations,
	}

	return entity, nil
}

func (c *Collector) MakeCtiInstanceFromExtension(id string, definedBy *raml.ArrayShape, values map[string]interface{}, valuesLocation string) *cti.Entity {
	ctiType := (*definedBy.Items).(*raml.ObjectShape)

	valuesBytes, _ := json.Marshal(values)
	displayName := ""
	displayNameProp := c.findPropertyWithAnnotation(ctiType, cti.DisplayName)
	if displayNameProp != nil {
		if _, ok := values[displayNameProp.Name]; ok {
			displayName = values[displayNameProp.Name].(string)
		}
	}

	description := ""
	descriptionProp := c.findPropertyWithAnnotation(ctiType, cti.Description)
	if descriptionProp != nil {
		if _, ok := values[descriptionProp.Name]; ok {
			description = values[descriptionProp.Name].(string)
		}
	}

	originalPath, _ := filepath.Rel(c.baseDir, valuesLocation)
	reference, _ := filepath.Rel(c.baseDir, definedBy.Location)

	return &cti.Entity{
		Final:       true,
		Cti:         id,
		DisplayName: displayName,
		Description: description,
		Values:      valuesBytes,
		SourceMap: cti.SourceMap{
			InstanceAnnotationReference: cti.InstanceAnnotationReference{
				AnnotationType: &cti.AnnotationType{
					Name:      definedBy.Name,
					Type:      definedBy.Type,
					Reference: filepath.ToSlash(reference),
				},
			},
			OriginalPath: filepath.ToSlash(originalPath),
			// SourcePath points to the same path since instance cannot be defined in another file.
			SourcePath: filepath.ToSlash(originalPath),
		},
	}
}

// TODO: Probably move to go-raml
func (c *Collector) traverseShape(shape *raml.Shape, history []raml.Shape, fns []func(*raml.Shape, []raml.Shape) error) error {
	for _, fn := range fns {
		if err := fn(shape, history); err != nil {
			return err
		}
	}
	s := *shape
	for _, h := range history {
		if s.Base().ID == h.Base().ID {
			return nil
		}
	}
	history = append(history, s)

	switch s := s.(type) {
	case *raml.ObjectShape:
		if s.Properties != nil {
			for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
				prop := pair.Value
				if err := c.traverseShape(prop.Shape, history, fns); err != nil {
					return err
				}
			}
		}
		if s.PatternProperties != nil {
			for pair := s.PatternProperties.Oldest(); pair != nil; pair = pair.Next() {
				prop := pair.Value
				if err := c.traverseShape(prop.Shape, history, fns); err != nil {
					return err
				}
			}
		}
	case *raml.ArrayShape:
		if s.Items != nil {
			if err := c.traverseShape(s.Items, history, fns); err != nil {
				return err
			}
		}
	case *raml.UnionShape:
		for _, item := range s.AnyOf {
			if err := c.traverseShape(item, history, fns); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Collector) preProcessCtiType(shape *raml.Shape) error {
	unwrapCtiSchema := func(s *raml.Shape, history []raml.Shape) error {
		base := (*s).Base()
		ctiSchema, ok := base.CustomDomainProperties.Get(cti.Schema)
		if !ok {
			return nil
		}
		switch v := ctiSchema.Extension.Value.(type) {
		case string:
			if _, err := c.ctiParser.Parse(v); err != nil {
				return fmt.Errorf("parse cti.schema: %w", err)
			}
			ss, ok := c.ramlCtiTypes[v]
			if !ok {
				return fmt.Errorf("cti type %s not found", v)
			}
			for _, h := range history {
				if (*ss).Base().ID == h.Base().ID {
					b := (*s).Base()
					*s = &raml.RecursiveShape{
						BaseShape: *b,
						Head:      ss,
					}
					return nil
				}
			}
			us, err := c.raml.UnwrapShape(ss, make([]raml.Shape, 0))
			if err != nil {
				return fmt.Errorf("unwrap cti schema: %w", err)
			}
			us.Base().CustomDomainProperties = base.CustomDomainProperties
			*s = us
		case []interface{}:
			anyOf := make([]*raml.Shape, len(v))
			for i, vv := range v {
				id := vv.(string)
				if _, err := c.ctiParser.Parse(id); err != nil {
					return fmt.Errorf("parse cti.schema[%d]: %w", i, err)
				}
				ss, ok := c.ramlCtiTypes[id]
				if !ok {
					return fmt.Errorf("cti type %s not found", id)
				}
				// History is required to prevent infinite recursion in further processing.
				for _, h := range history {
					if (*ss).Base().ID == h.Base().ID {
						b := (*s).Base()
						*s = &raml.RecursiveShape{
							BaseShape: *b,
							Head:      ss,
						}
						return nil
					}
				}
				us, err := c.raml.UnwrapShape(ss, make([]raml.Shape, 0))
				if err != nil {
					return fmt.Errorf("unwrap cti schema[%d]: %w", i, err)
				}
				us.Base().CustomDomainProperties = orderedmap.New[string, *raml.DomainExtension]()
				anyOf[i] = &us
			}
			us, err := c.raml.MakeConcreteShape(base, raml.TypeUnion, nil)
			if err != nil {
				return fmt.Errorf("make union shape: %w", err)
			}
			us.(*raml.UnionShape).AnyOf = anyOf
			// In-place replacement is fine since all shapes are copied during the unwrap process.
			*s = us
		}
		return nil
	}
	moveAnnotationsToArrayItem := func(s *raml.Shape, _ []raml.Shape) error {
		array, ok := (*s).(*raml.ArrayShape)
		if !ok {
			return nil
		}
		if array.Items == nil {
			return nil
		}
		arrayBase := array.Base()
		itemsBase := (*array.Items).Base()
		// Moving is fine since all shapes are copied during the unwrap process.
		// This does not affect other types.
		for _, annotationName := range annotationsToMove {
			if a, ok := arrayBase.CustomDomainProperties.Get(annotationName); ok {
				itemsBase.CustomDomainProperties.Set(annotationName, a)
				arrayBase.CustomDomainProperties.Delete(annotationName)
			}
		}
		return nil
	}

	return c.traverseShape(shape, make([]raml.Shape, 0), []func(*raml.Shape, []raml.Shape) error{
		moveAnnotationsToArrayItem,
		unwrapCtiSchema,
	})
}

func (c *Collector) readCtiType(shape *raml.Shape) error {
	ctiAnnotation, ok := (*shape).Base().CustomDomainProperties.Get(cti.Cti)
	if !ok {
		return nil
	}
	if _, ok := (*shape).(*raml.ObjectShape); !ok {
		return fmt.Errorf("cti %v must be object", ctiAnnotation.Extension.Value)
	}

	ext := ctiAnnotation.Extension
	switch v := ext.Value.(type) {
	case string:
		id := v
		if _, ok := c.ramlCtiTypes[id]; ok {
			return fmt.Errorf("duplicate cti.cti: %s", id)
		}
		if _, err := c.ctiParser.Parse(id); err != nil {
			return fmt.Errorf("parse cti.cti: %w", err)
		}
		c.ramlCtiTypes[id] = shape
	case []interface{}:
		for i, vv := range v {
			id := vv.(string)
			if _, ok := c.ramlCtiTypes[id]; ok {
				return fmt.Errorf("duplicate cti.cti[%d]: %s", i, id)
			}
			if _, err := c.ctiParser.Parse(id); err != nil {
				return fmt.Errorf("parse cti.cti[%d]: %w", i, err)
			}
			c.ramlCtiTypes[id] = shape
		}
	}
	return nil
}

func (c *Collector) readAndMakeCtiInstances(annotation *raml.DomainExtension) error {
	definedBy, err := c.raml.UnwrapShape(annotation.DefinedBy, make([]raml.Shape, 0))
	if err != nil {
		return fmt.Errorf("unwrap annotation type: %w", err)
	}
	s, ok := definedBy.(*raml.ArrayShape)
	if !ok {
		return fmt.Errorf("annotation is not an array shape")
	}
	items := *s.Items
	ctiAnnotation, ok := items.Base().CustomDomainProperties.Get(cti.Cti)
	if !ok {
		return fmt.Errorf("cti annotation not found")
	}

	parentCti := ctiAnnotation.Extension.Value.(string)
	parentCtiExpr, err := c.ctiParser.Parse(parentCti)
	if err != nil {
		return fmt.Errorf("parse parent cti: %w", err)
	}

	ctiType := items.(*raml.ObjectShape)
	// CTI types are checked before collecting CTI instances.
	// We can be sure that if annotation includes cti.cti, it uses array of objects schema.
	idProp := c.findPropertyWithAnnotation(ctiType, cti.ID)
	if idProp == nil {
		return fmt.Errorf("cti.id not found")
	}
	idKey := idProp.Name

	for _, item := range annotation.Extension.Value.([]interface{}) {
		obj := item.(map[string]interface{})
		id := obj[idKey].(string)

		childCtiExpr, err := c.ctiParser.Parse(id)
		if err != nil {
			return fmt.Errorf("parse child cti: %w", err)
		}
		if _, err := childCtiExpr.Match(parentCtiExpr); err != nil {
			return fmt.Errorf("child cti doesn't match parent cti: %w", err)
		}

		entity := c.MakeCtiInstanceFromExtension(id, s, obj, annotation.Extension.Location)
		if _, ok := c.Registry.TotalIndex[id]; ok {
			return fmt.Errorf("duplicate cti entity %s", id)
		}
		c.Registry.FragmentEntities[entity.SourceMap.OriginalPath] = append(c.Registry.FragmentEntities[entity.SourceMap.OriginalPath], entity)
		c.Registry.TotalIndex[id] = entity
		c.Registry.Instances[id] = entity
		c.Registry.Total = append(c.Registry.Total, entity)
	}
	return nil
}

func (c *Collector) findPropertyWithAnnotation(shape *raml.ObjectShape, annotationName string) *raml.Property {
	// TODO: Suboptimal since we iterate over all annotations every time we look up an annotation.
	for pair := shape.Properties.Oldest(); pair != nil; pair = pair.Next() {
		prop := pair.Value
		if s, ok := (*prop.Shape).(*raml.StringShape); ok {
			if _, ok := s.CustomDomainProperties.Get(annotationName); ok {
				return &prop
			}
		}
	}
	return nil
}

package collector

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

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
	ramlCtiTypes         map[string]*raml.BaseShape
	unwrappedCtiTypes    map[string]*raml.BaseShape
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
		ramlCtiTypes:      make(map[string]*raml.BaseShape),
		unwrappedCtiTypes: make(map[string]*raml.BaseShape),
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
				return fmt.Errorf("read cti type: %w", err)
			}
		}
		for pair := ref.Link.CustomDomainProperties.Oldest(); pair != nil; pair = pair.Next() {
			annotation := pair.Value
			if err := c.readAndMakeCtiInstances(annotation); err != nil {
				return fmt.Errorf("read and make cti instances: %w", err)
			}
		}
	}

	// NOTE: This is a custom pipeline for RAML-CTI types processing.
	// Unwrap implemented in go-raml cannot be used since CTI types require special handling.
	for k, shape := range c.ramlCtiTypes {
		// Create a copy of CTI type and unwrap it using special rules.
		//
		// NOTE: Copy is required since CTI types may share some RAML types.
		// RAML types get modified further (i.e., annotations are moved to some common types)
		// and we don't want to affect other CTI types.
		shape, err := c.unwrapCtiType(shape.CloneDetached())
		if err != nil {
			return fmt.Errorf("unwrap cti type: %w", err)
		}
		_, err = c.raml.FindAndMarkRecursion(shape)
		if err != nil {
			return fmt.Errorf("find and mark recursion: %w", err)
		}
		shape, err = c.preProcessCtiType(shape)
		if err != nil {
			return fmt.Errorf("preprocess cti type: %w", err)
		}
		shape, err = c.findAndInsertCtiSchema(shape, make([]string, 0))
		if err != nil {
			return fmt.Errorf("find and insert cti schema: %w", err)
		}
		entity, err := c.MakeCtiTypeFromShape(k, shape)
		if err != nil {
			return fmt.Errorf("make cti type: %w", err)
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

func (c *Collector) MakeCtiTypeFromShape(id string, shape *raml.BaseShape) (*cti.Entity, error) {
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
		traitsSchema, err := c.jsonSchemaConverter.Convert(t.Base.Shape)
		if err != nil {
			return nil, fmt.Errorf("convert traits schema: %w", err)
		}
		traitsSchemaBytes, _ = json.Marshal(traitsSchema)
		traitsAnnotations = c.annotationsCollector.Collect(t.Base.Shape)
	}
	schema, err := c.jsonSchemaConverter.Convert(shape.Shape)
	if err != nil {
		return nil, fmt.Errorf("convert schema: %w", err)
	}
	schemaBytes, _ := json.Marshal(schema)
	annotations := c.annotationsCollector.Collect(shape.Shape)

	originalPath, _ := filepath.Rel(c.baseDir, shape.Location)
	// FIXME: sourcePath points to itself or to next parent, if present.
	// However, this looks like a workaround rather than a proper solution.
	sourcePath, _ := filepath.Rel(c.baseDir, shape.Location)
	if shape.Inherits != nil {
		sourcePath, _ = filepath.Rel(c.baseDir, shape.Inherits[0].Location)
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
	ctiType := definedBy.Items.Shape.(*raml.ObjectShape)

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

func (c *Collector) findAndInsertCtiSchema(shape *raml.BaseShape, history []string) (*raml.BaseShape, error) {
	ctis, err := c.readCtiCti(shape)
	if err != nil {
		return nil, fmt.Errorf("read cti.cti: %w", err)
	}
	// Using CTI history to prevent infinite recursion over CTI types.
	// This also takes CTI type aliases into account.
	for _, val := range ctis {
		for _, item := range history {
			if strings.HasPrefix(val, item) {
				rs := c.raml.MakeRecursiveShape(shape)
				return rs, nil
			}
		}
		// Make sure we always have a new backing array for history slice.
		historyLen := len(history)
		newHistory := make([]string, historyLen+1)
		copy(newHistory, history)
		newHistory[historyLen] = val
		history = newHistory
	}

	// FIXME: This will keep cti.cti instead of cti.schema. Need to either apply post-processing
	// or change the logic.
	if ctiSchema, ok := shape.CustomDomainProperties.Get(cti.Schema); ok {
		rs, err := c.getCtiSchema(shape, ctiSchema)
		if err != nil {
			return nil, fmt.Errorf("unwrap cti schema: %w", err)
		}
		rs, err = c.findAndInsertCtiSchema(rs, history)
		if err != nil {
			return nil, fmt.Errorf("find and insert cti schema: %w", err)
		}
		return rs, nil
	}

	switch s := shape.Shape.(type) {
	case *raml.ObjectShape:
		if s.Properties != nil {
			for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
				prop := pair.Value
				rs, err := c.findAndInsertCtiSchema(prop.Base, history)
				if err != nil {
					return nil, fmt.Errorf("find and insert cti schema property: %w", err)
				}
				prop.Base = rs
				s.Properties.Set(pair.Key, prop)
			}
		}
		if s.PatternProperties != nil {
			for pair := s.PatternProperties.Oldest(); pair != nil; pair = pair.Next() {
				prop := pair.Value
				rs, err := c.findAndInsertCtiSchema(prop.Base, history)
				if err != nil {
					return nil, fmt.Errorf("find and insert cti schema pattern property: %w", err)
				}
				prop.Base = rs
				s.PatternProperties.Set(pair.Key, prop)
			}
		}
	case *raml.ArrayShape:
		if s.Items != nil {
			rs, err := c.findAndInsertCtiSchema(s.Items, history)
			if err != nil {
				return nil, fmt.Errorf("find and insert cti schema array item: %w", err)
			}
			s.Items = rs
		}
	case *raml.UnionShape:
		for i, member := range s.AnyOf {
			rs, err := c.findAndInsertCtiSchema(member, history)
			if err != nil {
				return nil, fmt.Errorf("find and insert cti schema union member %d: %w", i, err)
			}
			s.AnyOf[i] = rs
		}
	}
	return shape, nil
}

// getOrUnwrapCtiType returns cached unwrapped CTI type. If cache was found, returns it.
// Otherwise, it unwraps the type, puts into cache and returns it.
func (c *Collector) getOrUnwrapCtiType(id string) (*raml.BaseShape, error) {
	if schema, ok := c.unwrappedCtiTypes[id]; ok {
		return schema, nil
	}
	shape, ok := c.ramlCtiTypes[id]
	if !ok {
		return nil, fmt.Errorf("cti type %s not found", id)
	}
	us, err := c.raml.UnwrapShape(shape.CloneDetached())
	if err != nil {
		return nil, fmt.Errorf("unwrap cti type: %w", err)
	}
	c.unwrappedCtiTypes[id] = us
	return us, nil
}

func (c *Collector) preProcessCtiType(shape *raml.BaseShape) (*raml.BaseShape, error) {
	switch s := shape.Shape.(type) {
	case *raml.ObjectShape:
		if s.Properties != nil {
			for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
				prop := pair.Value
				rs, err := c.preProcessCtiType(prop.Base)
				if err != nil {
					return nil, fmt.Errorf("preprocess property: %w", err)
				}
				prop.Base = rs
				s.Properties.Set(pair.Key, prop)
			}
		}
		if s.PatternProperties != nil {
			for pair := s.PatternProperties.Oldest(); pair != nil; pair = pair.Next() {
				prop := pair.Value
				rs, err := c.preProcessCtiType(prop.Base)
				if err != nil {
					return nil, fmt.Errorf("preprocess pattern property: %w", err)
				}
				prop.Base = rs
				s.PatternProperties.Set(pair.Key, prop)
			}
		}
	case *raml.ArrayShape:
		if s.Items != nil {
			c.moveAnnotationsToArrayItem(s)

			rs, err := c.preProcessCtiType(s.Items)
			if err != nil {
				return nil, fmt.Errorf("preprocess array item: %w", err)
			}
			s.Items = rs
		}
	case *raml.UnionShape:
		for i, member := range s.AnyOf {
			rs, err := c.preProcessCtiType(member)
			if err != nil {
				return nil, fmt.Errorf("preprocess union member %d: %w", i, err)
			}
			s.AnyOf[i] = rs
		}
	}
	return shape, nil
}

func (c *Collector) moveAnnotationsToArrayItem(array *raml.ArrayShape) {
	// Moving is fine since all shapes are copied during the unwrap process.
	for _, annotationName := range annotationsToMove {
		if a, ok := array.CustomDomainProperties.Get(annotationName); ok {
			array.Items.CustomDomainProperties.Set(annotationName, a)
			array.CustomDomainProperties.Delete(annotationName)
		}
	}
}

func (c *Collector) getCtiSchema(base *raml.BaseShape, ctiSchema *raml.DomainExtension) (*raml.BaseShape, error) {
	var shape *raml.BaseShape
	switch v := ctiSchema.Extension.Value.(type) {
	case string:
		if _, err := c.ctiParser.Parse(v); err != nil {
			return nil, fmt.Errorf("parse cti.schema: %w", err)
		}
		us, err := c.getOrUnwrapCtiType(v)
		if err != nil {
			return nil, fmt.Errorf("get or unwrap cti schema: %w", err)
		}
		// us.CustomDomainProperties = base.CustomDomainProperties
		shape = us
	case []interface{}:
		anyOf := make([]*raml.BaseShape, len(v))
		for i, vv := range v {
			id := vv.(string)
			if _, err := c.ctiParser.Parse(id); err != nil {
				return nil, fmt.Errorf("parse cti.schema[%d]: %w", i, err)
			}
			us, err := c.getOrUnwrapCtiType(id)
			if err != nil {
				return nil, fmt.Errorf("get or unwrap cti schema: %w", err)
			}
			anyOf[i] = us
		}
		us, err := c.raml.MakeConcreteShapeYAML(base, raml.TypeUnion, nil)
		if err != nil {
			return nil, fmt.Errorf("make union shape: %w", err)
		}
		us.(*raml.UnionShape).AnyOf = anyOf
		base.SetShape(us)
		base.CustomDomainProperties = orderedmap.New[string, *raml.DomainExtension]()
		shape = base
	}
	return shape, nil
}

func (c *Collector) readCtiCti(base *raml.BaseShape) ([]string, error) {
	ctiAnnotation, ok := base.CustomDomainProperties.Get(cti.Cti)
	if !ok {
		return nil, nil
	}
	switch v := ctiAnnotation.Extension.Value.(type) {
	case string:
		return []string{v}, nil
	case []interface{}:
		res := make([]string, len(v))
		for i, vv := range v {
			res[i] = vv.(string)
		}
		return res, nil
	}
	return nil, fmt.Errorf("cti.cti must be string or array of strings")
}

func (c *Collector) readCtiType(base *raml.BaseShape) error {
	ctis, err := c.readCtiCti(base)
	if err != nil {
		return fmt.Errorf("read cti.cti: %w", err)
	}
	if ctis == nil {
		return nil
	}
	if _, ok := base.Shape.(*raml.ObjectShape); !ok {
		return fmt.Errorf("cti type %v must be object", base.Name)
	}

	for _, cti := range ctis {
		if _, ok := c.ramlCtiTypes[cti]; ok {
			return fmt.Errorf("duplicate cti.cti: %s", cti)
		}
		if _, err := c.ctiParser.Parse(cti); err != nil {
			return fmt.Errorf("parse cti.cti: %w", err)
		}
		c.ramlCtiTypes[cti] = base
	}
	return nil
}

func (c *Collector) readAndMakeCtiInstances(annotation *raml.DomainExtension) error {
	definedBy := annotation.DefinedBy
	s, ok := definedBy.Shape.(*raml.ArrayShape)
	if !ok {
		return fmt.Errorf("annotation is not an array shape")
	}
	// NOTE: CTI annotation types are usually aliased.
	items := s.Items.Alias
	if items == nil {
		return fmt.Errorf("items alias is nil")
	}
	ctiAnnotation, ok := items.CustomDomainProperties.Get(cti.Cti)
	if !ok {
		return fmt.Errorf("cti annotation not found")
	}

	parentCti := ctiAnnotation.Extension.Value.(string)
	parentCtiExpr, err := c.ctiParser.Parse(parentCti)
	if err != nil {
		return fmt.Errorf("parse parent cti: %w", err)
	}

	// NOTE: Cannot use getOrUnwrapCtiType because CTI type may not be discovered yet.
	// Use cached CTI type if found, otherwise unwrap it and put into cache.
	if base, ok := c.unwrappedCtiTypes[parentCti]; !ok {
		items, err = c.raml.UnwrapShape(items.CloneDetached())
		if err != nil {
			return fmt.Errorf("unwrap annotation type: %w", err)
		}
		c.unwrappedCtiTypes[parentCti] = items
	} else {
		items = base
	}

	ctiType := items.Shape.(*raml.ObjectShape)
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
		if s, ok := prop.Base.Shape.(*raml.StringShape); ok {
			if _, ok := s.CustomDomainProperties.Get(annotationName); ok {
				return &prop
			}
		}
	}
	return nil
}

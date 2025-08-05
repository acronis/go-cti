package collector

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/acronis/go-cti/metadata/registry"
	"github.com/acronis/go-raml/v2"
)

type RAMLXCollector struct {
	collector.BaseCollector
	Entry *raml.Library

	raml                *raml.RAML
	jsonSchemaConverter *raml.JSONSchemaConverter[*jsonschema.JSONSchemaCTI]

	ramlCtiTypes      map[string]*raml.BaseShape
	unwrappedCtiTypes map[string]*raml.BaseShape
}

func NewRAMLXCollector(r *raml.RAML) (*RAMLXCollector, error) {
	if r == nil {
		return nil, errors.New("raml is nil")
	}
	library, ok := r.EntryPoint().(*raml.Library)
	if !ok {
		return nil, errors.New("entry point is not a library")
	}
	conv, err := raml.NewJSONSchemaConverter(raml.WithWrapper(jsonschema.JSONSchemaWrapper))
	if err != nil {
		return nil, fmt.Errorf("create json schema converter: %w", err)
	}
	return &RAMLXCollector{
		BaseCollector: collector.BaseCollector{
			CTIParser: cti.NewParser(),
			Registry:  registry.New(),
			BaseDir:   filepath.Dir(r.GetLocation()),
		},
		jsonSchemaConverter: conv,
		ramlCtiTypes:        make(map[string]*raml.BaseShape),
		unwrappedCtiTypes:   make(map[string]*raml.BaseShape),
		raml:                r,
		Entry:               library,
	}, nil
}

func (c *RAMLXCollector) Collect() (*registry.MetadataRegistry, error) {
	if c.Entry == nil {
		return nil, errors.New("entry point is not set")
	}
	if c.raml == nil {
		return nil, errors.New("raml is not set")
	}
	for pair := c.Entry.Uses.Oldest(); pair != nil; pair = pair.Next() {
		ref := pair.Value
		for pair := ref.Link.Types.Oldest(); pair != nil; pair = pair.Next() {
			shape := pair.Value
			if err := c.ReadCTIType(shape); err != nil {
				return nil, fmt.Errorf("read cti type: %w", err)
			}
		}
		for pair := ref.Link.CustomDomainProperties.Oldest(); pair != nil; pair = pair.Next() {
			annotation := pair.Value
			if err := c.readAndMakeCtiInstances(annotation); err != nil {
				return nil, fmt.Errorf("read and make cti instances: %w", err)
			}
		}
	}

	// NOTE: This is a custom pipeline for RAML-CTI types processing.
	// Unwrap that is implemented in go-raml cannot be used since CTI types require special handling.
	for k, shape := range c.ramlCtiTypes {
		// Create a copy of CTI type and unwrap it using special rules.
		//
		// NOTE: Copy is required since CTI types may share some RAML types.
		// RAML types get modified further (i.e., annotations are moved to some common types)
		// and we don't want to affect other CTI types.
		shape, err := c.unwrapMetadataType(shape.CloneDetached())
		if err != nil {
			return nil, fmt.Errorf("unwrap cti type: %w", err)
		}
		_, err = c.raml.FindAndMarkRecursion(shape)
		if err != nil {
			return nil, fmt.Errorf("find and mark recursion: %w", err)
		}
		shape, err = c.preProcessCtiType(shape)
		if err != nil {
			return nil, fmt.Errorf("preprocess cti type: %w", err)
		}
		entity, err := c.MakeMetadataType(k, shape)
		if err != nil {
			return nil, fmt.Errorf("make cti type: %w", err)
		}
		err = c.Registry.Add(entity)
		if err != nil {
			return nil, fmt.Errorf("add local cti entity: %w", err)
		}
	}

	return c.Registry, nil
}

func (c *RAMLXCollector) MakeMetadataType(id string, shape *raml.BaseShape) (*metadata.EntityType, error) {
	jsonSchema, err := c.jsonSchemaConverter.Convert(shape.Shape)
	if err != nil {
		return nil, fmt.Errorf("convert schema: %w", err)
	}

	entity, err := metadata.NewEntityType(
		id,
		jsonSchema,
		map[metadata.GJsonPath]*metadata.Annotations{},
	)
	if err != nil {
		return nil, fmt.Errorf("make entity type: %w", err)
	}

	originalPath, _ := filepath.Rel(c.BaseDir, shape.Location)
	// FIXME: sourcePath points to itself or to next parent, if present.
	// However, this looks like a workaround rather than a proper solution.
	sourcePath, _ := filepath.Rel(c.BaseDir, shape.Location)
	if shape.Inherits != nil {
		sourcePath, _ = filepath.Rel(c.BaseDir, shape.Inherits[0].Location)
	}

	if shape.DisplayName != nil {
		entity.SetDisplayName(*shape.DisplayName)
	} else {
		entity.SetDisplayName(shape.Name)
	}
	if shape.Description != nil {
		entity.SetDescription(*shape.Description)
	}
	if val, ok := shape.CustomDomainProperties.Get(consts.Final); ok {
		entity.SetFinal(val.Extension.Value.(bool))
	}
	if val, ok := shape.CustomDomainProperties.Get(consts.Resilient); ok {
		entity.SetResilient(val.Extension.Value.(bool))
	}
	if val, ok := shape.CustomDomainProperties.Get(consts.Access); ok {
		entity.SetAccess(val.Extension.Value.(consts.AccessModifier))
	}
	if shape.CustomShapeFacets != nil {
		if t, ok := shape.CustomShapeFacets.Get(consts.Traits); ok {
			traits, ok := t.Value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("traits must be a map[string]any, got %T", t.Value)
			}
			entity.SetTraits(traits)
		}
	}
	if t, ok := shape.CustomShapeFacetDefinitions.Get(consts.Traits); ok {
		traitsJsonSchema, err := c.jsonSchemaConverter.Convert(t.Base.Shape)
		if err != nil {
			return nil, fmt.Errorf("convert traits schema: %w", err)
		}
		// Annotations will be collected later during the transformation phase.
		entity.SetTraitsSchema(traitsJsonSchema, map[metadata.GJsonPath]*metadata.Annotations{})
	}

	entity.SetSourceMap(metadata.EntityTypeSourceMap{
		Name: shape.Name,
		EntitySourceMap: metadata.EntitySourceMap{
			OriginalPath: filepath.ToSlash(originalPath),
			SourcePath:   filepath.ToSlash(sourcePath),
			Line:         shape.Line,
		},
	})

	return entity, nil
}

func (c *RAMLXCollector) MakeMetadataInstance(
	id string,
	definedBy *raml.ArrayShape,
	extension *raml.DomainExtension,
	values map[string]any,
) (*metadata.EntityInstance, error) {
	entity, err := metadata.NewEntityInstance(id, values)
	if err != nil {
		return nil, fmt.Errorf("make entity instance: %w", err)
	}

	ctiType := definedBy.Items.Shape.(*raml.ObjectShape)

	entity.SetResilient(false) // TODO

	displayNameProp := c.findPropertyWithAnnotation(ctiType, consts.DisplayName)
	if displayNameProp != nil {
		if _, ok := values[displayNameProp.Name]; ok {
			entity.SetDescription(values[displayNameProp.Name].(string))
		}
	}

	descriptionProp := c.findPropertyWithAnnotation(ctiType, consts.Description)
	if descriptionProp != nil {
		if _, ok := values[descriptionProp.Name]; ok {
			entity.SetDescription(values[descriptionProp.Name].(string))
		}
	}

	accessFieldProp := c.findPropertyWithAnnotation(ctiType, consts.AccessField)
	if accessFieldProp != nil {
		if _, ok := values[accessFieldProp.Name]; ok {
			entity.SetAccess(values[accessFieldProp.Name].(consts.AccessModifier))
		}
	}

	originalPath, _ := filepath.Rel(c.BaseDir, extension.Location)
	reference, _ := filepath.Rel(c.BaseDir, definedBy.Location)

	entity.SetSourceMap(metadata.EntityInstanceSourceMap{
		AnnotationType: metadata.AnnotationType{
			Name:      definedBy.Name,
			Type:      definedBy.Type,
			Reference: filepath.ToSlash(reference),
		},
		EntitySourceMap: metadata.EntitySourceMap{
			OriginalPath: filepath.ToSlash(originalPath),
			SourcePath:   filepath.ToSlash(originalPath),
			Line:         extension.Line,
		},
	})

	return entity, nil
}

func (c *RAMLXCollector) preProcessCtiType(shape *raml.BaseShape) (*raml.BaseShape, error) {
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

func (c *RAMLXCollector) moveAnnotationsToArrayItem(array *raml.ArrayShape) {
	copied := false
	for _, annotationName := range collector.AnnotationsToMove {
		if a, ok := array.CustomDomainProperties.Get(annotationName); ok {
			// Need to perform shallow clone since we want to create and modify the base shape of the same underlying type.
			if !copied {
				array.Items = array.Items.CloneShallow()
				copied = true
			}

			array.Items.CustomDomainProperties.Set(annotationName, a)
			array.CustomDomainProperties.Delete(annotationName)
		}
	}
}

func (c *RAMLXCollector) readMetadataCti(base *raml.BaseShape) ([]string, error) {
	ctiAnnotation, ok := base.CustomDomainProperties.Get(consts.CTI)
	if !ok {
		return nil, nil
	}
	switch v := ctiAnnotation.Extension.Value.(type) {
	case string:
		return []string{v}, nil
	case []any:
		res := make([]string, len(v))
		for i, vv := range v {
			res[i] = vv.(string)
		}
		return res, nil
	}
	return nil, errors.New("cti.cti must be string or array of strings")
}

func (c *RAMLXCollector) ReadCTIType(base *raml.BaseShape) error {
	ctis, err := c.readMetadataCti(base)
	if err != nil {
		return fmt.Errorf("read cti.cti: %w", err)
	}
	if ctis == nil {
		return nil
	}

	for _, cti := range ctis {
		if _, ok := c.ramlCtiTypes[cti]; ok {
			return fmt.Errorf("duplicate cti.cti: %s", cti)
		}
		_, err = c.CTIParser.ParseIdentifier(cti)
		if err != nil {
			return fmt.Errorf("parse cti.cti: %w", err)
		}
		c.ramlCtiTypes[cti] = base
	}
	return nil
}

func (c *RAMLXCollector) readAndMakeCtiInstances(annotation *raml.DomainExtension) error {
	definedBy := annotation.DefinedBy
	s, ok := definedBy.Shape.(*raml.ArrayShape)
	if !ok {
		return errors.New("annotation is not an array shape")
	}
	// NOTE: CTI annotation array types are usually aliased.
	items := s.Items.Alias
	if items == nil {
		return errors.New("items alias is nil")
	}
	ctiAnnotation, ok := items.CustomDomainProperties.Get(consts.CTI)
	if !ok {
		return errors.New("cti annotation not found")
	}

	parentCti := ctiAnnotation.Extension.Value.(string)
	parentCtiExpr, err := c.CTIParser.ParseIdentifier(parentCti)
	if err != nil {
		return fmt.Errorf("parse parent cti: %w", err)
	}

	// Use cached CTI type if found, otherwise unwrap it and put into cache.
	if base, ok := c.unwrappedCtiTypes[parentCti]; !ok {
		items, err = c.raml.UnwrapShape(items.CloneDetached())
		if err != nil {
			return fmt.Errorf("unwrap annotation type: %w", err)
		}
		_, err = c.raml.FindAndMarkRecursion(items)
		if err != nil {
			return fmt.Errorf("find and mark recursion: %w", err)
		}
		c.unwrappedCtiTypes[parentCti] = items
	} else {
		items = base
	}

	ctiType, ok := items.Shape.(*raml.ObjectShape)
	if !ok {
		return errors.New("instance must be created from an object type")
	}
	// CTI types are checked before collecting CTI instances.
	// We can be sure that if annotation includes cti.cti, it uses array of objects schema.
	idProp := c.findPropertyWithAnnotation(ctiType, consts.ID)
	if idProp == nil {
		return errors.New("cti.id not found")
	}
	idKey := idProp.Name

	for _, item := range annotation.Extension.Value.([]any) {
		obj := item.(map[string]any)
		id := obj[idKey].(string)

		childCtiExpr, err := c.CTIParser.ParseIdentifier(id)
		if err != nil {
			return fmt.Errorf("parse child cti: %w", err)
		}
		if _, err := childCtiExpr.Match(parentCtiExpr); err != nil {
			return fmt.Errorf("child cti doesn't match parent cti: %w", err)
		}

		entity, err := c.MakeMetadataInstance(id, s, annotation, obj)
		if err != nil {
			return fmt.Errorf("make cti instance: %w", err)
		}
		err = c.Registry.Add(entity)
		if err != nil {
			return fmt.Errorf("add local cti entity: %w", err)
		}
	}
	return nil
}

func (c *RAMLXCollector) findPropertyWithAnnotation(shape *raml.ObjectShape, annotationName string) *raml.Property {
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

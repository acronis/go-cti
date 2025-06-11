package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/attribute_selector"
	"github.com/acronis/go-raml"
)

var annotationsToMove = []string{metadata.Reference, metadata.Schema}

type Collector struct {
	baseDir              string
	raml                 *raml.RAML
	jsonSchemaConverter  *raml.JSONSchemaConverter
	annotationsCollector *AnnotationsCollector

	ctiParser *cti.Parser

	// Local Registry holds entities that are declared by the package.
	LocalRegistry *MetadataRegistry

	// Global Registry holds all entities collected during the session. May include both local and external entities.
	GlobalRegistry *MetadataRegistry

	localRamlCtiTypes  map[string]*raml.BaseShape
	globalRamlCtiTypes map[string]*raml.BaseShape
	unwrappedCtiTypes  map[string]*raml.BaseShape
}

func New() *Collector {
	return &Collector{
		jsonSchemaConverter:  raml.NewJSONSchemaConverter(raml.WithOmitRefs(true)),
		annotationsCollector: NewAnnotationsCollector(),
		ctiParser:            cti.NewParser(),
		LocalRegistry:        NewMetadataRegistry(),
		GlobalRegistry:       NewMetadataRegistry(),
		localRamlCtiTypes:    make(map[string]*raml.BaseShape),
		globalRamlCtiTypes:   make(map[string]*raml.BaseShape),
		unwrappedCtiTypes:    make(map[string]*raml.BaseShape),
	}
}

func (c *Collector) SetRaml(r *raml.RAML) {
	c.raml = r
	c.baseDir = filepath.Dir(r.GetLocation())
	c.localRamlCtiTypes = make(map[string]*raml.BaseShape)
}

func (c *Collector) Collect(isLocal bool) error {
	if c.raml == nil {
		return errors.New("raml is not set")
	}
	idx, ok := c.raml.EntryPoint().(*raml.Library)
	if !ok {
		return errors.New("entry point is not a library")
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
			if err := c.readAndMakeCtiInstances(annotation, isLocal); err != nil {
				return fmt.Errorf("read and make cti instances: %w", err)
			}
		}
	}

	// NOTE: This is a custom pipeline for RAML-CTI types processing.
	// Unwrap implemented in go-raml cannot be used since CTI types require special handling.
	for k, shape := range c.localRamlCtiTypes {
		// Create a copy of CTI type and unwrap it using special rules.
		//
		// NOTE: Copy is required since CTI types may share some RAML types.
		// RAML types get modified further (i.e., annotations are moved to some common types)
		// and we don't want to affect other CTI types.
		shape, err := c.unwrapMetadataType(shape.CloneDetached())
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
		entity, err := c.MakeMetadataType(k, shape)
		if err != nil {
			return fmt.Errorf("make cti type: %w", err)
		}
		err = c.GlobalRegistry.Add(entity.SourceMap.OriginalPath, entity)
		if err != nil {
			return fmt.Errorf("add cti entity: %w", err)
		}
		if isLocal {
			err = c.LocalRegistry.Add(entity.SourceMap.OriginalPath, entity)
			if err != nil {
				return fmt.Errorf("add cti entity: %w", err)
			}
		}
	}

	if err := c.Link(); err != nil {
		return fmt.Errorf("link cti types: %w", err)
	}

	return nil
}

func (c *Collector) Link() error {
	if c.GlobalRegistry == nil {
		return errors.New("global registry is not set")
	}
	for _, object := range c.GlobalRegistry.Index {
		cti := object.GetCti()
		parentID := metadata.GetParentCti(cti)
		if parentID != cti {
			parent, ok := c.GlobalRegistry.Types[parentID]
			if !ok {
				return fmt.Errorf("parent type %s not found", parentID)
			}
			if err := object.SetParent(parent); err != nil {
				return fmt.Errorf("set parent %s for %s: %w", parentID, cti, err)
			}
			// if err := parent.AddChild(object); err != nil {
			// 	return fmt.Errorf("add child %s to %s: %w", cti, parentID, err)
			// }
		}
	}
	return nil
}

func (c *Collector) MakeMetadataType(id string, shape *raml.BaseShape) (*metadata.EntityType, error) {
	jsonSchema, err := c.jsonSchemaConverter.Convert(shape.Shape)
	if err != nil {
		return nil, fmt.Errorf("convert schema: %w", err)
	}
	jsonSchemaBytes, err := json.Marshal(jsonSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal json schema: %w", err)
	}
	var schema map[string]interface{}
	err = json.Unmarshal(jsonSchemaBytes, &schema)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json schema: %w", err)
	}

	annotations := c.annotationsCollector.Collect(shape.Shape)

	entity, err := metadata.NewEntityType(
		id,
		schema,
		annotations,
	)
	if err != nil {
		return nil, fmt.Errorf("make entity type: %w", err)
	}
	// TODO: To remove when go-cti supports raml.JSONSchema merging.
	entity.RawSchema = jsonSchemaBytes

	originalPath, _ := filepath.Rel(c.baseDir, shape.Location)
	// FIXME: sourcePath points to itself or to next parent, if present.
	// However, this looks like a workaround rather than a proper solution.
	sourcePath, _ := filepath.Rel(c.baseDir, shape.Location)
	if shape.Inherits != nil {
		sourcePath, _ = filepath.Rel(c.baseDir, shape.Inherits[0].Location)
	}

	if shape.DisplayName != nil {
		entity.SetDisplayName(*shape.DisplayName)
	} else {
		entity.SetDisplayName(shape.Name)
	}
	if shape.Description != nil {
		entity.SetDescription(*shape.Description)
	}
	if val, ok := shape.CustomDomainProperties.Get(metadata.Final); ok {
		entity.SetFinal(val.Extension.Value.(bool))
	}
	if val, ok := shape.CustomDomainProperties.Get(metadata.Resilient); ok {
		entity.SetResilient(val.Extension.Value.(bool))
	}
	if val, ok := shape.CustomDomainProperties.Get(metadata.Access); ok {
		entity.SetAccess(val.Extension.Value.(metadata.AccessModifier))
	}
	if shape.CustomShapeFacets != nil {
		if t, ok := shape.CustomShapeFacets.Get(metadata.Traits); ok {
			entity.SetTraits(t.Value)
		}
	}
	if t, ok := shape.CustomShapeFacetDefinitions.Get(metadata.Traits); ok {
		traitsJsonSchema, err := c.jsonSchemaConverter.Convert(t.Base.Shape)
		if err != nil {
			return nil, fmt.Errorf("convert traits schema: %w", err)
		}
		// Required to convert *raml.JsonSchema to map[string]interface{}.
		traitsSchemaBytes, err := json.Marshal(traitsJsonSchema)
		if err != nil {
			return nil, fmt.Errorf("marshal traits schema: %w", err)
		}
		var traitsSchema map[string]interface{}
		err = json.Unmarshal(traitsSchemaBytes, &traitsSchema)
		if err != nil {
			return nil, fmt.Errorf("unmarshal traits schema: %w", err)
		}
		traitsAnnotations := c.annotationsCollector.Collect(t.Base.Shape)

		entity.SetTraitsSchema(traitsSchema, traitsAnnotations)
	}

	entity.SetSourceMap(metadata.EntityTypeSourceMap{
		Name: shape.Name,
		EntitySourceMap: metadata.EntitySourceMap{
			OriginalPath: filepath.ToSlash(originalPath),
			SourcePath:   filepath.ToSlash(sourcePath),
		},
	})

	return entity, nil
}

func (c *Collector) MakeMetadataInstance(
	id string,
	definedBy *raml.ArrayShape,
	values map[string]interface{},
	valuesLocation string,
) (*metadata.EntityInstance, error) {
	entity, err := metadata.NewEntityInstance(id, values)
	if err != nil {
		return nil, fmt.Errorf("make entity instance: %w", err)
	}

	ctiType := definedBy.Items.Shape.(*raml.ObjectShape)

	entity.SetResilient(false) // TODO

	displayNameProp := c.findPropertyWithAnnotation(ctiType, metadata.DisplayName)
	if displayNameProp != nil {
		if _, ok := values[displayNameProp.Name]; ok {
			entity.SetDescription(values[displayNameProp.Name].(string))
		}
	}

	descriptionProp := c.findPropertyWithAnnotation(ctiType, metadata.Description)
	if descriptionProp != nil {
		if _, ok := values[descriptionProp.Name]; ok {
			entity.SetDescription(values[descriptionProp.Name].(string))
		}
	}

	accessFieldProp := c.findPropertyWithAnnotation(ctiType, metadata.AccessField)
	if accessFieldProp != nil {
		if _, ok := values[accessFieldProp.Name]; ok {
			entity.SetAccess(metadata.AccessModifier(values[accessFieldProp.Name].(string)))
		}
	}

	originalPath, _ := filepath.Rel(c.baseDir, valuesLocation)
	reference, _ := filepath.Rel(c.baseDir, definedBy.Location)

	entity.SetSourceMap(metadata.EntityInstanceSourceMap{
		AnnotationType: metadata.AnnotationType{
			Name:      definedBy.Name,
			Type:      definedBy.Type,
			Reference: filepath.ToSlash(reference),
		},
		EntitySourceMap: metadata.EntitySourceMap{
			OriginalPath: filepath.ToSlash(originalPath),
			SourcePath:   filepath.ToSlash(originalPath),
		},
	})

	return entity, nil
}

func (c *Collector) findAndInsertCtiSchema(shape *raml.BaseShape, history []string) (*raml.BaseShape, error) {
	ctis, err := c.readMetadataCti(shape)
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

	if ctiSchema, ok := shape.CustomDomainProperties.Get(metadata.Schema); ok {
		rs, err := c.getCtiSchema(shape, ctiSchema, history)
		if err != nil {
			return nil, fmt.Errorf("unwrap cti schema: %w", err)
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
	expr, err := c.ctiParser.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse cti type %s: %w", id, err)
	}
	attributeSelector := string(expr.AttributeSelector)
	// Strip the attribute selector from the ID.
	if attributeSelector != "" {
		id = id[:len(id)-len(attributeSelector)-1]
	}
	as, err := attribute_selector.NewAttributeSelector(attributeSelector)
	if err != nil {
		return nil, fmt.Errorf("parse cti type %s attribute selector: %w", id, err)
	}
	if schema, ok := c.unwrappedCtiTypes[id]; ok {
		return as.WalkBaseShape(schema)
	}
	shape, ok := c.globalRamlCtiTypes[id]
	if !ok {
		return nil, fmt.Errorf("cti type %s not found", id)
	}
	us, err := c.raml.UnwrapShape(shape.CloneDetached())
	if err != nil {
		return nil, fmt.Errorf("unwrap cti type: %w", err)
	}
	_, err = c.raml.FindAndMarkRecursion(us)
	if err != nil {
		return nil, fmt.Errorf("find and mark recursion: %w", err)
	}
	c.unwrappedCtiTypes[id] = us
	return as.WalkBaseShape(us)
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
	copied := false
	for _, annotationName := range annotationsToMove {
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

func (c *Collector) getCtiSchema(base *raml.BaseShape, ctiSchema *raml.DomainExtension, history []string) (*raml.BaseShape, error) {
	var shape *raml.BaseShape
	switch v := ctiSchema.Extension.Value.(type) {
	case string:
		us, err := c.getOrUnwrapCtiType(v)
		if err != nil {
			return nil, fmt.Errorf("get or unwrap cti schema: %w", err)
		}
		us, err = c.findAndInsertCtiSchema(us, history)
		if err != nil {
			return nil, fmt.Errorf("find and insert cti schema: %w", err)
		}
		// This allows keeping the original base unmodified, while branching out to a new shape.
		cus := us.CloneShallow()
		cus.CustomDomainProperties = base.CustomDomainProperties
		shape = cus
	case []interface{}:
		anyOf := make([]*raml.BaseShape, len(v))
		for i, vv := range v {
			id := vv.(string)
			us, err := c.getOrUnwrapCtiType(id)
			if err != nil {
				return nil, fmt.Errorf("get or unwrap cti schema[%d]: %w", i, err)
			}
			us, err = c.findAndInsertCtiSchema(us, history)
			if err != nil {
				return nil, fmt.Errorf("find and insert cti schema[%d]: %w", i, err)
			}
			anyOf[i] = us
		}
		us, err := c.raml.MakeConcreteShapeYAML(base, raml.TypeUnion, nil)
		if err != nil {
			return nil, fmt.Errorf("make union shape: %w", err)
		}
		us.(*raml.UnionShape).AnyOf = anyOf
		base.SetShape(us)
		shape = base
	}
	return shape, nil
}

func (c *Collector) readMetadataCti(base *raml.BaseShape) ([]string, error) {
	ctiAnnotation, ok := base.CustomDomainProperties.Get(metadata.Cti)
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
	return nil, errors.New("cti.cti must be string or array of strings")
}

func (c *Collector) readCtiType(base *raml.BaseShape) error {
	ctis, err := c.readMetadataCti(base)
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
		if _, ok := c.localRamlCtiTypes[cti]; ok {
			return fmt.Errorf("duplicate cti.cti: %s", cti)
		}
		_, err = c.ctiParser.ParseIdentifier(cti)
		if err != nil {
			return fmt.Errorf("parse cti.cti: %w", err)
		}
		c.globalRamlCtiTypes[cti] = base
		c.localRamlCtiTypes[cti] = base
	}
	return nil
}

func (c *Collector) readAndMakeCtiInstances(annotation *raml.DomainExtension, isLocal bool) error {
	definedBy := annotation.DefinedBy
	s, ok := definedBy.Shape.(*raml.ArrayShape)
	if !ok {
		return errors.New("annotation is not an array shape")
	}
	// NOTE: CTI annotation types are usually aliased.
	items := s.Items.Alias
	if items == nil {
		return errors.New("items alias is nil")
	}
	ctiAnnotation, ok := items.CustomDomainProperties.Get(metadata.Cti)
	if !ok {
		return errors.New("cti annotation not found")
	}

	parentCti := ctiAnnotation.Extension.Value.(string)
	parentCtiExpr, err := c.ctiParser.ParseIdentifier(parentCti)
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
		_, err = c.raml.FindAndMarkRecursion(items)
		if err != nil {
			return fmt.Errorf("find and mark recursion: %w", err)
		}
		c.unwrappedCtiTypes[parentCti] = items
	} else {
		items = base
	}

	ctiType := items.Shape.(*raml.ObjectShape)
	// CTI types are checked before collecting CTI instances.
	// We can be sure that if annotation includes cti.cti, it uses array of objects schema.
	idProp := c.findPropertyWithAnnotation(ctiType, metadata.ID)
	if idProp == nil {
		return errors.New("cti.id not found")
	}
	idKey := idProp.Name

	for _, item := range annotation.Extension.Value.([]interface{}) {
		obj := item.(map[string]interface{})
		id := obj[idKey].(string)

		childCtiExpr, err := c.ctiParser.ParseIdentifier(id)
		if err != nil {
			return fmt.Errorf("parse child cti: %w", err)
		}
		if _, err := childCtiExpr.Match(parentCtiExpr); err != nil {
			return fmt.Errorf("child cti doesn't match parent cti: %w", err)
		}

		entity, err := c.MakeMetadataInstance(id, s, obj, annotation.Extension.Location)
		if err != nil {
			return fmt.Errorf("make cti instance: %w", err)
		}
		err = c.GlobalRegistry.Add(entity.SourceMap.OriginalPath, entity)
		if err != nil {
			return fmt.Errorf("add cti entity: %w", err)
		}
		if isLocal {
			err = c.LocalRegistry.Add(entity.SourceMap.OriginalPath, entity)
			if err != nil {
				return fmt.Errorf("add cti entity: %w", err)
			}
		}
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

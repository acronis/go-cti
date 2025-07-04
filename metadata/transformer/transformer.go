package transformer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/acronis/go-cti/metadata/registry"
)

type Transformer struct {
	registry  *registry.MetadataRegistry
	ctiParser *cti.Parser
}

type history []string

func (h history) add(item string) history {
	// Make sure we always have a new backing array for history slice.
	historyLen := len(h)
	newHistory := make([]string, historyLen+1)
	copy(newHistory, h)
	newHistory[historyLen] = item
	return newHistory
}

type context struct {
	// path is the current path in the schema.
	path string

	// entity is the entity type being processed.
	entity *metadata.EntityType

	// history is used to prevent infinite recursion in CTI types.
	history history
}

func New(r *registry.MetadataRegistry) *Transformer {
	return &Transformer{
		registry:  r,
		ctiParser: cti.NewParser(),
	}
}

func (t *Transformer) Transform() error {
	if t.registry == nil {
		return errors.New("registry is not set")
	}
	if err := t.linkEntities(); err != nil {
		return fmt.Errorf("link entities: %w", err)
	}
	if err := t.replaceRefNameWithRefCti(); err != nil {
		return fmt.Errorf("replace ref name with ref cti: %w", err)
	}
	if err := t.mergeSchemas(); err != nil {
		return fmt.Errorf("generate merged schemas: %w", err)
	}
	if err := t.findAndInsertCtiSchemas(); err != nil {
		return fmt.Errorf("find and insert cti schemas: %w", err)
	}
	if err := t.collectAnnotations(); err != nil {
		return fmt.Errorf("collect annotations: %w", err)
	}
	return nil
}

func (t *Transformer) replaceRefNameWithRefCti() error {
	for _, entity := range t.registry.Types {
		cti := entity.GetCti()
		if entity.Schema == nil {
			return fmt.Errorf("entity %s has no schema", cti)
		}
		_, ref, err := jsonschema.ExtractSchemaDefinition(entity.Schema)
		if ref == entity.Cti {
			continue
		}
		if err != nil {
			return fmt.Errorf("extract schema definition for %s: %w", cti, err)
		}
		definitions, ok := entity.Schema["definitions"].(map[string]any)
		if !ok {
			return fmt.Errorf("definitions not found in schema of %s", cti)
		}
		entity.Schema["$ref"] = "#/definitions/" + cti
		definitions[cti] = definitions[ref]
		delete(definitions, ref)
	}
	return nil
}

func (t *Transformer) mergeSchemas() error {
	for _, entity := range t.registry.Types {
		if _, err := entity.GetMergedSchema(); err != nil {
			return fmt.Errorf("get merged schema for %s: %w", entity.GetCti(), err)
		}
	}
	return nil
}

func (t *Transformer) linkEntities() error {
	for _, object := range t.registry.Index {
		cti := object.GetCti()
		parentID := metadata.GetParentCti(cti)
		if parentID == cti {
			continue
		}
		entityID, ok := metadata.GlobalCTITable.Lookup(parentID)
		if !ok {
			return fmt.Errorf("entity %s not found in global CTI table", parentID)
		}
		parent, ok := t.registry.Types[entityID]
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
	return nil
}

func (t *Transformer) collectAnnotations() error {
	annotationsCollector := NewAnnotationsCollector()
	for _, entity := range t.registry.Types {
		if entity.Schema == nil {
			return fmt.Errorf("entity %s has no schema", entity.GetCti())
		}
		schema, _, err := jsonschema.ExtractSchemaDefinition(entity.Schema)
		if err != nil {
			return fmt.Errorf("extract schema definition for %s: %w", entity.GetCti(), err)
		}
		entity.Annotations = annotationsCollector.Collect(schema)
		if entity.TraitsSchema == nil {
			continue
		}
		schema, _, err = jsonschema.ExtractSchemaDefinition(entity.TraitsSchema)
		if err != nil {
			return fmt.Errorf("extract schema definition for %s: %w", entity.GetCti(), err)
		}
		entity.TraitsAnnotations = annotationsCollector.Collect(schema)
	}
	return nil
}

func (t *Transformer) findAndInsertCtiSchemas() error {
	for _, entity := range t.registry.Types {
		if entity.Schema == nil {
			continue
		}
		ctx := context{entity: entity}
		schema, _, err := jsonschema.ExtractSchemaDefinition(entity.Schema)
		if err != nil {
			return fmt.Errorf("extract schema definition for %s: %w", entity.GetCti(), err)
		}
		if _, err = t.findAndInsertCtiSchema(ctx, schema); err != nil {
			return fmt.Errorf("visit schema for %s: %w", entity.GetCti(), err)
		}
	}
	return nil
}

func (t *Transformer) findAndInsertCtiSchema(ctx context, s map[string]any) (map[string]any, error) {
	if s == nil {
		return nil, fmt.Errorf("schema at %s is nil", ctx.path)
	}

	// Using CTI history to prevent infinite recursion over CTI types.
	// This also takes CTI type aliases into account.
	ctis, err := t.readMetadataCti(s)
	if err != nil {
		return nil, fmt.Errorf("read cti.cti: %w", err)
	}

	// TODO: This may produce duplicate definitions if CTI with aliases is used.
	for _, cti := range ctis {
		for _, item := range ctx.history {
			if cti == item {
				defs, ok := ctx.entity.Schema["definitions"].(map[string]any)
				if !ok {
					return nil, fmt.Errorf("definitions not found in schema of %s", ctx.entity.GetCti())
				}
				defs[cti] = s
				newSchema := map[string]any{"$ref": "#/definitions/" + cti}
				for k, v := range s {
					if strings.HasPrefix(k, annotationPrefix) {
						newSchema[k] = v
					}
				}
				return newSchema, nil
			}
		}
		ctx.history = ctx.history.add(cti)
	}

	if val, ok := s[metadata.XSchema]; ok {
		// Inserted CTI schema will contain cti.cti, so we can use it to detect if we already processed it.
		// We don't need to process it again since it's done for each type separately.
		if ctis != nil {
			return s, nil
		}
		return t.getCtiSchema(ctx, val)
	}

	switch {
	case jsonschema.IsAnyOf(s):
		return t.visitAnyOf(ctx, s)
	default:
		// TODO: Support for the list of types
		typ, ok := s["type"].(string)
		if !ok && !jsonschema.IsAny(s) {
			return nil, fmt.Errorf("source schema does not have a valid type: %v", s["type"])
		}
		switch typ {
		case "array":
			return t.visitArray(ctx, s)
		case "object":
			return t.visitObject(ctx, s)
		}
	}
	return s, nil
}

func (t *Transformer) getCtiSchema(ctx context, val any) (map[string]any, error) {
	switch vv := val.(type) {
	case string:
		schema, err := t.resolveCtiSchema(vv)
		if err != nil {
			return nil, fmt.Errorf("get cti schema for %s: %w", vv, err)
		}
		schema, err = t.findAndInsertCtiSchema(ctx, schema)
		if err != nil {
			return nil, fmt.Errorf("find and insert cti schema for %s: %w", vv, err)
		}
		schema = shallowCopy(schema)
		schema[metadata.XSchema] = vv
		return schema, nil
	case []any:
		schemas := make([]any, len(vv))
		for i, v := range vv {
			ref, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("expected string in x-%s, got %T", metadata.Schema, v)
			}
			schema, err := t.resolveCtiSchema(ref)
			if err != nil {
				return nil, fmt.Errorf("get cti schema for %s: %w", ref, err)
			}
			schema, err = t.findAndInsertCtiSchema(ctx, schema)
			if err != nil {
				return nil, fmt.Errorf("find and insert cti schema for %s: %w", ref, err)
			}
			schemas[i] = schema
		}
		return map[string]any{"anyOf": schemas, metadata.Schema: vv}, nil
	default:
		return nil, fmt.Errorf("unexpected type %T for x-%s", vv, metadata.Schema)
	}
}

func (t *Transformer) visitAnyOf(ctx context, s map[string]any) (map[string]any, error) {
	anyOfList, ok := s["anyOf"].([]any)
	if !ok {
		return nil, fmt.Errorf("anyOf at %s is not a list", ctx.path)
	}
	for i, item := range anyOfList {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("anyOf item at %s is not a map", ctx.path)
		}
		newCtx := ctx
		newCtx.path = fmt.Sprintf("%s.anyOf[%d]", ctx.path, i)
		s, err := t.findAndInsertCtiSchema(newCtx, itemMap)
		if err != nil {
			return nil, fmt.Errorf("visit anyOf item %d at %s: %w", i, ctx.path, err)
		}
		anyOfList[i] = s
	}
	return s, nil
}

func (t *Transformer) visitArray(ctx context, s map[string]any) (map[string]any, error) {
	items, ok := s["items"]
	if !ok {
		return s, nil // No items means no further processing needed.
	}
	itemMap, ok := items.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("items at %s is not a map", ctx.path)
	}
	newCtx := ctx
	newCtx.path += ".items"
	newItems, err := t.findAndInsertCtiSchema(newCtx, itemMap)
	if err != nil {
		return nil, fmt.Errorf("visit items at %s: %w", ctx.path, err)
	}
	s["items"] = newItems
	return s, nil
}

func (t *Transformer) visitObject(ctx context, s map[string]any) (map[string]any, error) {
	if props, ok := s["properties"]; ok {
		propsMap, ok := props.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("properties at %s is not a map", ctx.path)
		}
		for k, v := range propsMap {
			propMap, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("property %s at %s is not a map", k, ctx.path)
			}
			newCtx := ctx
			newCtx.path += ".properties." + k
			s, err := t.findAndInsertCtiSchema(newCtx, propMap)
			if err != nil {
				return nil, fmt.Errorf("visit property %s at %s: %w", k, ctx.path, err)
			}
			propsMap[k] = s
		}
	}

	if patternProps, ok := s["patternProperties"]; ok {
		patternPropsMap, ok := patternProps.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("patternProperties at %s is not a map", ctx.path)
		}
		for k, v := range patternPropsMap {
			propMap, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("pattern property %s at %s is not a map", k, ctx.path)
			}
			newCtx := ctx
			newCtx.path += ".patternProperties." + k
			s, err := t.findAndInsertCtiSchema(newCtx, propMap)
			if err != nil {
				return nil, fmt.Errorf("visit pattern property %s at %s: %w", k, ctx.path, err)
			}
			patternPropsMap[k] = s
		}
	}
	return s, nil
}

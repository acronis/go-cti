package transformer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/acronis/go-cti/metadata/registry"
)

type Transformer struct {
	registry  *registry.MetadataRegistry
	ctiParser *cti.Parser
	schemas   map[string]*jsonschema.JSONSchemaCTI
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
		schemas:   make(map[string]*jsonschema.JSONSchemaCTI),
	}
}

func (t *Transformer) Transform() error {
	if t.registry == nil {
		return errors.New("registry is not set")
	}
	if err := t.linkEntities(); err != nil {
		return fmt.Errorf("link entities: %w", err)
	}
	// TODO: Cannot use CTI as ref name because of tilde (~) in CTI names.
	// if err := t.replaceRefNameWithRefCti(); err != nil {
	// 	return fmt.Errorf("replace ref name with ref cti: %w", err)
	// }
	if err := t.mergeSchemas(); err != nil {
		return fmt.Errorf("generate merged schemas: %w", err)
	}
	if err := t.findAndInsertCtiSchemas(); err != nil {
		return fmt.Errorf("find and insert cti schemas: %w", err)
	}
	if err := t.resetCachedSchemas(); err != nil {
		return fmt.Errorf("generate merged schemas: %w", err)
	}
	if err := t.collectAnnotations(); err != nil {
		return fmt.Errorf("collect annotations: %w", err)
	}
	return nil
}

// func (t *Transformer) replaceRefNameWithRefCti() error {
// 	for cti, entity := range t.registry.Types {
// 		if entity.Schema == nil {
// 			return fmt.Errorf("entity %s has no schema", entity.GetCTI())
// 		}
// 		_, ref, err := entity.Schema.GetRefSchema()
// 		if err != nil {
// 			return fmt.Errorf("extract schema definition for %s: %w", entity.GetCTI(), err)
// 		}
// 		if ref == entity.CTI {
// 			continue
// 		}
// 		entity.Schema.Ref = "#/definitions/" + cti
// 		entity.Schema.Definitions[cti] = entity.Schema.Definitions[ref]
// 		delete(entity.Schema.Definitions, ref)
// 	}
// 	return nil
// }

func (t *Transformer) resetCachedSchemas() error {
	for _, entity := range t.registry.Types {
		entity.ResetMergedSchema()
	}
	return nil
}

func (t *Transformer) mergeSchemas() error {
	for _, entity := range t.registry.Types {
		s, err := entity.GetMergedSchema()
		if err != nil {
			return fmt.Errorf("get merged schema for %s: %w", entity.GetCTI(), err)
		}
		t.schemas[entity.CTI] = s
	}
	return nil
}

func (t *Transformer) linkEntities() error {
	for _, object := range t.registry.Index {
		cti := object.GetCTI()
		parentID := metadata.GetParentCTI(cti)
		if parentID == "" {
			continue
		}
		parent, ok := t.registry.Types[parentID]
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
	for cti, entity := range t.registry.Types {
		if entity.Schema == nil {
			return fmt.Errorf("entity %s has no schema", cti)
		}
		schema, _, err := entity.Schema.GetRefSchema()
		if err != nil {
			return fmt.Errorf("extract schema definition for %s: %w", cti, err)
		}
		entity.Annotations = annotationsCollector.Collect(schema)
		if entity.TraitsSchema == nil {
			continue
		}
		schema, _, err = entity.TraitsSchema.GetRefSchema()
		if err != nil {
			return fmt.Errorf("extract schema definition for %s: %w", cti, err)
		}
		entity.TraitsAnnotations = annotationsCollector.Collect(schema)
	}
	return nil
}

func (t *Transformer) findAndInsertCtiSchemas() error {
	for cti, entity := range t.registry.Types {
		if entity.Schema == nil {
			continue
		}
		newEntity := *entity
		newEntity.Schema = newEntity.Schema.DeepCopy()

		ctx := context{entity: &newEntity}
		schema, _, err := newEntity.Schema.GetRefSchema()
		if err != nil {
			return fmt.Errorf("extract schema definition for %s: %w", cti, err)
		}
		if _, err = t.findAndInsertCtiSchema(ctx, schema); err != nil {
			return fmt.Errorf("find and insert cti schema for %s: %w", cti, err)
		}
		entity.Schema = newEntity.Schema
	}
	return nil
}

func (t *Transformer) findAndInsertCtiSchema(ctx context, s *jsonschema.JSONSchemaCTI) (*jsonschema.JSONSchemaCTI, error) {
	if s == nil {
		return nil, fmt.Errorf("schema at %s is nil", ctx.path)
	}

	// Using CTI history to prevent infinite recursion over CTI types.
	// This also takes CTI type aliases into account.
	ctis, err := t.readMetadataCti(s)
	if err != nil {
		return nil, fmt.Errorf("read cti.cti: %w", err)
	}

	// FIXME: This most likely will not work with aliases correctly.
	for _, cti := range ctis {
		for _, item := range ctx.history {
			if cti != item {
				continue
			}
			// If found CTI matches context CTI - that's a self-recursion which may lead directly to root.
			if ctx.entity.CTI == cti {
				return &jsonschema.JSONSchemaCTI{
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Ref: "#"},
					Annotations:       s.Annotations,
				}, nil
			}
			// Otherwise, that's an external recursion and we need to insert the schema into definitions.
			// NOTE: We need to escape the tilde (~) according to JSON Pointer spec.
			escapedCTI := strings.Replace(cti, "~", "~0", -1)
			if _, ok := ctx.entity.Schema.Definitions[cti]; ok {
				// If the schema is already in definitions, we can return a ref to it.
				return &jsonschema.JSONSchemaCTI{
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Ref: "#/definitions/" + escapedCTI},
					Annotations:       s.Annotations,
				}, nil
			}
			ctx.entity.Schema.Definitions[cti] = nil // Initialize with an empty value to reserve the key and avoid recursion.
			ctx.history = make(history, 0)           // Reset history for the new context to keep traversing nested recursion.
			recursiveSchema, err := t.getCtiSchema(ctx, cti)
			if err != nil {
				return nil, fmt.Errorf("find and insert cti schema for %s at %s: %w", cti, ctx.path, err)
			}
			ctx.entity.Schema.Definitions[cti] = recursiveSchema
			return &jsonschema.JSONSchemaCTI{
				JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Ref: "#/definitions/" + escapedCTI},
				Annotations:       s.Annotations,
			}, nil
		}
		ctx.history = ctx.history.add(cti)
	}

	if s.CTISchema != nil {
		return t.getCtiSchema(ctx, s.CTISchema)
	}

	switch {
	case s.IsAnyOf():
		return t.visitAnyOf(ctx, s)
	default:
		switch s.Type {
		case "array":
			return t.visitArray(ctx, s)
		case "object":
			return t.visitObject(ctx, s)
		}
	}
	return s, nil
}

func (t *Transformer) getCtiSchema(ctx context, val any) (*jsonschema.JSONSchemaCTI, error) {
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
		schema.CTISchema = vv
		return schema, nil
	case []any:
		schemas := make([]*jsonschema.JSONSchemaCTI, len(vv))
		for i, v := range vv {
			switch v := v.(type) {
			case string:
				schema, err := t.resolveCtiSchema(v)
				if err != nil {
					return nil, fmt.Errorf("get cti schema for %s: %w", v, err)
				}
				schema, err = t.findAndInsertCtiSchema(ctx, schema)
				if err != nil {
					return nil, fmt.Errorf("find and insert cti schema for %s: %w", v, err)
				}
				schemas[i] = schema
			case nil:
				schemas[i] = &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "null"}}
			default:
				return nil, fmt.Errorf("expected string or nil in x-%s, got %T", consts.Schema, v)
			}
		}
		return &jsonschema.JSONSchemaCTI{
			JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{AnyOf: schemas},
			Annotations:       jsonschema.Annotations{CTISchema: vv},
		}, nil
	default:
		return nil, fmt.Errorf("unexpected type %T for x-%s", vv, consts.Schema)
	}
}

func (t *Transformer) visitAnyOf(ctx context, schema *jsonschema.JSONSchemaCTI) (*jsonschema.JSONSchemaCTI, error) {
	for i, item := range schema.AnyOf {
		newCtx := ctx
		newCtx.path = fmt.Sprintf("%s.anyOf[%d]", ctx.path, i)
		s, err := t.findAndInsertCtiSchema(newCtx, item)
		if err != nil {
			return nil, fmt.Errorf("visit anyOf item %d at %s: %w", i, ctx.path, err)
		}
		schema.AnyOf[i] = s
	}
	return schema, nil
}

func (t *Transformer) visitArray(ctx context, schema *jsonschema.JSONSchemaCTI) (*jsonschema.JSONSchemaCTI, error) {
	if schema.Items == nil {
		return schema, nil // No items means no further processing needed.
	}
	newCtx := ctx
	newCtx.path += ".items"
	newItems, err := t.findAndInsertCtiSchema(newCtx, schema.Items)
	if err != nil {
		return nil, fmt.Errorf("visit items at %s: %w", ctx.path, err)
	}
	schema.Items = newItems
	return schema, nil
}

func (t *Transformer) visitObject(ctx context, schema *jsonschema.JSONSchemaCTI) (*jsonschema.JSONSchemaCTI, error) {
	if schema.Properties != nil {
		for p := schema.Properties.Oldest(); p != nil; p = p.Next() {
			newCtx := ctx
			newCtx.path += ".properties." + p.Key
			s, err := t.findAndInsertCtiSchema(newCtx, p.Value)
			if err != nil {
				return nil, fmt.Errorf("visit property %s at %s: %w", p.Key, ctx.path, err)
			}
			schema.Properties.Set(p.Key, s)
		}
	}

	if schema.PatternProperties != nil {
		for p := schema.PatternProperties.Oldest(); p != nil; p = p.Next() {
			newCtx := ctx
			newCtx.path += ".patternProperties." + p.Key
			s, err := t.findAndInsertCtiSchema(newCtx, p.Value)
			if err != nil {
				return nil, fmt.Errorf("visit pattern property %s at %s: %w", p.Key, ctx.path, err)
			}
			schema.PatternProperties.Set(p.Key, s)
		}
	}
	return schema, nil
}

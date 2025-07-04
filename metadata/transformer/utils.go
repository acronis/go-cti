package transformer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/attribute_selector"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

func shallowCopy(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		if strings.HasPrefix(key, annotationPrefix) {
			output[key] = jsonschema.DeepCopy(value)
		} else {
			output[key] = value
		}
	}
	return output
}

func (t *Transformer) readMetadataCti(s map[string]any) ([]string, error) {
	// If schema has no reference to CTI schema, return it as is.
	val, ok := s[metadata.XCti]
	if !ok {
		return nil, nil
	}
	switch v := val.(type) {
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

func (t *Transformer) resolveCtiSchema(ref string) (map[string]any, error) {
	expr, err := t.ctiParser.Parse(ref)
	if err != nil {
		return nil, fmt.Errorf("parse cti type %s: %w", ref, err)
	}
	attributeSelector := string(expr.AttributeSelector)
	// Strip the attribute selector from the ID.
	if attributeSelector != "" {
		ref = ref[:len(ref)-len(attributeSelector)-1]
	}
	entityID, ok := metadata.GlobalCTITable.Lookup(ref)
	if !ok {
		return nil, fmt.Errorf("entity %s not found in global CTI table", ref)
	}
	entity, ok := t.registry.Types[entityID]
	if !ok {
		return nil, fmt.Errorf("cti type %s not found in registry", ref)
	}
	as, err := attribute_selector.NewAttributeSelector(attributeSelector)
	if err != nil {
		return nil, fmt.Errorf("parse cti type %s attribute selector: %w", ref, err)
	}
	schema, err := entity.GetMergedSchema()
	if err != nil {
		return nil, fmt.Errorf("get merged schema for cti type %s: %w", ref, err)
	}
	schema, _, err = jsonschema.ExtractSchemaDefinition(schema)
	if err != nil {
		return nil, fmt.Errorf("extract schema definition for cti type %s: %w", ref, err)
	}
	return as.WalkJSONSchema(schema)
}

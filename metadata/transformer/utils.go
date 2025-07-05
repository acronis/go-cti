package transformer

import (
	"errors"
	"fmt"

	"github.com/acronis/go-cti/metadata/attribute_selector"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

func (t *Transformer) readMetadataCti(s *jsonschema.JSONSchemaCTI) ([]string, error) {
	if s.Annotations == nil || s.CTICTI == nil {
		return nil, nil
	}
	switch v := s.CTICTI.(type) {
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

func (t *Transformer) resolveCtiSchema(ref string) (*jsonschema.JSONSchemaCTI, error) {
	expr, err := t.ctiParser.Parse(ref)
	if err != nil {
		return nil, fmt.Errorf("parse cti type %s: %w", ref, err)
	}
	attributeSelector := string(expr.AttributeSelector)
	// Strip the attribute selector from the ID.
	if attributeSelector != "" {
		ref = ref[:len(ref)-len(attributeSelector)-1]
	}
	entity, ok := t.registry.Types[ref]
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
	schema, _, err = schema.GetRefSchema()
	if err != nil {
		return nil, fmt.Errorf("extract schema definition for cti type %s: %w", ref, err)
	}
	return as.WalkJSONSchema(schema)
}

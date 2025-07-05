package attribute_selector

import (
	"fmt"
	"strings"

	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/acronis/go-raml/v2"
)

type AttributeSelector struct {
	Path []string
}

// NewAttributeSelector converts "foo.bar.baz" to []string{"foo","bar","baz"}.
func NewAttributeSelector(q string) (*AttributeSelector, error) {
	// If query is empty, assume root selector.
	if q == "" {
		return &AttributeSelector{Path: []string{}}, nil
	}
	parts := strings.Split(q, ".")
	for i, p := range parts {
		if p == "" {
			return nil, fmt.Errorf("empty token at position %d", i)
		}
	}
	return &AttributeSelector{Path: parts}, nil
}

func (as *AttributeSelector) WalkJSON(v map[string]any) (any, error) {
	var cur any = v
	for _, tok := range as.Path {
		switch node := cur.(type) {
		case map[string]any:
			next, ok := node[tok]
			if !ok {
				return nil, fmt.Errorf("key %q not found", tok)
			}
			cur = next
		default:
			return nil, fmt.Errorf("cannot descend via %q into %T", tok, cur)
		}
	}
	return cur, nil
}

func (as *AttributeSelector) WalkJSONSchema(v *jsonschema.JSONSchemaCTI) (*jsonschema.JSONSchemaCTI, error) {
	// TODO: May need to support walking $ref links and more complex structures.
	cur := v
	for _, tok := range as.Path {
		if cur.Properties == nil {
			return nil, fmt.Errorf("cannot descend into %T", cur)
		}
		property, ok := cur.Properties.Get(tok)
		if !ok {
			return nil, fmt.Errorf("key %q not found", tok)
		}
		cur = property
	}
	return cur, nil
}

func (as *AttributeSelector) WalkBaseShape(v *raml.BaseShape) (*raml.BaseShape, error) {
	cur := v
	for _, tok := range as.Path {
		switch node := cur.Shape.(type) {
		case *raml.ObjectShape:
			next, ok := node.Properties.Get(tok)
			if !ok {
				return nil, fmt.Errorf("key %q not found", tok)
			}
			cur = next.Base
		default:
			return nil, fmt.Errorf("cannot descend via %q into %T", tok, cur)
		}
	}
	return cur, nil
}

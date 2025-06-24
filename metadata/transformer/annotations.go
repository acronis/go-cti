package transformer

import (
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

const metadataPrefix = "cti."
const annotationPrefix = jsonschema.XAnnotationKey + metadataPrefix

type AnnotationsCollector struct {
	annotations map[metadata.GJsonPath]*metadata.Annotations
}

func NewAnnotationsCollector() *AnnotationsCollector {
	return &AnnotationsCollector{}
}

func (c *AnnotationsCollector) Collect(s map[string]any) map[metadata.GJsonPath]*metadata.Annotations {
	c.annotations = make(map[metadata.GJsonPath]*metadata.Annotations)
	c.Visit(".", s)
	return c.annotations
}

func (c *AnnotationsCollector) Visit(ctx string, s map[string]any) {
	c.collectAnnotations(ctx, s)

	t, ok := s["type"]
	if !ok {
		if s["anyOf"] != nil {
			c.VisitUnionShape(ctx, s)
		}
	} else {
		switch t {
		case "object":
			c.VisitObjectShape(ctx, s)
		case "array":
			c.VisitArrayShape(ctx, s)
		}
	}
}

func (c *AnnotationsCollector) VisitObjectShape(ctx string, s map[string]any) any {
	if ctx != "." {
		ctx += "."
	}

	if props, ok := s["properties"]; ok {
		for k, v := range props.(map[string]any) {
			c.Visit(ctx+k, v.(map[string]any))
		}
	}

	if patternProps, ok := s["patternProperties"]; ok {
		for k, v := range patternProps.(map[string]any) {
			c.Visit(ctx+k, v.(map[string]any))
		}
	}
	return nil
}

func (c *AnnotationsCollector) VisitArrayShape(ctx string, s map[string]any) {
	if ctx == "." {
		ctx += "#"
	} else {
		ctx += ".#"
	}

	if items, ok := s["items"]; ok {
		c.Visit(ctx, items.(map[string]any))
	}
}

func (c *AnnotationsCollector) VisitUnionShape(ctx string, s map[string]any) {
	for _, item := range s["anyOf"].([]any) {
		c.Visit(ctx, item.(map[string]any))
	}
}

func (c *AnnotationsCollector) collectAnnotations(ctx string, s map[string]any) {
	annotations, ok := s[jsonschema.XCustomKey].(map[string]any)
	if !ok {
		return
	}
	filtered := make(map[string]any, 0)
	for k, v := range annotations {
		if strings.HasPrefix(k, annotationPrefix) {
			annotationName := strings.TrimPrefix(k, jsonschema.XAnnotationKey)
			if annotationName == k {
				// Not a metadata annotation, skip it.
				continue
			}
			filtered[annotationName] = v
		}
	}
	if len(filtered) == 0 {
		return
	}
	key := metadata.GJsonPath(ctx)
	item := c.annotations[key]
	if item == nil {
		item = &metadata.Annotations{}
		c.annotations[key] = item
	}
	for name, value := range filtered {
		switch name {
		case metadata.Cti:
			item.Cti = value
		case metadata.Final:
			v := value.(bool)
			item.Final = &v
		case metadata.Access:
			item.Access = value.(metadata.AccessModifier)
		case metadata.Resilient:
			v := value.(bool)
			item.Resilient = &v
		case metadata.ID:
			v := value.(bool)
			item.ID = &v
		case metadata.L10n:
			v := value.(bool)
			item.L10N = &v
		case metadata.Asset:
			v := value.(bool)
			item.Asset = &v
		case metadata.Overridable:
			v := value.(bool)
			item.Overridable = &v
		case metadata.Reference:
			item.Reference = value
		case metadata.Schema:
			item.Schema = value
		case metadata.Meta:
			item.Meta = value.(string)
		case metadata.DisplayName:
			v := value.(bool)
			item.DisplayName = &v
		case metadata.Description:
			v := value.(bool)
			item.Description = &v
		case metadata.PropertyNames:
			item.PropertyNames = value.(map[string]any)
		}
	}
	c.annotations[metadata.GJsonPath(ctx)] = item
	// TODO: Optional cleanup?
	// delete(s, jsonschema.XCustomKey)
}

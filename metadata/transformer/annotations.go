package transformer

import (
	"strings"

	"github.com/acronis/go-cti/metadata"
)

const annotationPrefix = "x-cti."

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
			c.VisitAnyOf(ctx, s)
		}
	} else {
		switch t {
		case "object":
			c.VisitObject(ctx, s)
		case "array":
			c.VisitArray(ctx, s)
		}
	}
}

func (c *AnnotationsCollector) VisitObject(ctx string, s map[string]any) any {
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

func (c *AnnotationsCollector) VisitArray(ctx string, s map[string]any) {
	if ctx == "." {
		ctx += "#"
	} else {
		ctx += ".#"
	}

	if items, ok := s["items"]; ok {
		c.Visit(ctx, items.(map[string]any))
	}
}

func (c *AnnotationsCollector) VisitAnyOf(ctx string, s map[string]any) {
	for _, item := range s["anyOf"].([]any) {
		c.Visit(ctx, item.(map[string]any))
	}
}

func (c *AnnotationsCollector) collectAnnotations(ctx string, s map[string]any) {
	filtered := make(map[string]any)
	for k := range s {
		if strings.HasPrefix(k, annotationPrefix) {
			filtered[k] = s[k]
		}
	}
	if len(filtered) == 0 {
		return // No CTI annotations found, nothing to do.
	}
	key := metadata.GJsonPath(ctx)
	item := c.annotations[key]
	if item == nil {
		item = &metadata.Annotations{}
		c.annotations[key] = item
	}
	for name, value := range filtered {
		switch name {
		case metadata.XCti:
			item.Cti = value
		case metadata.XFinal:
			v := value.(bool)
			item.Final = &v
		case metadata.XAccess:
			item.Access = value.(metadata.AccessModifier)
		case metadata.XResilient:
			v := value.(bool)
			item.Resilient = &v
		case metadata.XID:
			v := value.(bool)
			item.ID = &v
		case metadata.XL10n:
			v := value.(bool)
			item.L10N = &v
		case metadata.XAsset:
			v := value.(bool)
			item.Asset = &v
		case metadata.XOverridable:
			v := value.(bool)
			item.Overridable = &v
		case metadata.XReference:
			item.Reference = value
		case metadata.XSchema:
			item.Schema = value
		case metadata.XMeta:
			item.Meta = value.(string)
		case metadata.XDisplayName:
			v := value.(bool)
			item.DisplayName = &v
		case metadata.XDescription:
			v := value.(bool)
			item.Description = &v
		case metadata.XPropertyNames:
			item.PropertyNames = value.(map[string]any)
		}
	}
	c.annotations[key] = item
	// TODO: Optional cleanup?
}

package annotations_collector

import (
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

type AnnotationsCollector struct {
	annotations map[metadata.GJsonPath]*metadata.Annotations
}

func New() *AnnotationsCollector {
	return &AnnotationsCollector{}
}

func (c *AnnotationsCollector) Collect(s *jsonschema.JSONSchemaCTI) map[metadata.GJsonPath]*metadata.Annotations {
	c.annotations = make(map[metadata.GJsonPath]*metadata.Annotations)
	c.Visit(".", s)
	return c.annotations
}

func (c *AnnotationsCollector) Visit(pth string, s *jsonschema.JSONSchemaCTI) {
	c.collectAnnotations(pth, s)

	switch {
	case s.IsAnyOf():
		c.VisitAnyOf(pth, s)
	case s.Type == "object":
		c.VisitObject(pth, s)
	case s.Type == "array":
		c.VisitArray(pth, s)
	}
}

func (c *AnnotationsCollector) VisitObject(pth string, s *jsonschema.JSONSchemaCTI) any {
	if pth != "." {
		pth += "."
	}

	if s.Properties != nil {
		for p := s.Properties.Oldest(); p != nil; p = p.Next() {
			c.Visit(pth+p.Key, p.Value)
		}
	}

	if s.PatternProperties != nil {
		for p := s.PatternProperties.Oldest(); p != nil; p = p.Next() {
			c.Visit(pth+p.Key, p.Value)
		}
	}
	return nil
}

func (c *AnnotationsCollector) VisitArray(pth string, s *jsonschema.JSONSchemaCTI) {
	if pth == "." {
		pth += "#"
	} else {
		pth += ".#"
	}

	if s.Items != nil {
		c.Visit(pth, s.Items)
	}
}

func (c *AnnotationsCollector) VisitAnyOf(pth string, s *jsonschema.JSONSchemaCTI) {
	for _, item := range s.AnyOf {
		c.Visit(pth, item)
	}
}

func (c *AnnotationsCollector) collectAnnotations(pth string, s *jsonschema.JSONSchemaCTI) {
	key := metadata.GJsonPath(pth)
	item, ok := c.annotations[key]
	if !ok {
		item = &metadata.Annotations{}
	}

	changed := false
	if s.CTIID != nil {
		item.ID = s.CTIID
		changed = true
	}
	if s.CTIL10N != nil {
		item.L10N = s.CTIL10N
		changed = true
	}
	if s.CTIAsset != nil {
		item.Asset = s.CTIAsset
		changed = true
	}
	if s.CTIOverridable != nil {
		item.Overridable = s.CTIOverridable
		changed = true
	}
	if s.CTIReference != nil {
		item.Reference = s.CTIReference
		changed = true
	}
	if s.CTISchema != nil {
		item.Schema = s.CTISchema
		changed = true
	}
	if s.CTIMeta != "" {
		item.Meta = s.CTIMeta
		changed = true
	}
	if s.CTIDisplayName != nil {
		item.DisplayName = s.CTIDisplayName
		changed = true
	}
	if s.CTIDescription != nil {
		item.Description = s.CTIDescription
		changed = true
	}
	if s.CTIPropertyNames != nil {
		item.PropertyNames = s.CTIPropertyNames
		changed = true
	}
	if changed {
		c.annotations[key] = item
	}
	// TODO: Optional cleanup?
}

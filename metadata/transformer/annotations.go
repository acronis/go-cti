package transformer

import (
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

type AnnotationsCollector struct {
	annotations map[metadata.GJsonPath]*metadata.Annotations
}

func NewAnnotationsCollector() *AnnotationsCollector {
	return &AnnotationsCollector{}
}

func (c *AnnotationsCollector) Collect(s *jsonschema.JSONSchemaCTI) map[metadata.GJsonPath]*metadata.Annotations {
	c.annotations = make(map[metadata.GJsonPath]*metadata.Annotations)
	c.Visit(".", s)
	return c.annotations
}

func (c *AnnotationsCollector) Visit(ctx string, s *jsonschema.JSONSchemaCTI) {
	c.collectAnnotations(ctx, s)

	if s.IsAnyOf() {
		c.VisitAnyOf(ctx, s)
	} else {
		switch s.Type {
		case "object":
			c.VisitObject(ctx, s)
		case "array":
			c.VisitArray(ctx, s)
		}
	}
}

func (c *AnnotationsCollector) VisitObject(ctx string, s *jsonschema.JSONSchemaCTI) any {
	if ctx != "." {
		ctx += "."
	}

	if s.Properties != nil {
		for p := s.Properties.Oldest(); p != nil; p = p.Next() {
			c.Visit(ctx+p.Key, p.Value)
		}
	}

	if s.PatternProperties != nil {
		for p := s.PatternProperties.Oldest(); p != nil; p = p.Next() {
			c.Visit(ctx+p.Key, p.Value)
		}
	}
	return nil
}

func (c *AnnotationsCollector) VisitArray(ctx string, s *jsonschema.JSONSchemaCTI) {
	if ctx == "." {
		ctx += "#"
	} else {
		ctx += ".#"
	}

	if s.Items != nil {
		c.Visit(ctx, s.Items)
	}
}

func (c *AnnotationsCollector) VisitAnyOf(ctx string, s *jsonschema.JSONSchemaCTI) {
	for _, item := range s.AnyOf {
		c.Visit(ctx, item)
	}
}

func (c *AnnotationsCollector) collectAnnotations(ctx string, s *jsonschema.JSONSchemaCTI) {
	if s.Annotations == nil {
		return // No annotations to collect.
	}
	key := metadata.GJsonPath(ctx)
	item := c.annotations[key]
	if item == nil {
		item = &metadata.Annotations{}
		c.annotations[key] = item
	}
	if s.CTICTI != nil {
		item.CTI = s.CTICTI
	}
	if s.CTIFinal != nil {
		item.Final = s.CTIFinal
	}
	if s.CTIAccess != "" {
		item.Access = s.CTIAccess
	}
	if s.CTIResilient != nil {
		item.Resilient = s.CTIResilient
	}
	if s.CTIID != nil {
		item.ID = s.CTIID
	}
	if s.CTIL10N != nil {
		item.L10N = s.CTIL10N
	}
	if s.CTIAsset != nil {
		item.Asset = s.CTIAsset
	}
	if s.CTIOverridable != nil {
		item.Overridable = s.CTIOverridable
	}
	if s.CTIReference != nil {
		item.Reference = s.CTIReference
	}
	if s.CTISchema != nil {
		item.Schema = s.CTISchema
	}
	if s.CTIMeta != "" {
		item.Meta = s.CTIMeta
	}
	if s.CTIDisplayName != nil {
		item.DisplayName = s.CTIDisplayName
	}
	if s.CTIDescription != nil {
		item.Description = s.CTIDescription
	}
	if s.CTIPropertyNames != nil {
		item.PropertyNames = s.CTIPropertyNames
	}
	c.annotations[key] = item
	// TODO: Optional cleanup?
}

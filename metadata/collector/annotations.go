package collector

import (
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-raml"
)

const MetadataPrefix = "cti."

type AnnotationsCollector struct {
	annotations map[metadata.GJsonPath]metadata.Annotations
}

func NewAnnotationsCollector() *AnnotationsCollector {
	return &AnnotationsCollector{}
}

func (c *AnnotationsCollector) Collect(s raml.Shape) map[metadata.GJsonPath]metadata.Annotations {
	c.annotations = make(map[metadata.GJsonPath]metadata.Annotations)
	c.Visit(".", s)
	return c.annotations
}

func (c *AnnotationsCollector) Visit(ctx string, s raml.Shape) {
	c.collectAnnotations(ctx, s.Base())

	switch s := s.(type) {
	case *raml.ObjectShape:
		c.VisitObjectShape(ctx, s)
	case *raml.ArrayShape:
		c.VisitArrayShape(ctx, s)
	case *raml.UnionShape:
		c.VisitUnionShape(ctx, s)
	}
}

func (c *AnnotationsCollector) VisitObjectShape(ctx string, s *raml.ObjectShape) any {
	if ctx != "." {
		ctx += "."
	}

	if s.Properties != nil {
		for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
			v := pair.Value
			c.Visit(ctx+v.Name, v.Base.Shape)
		}
	}
	if s.PatternProperties != nil {
		for pair := s.PatternProperties.Oldest(); pair != nil; pair = pair.Next() {
			k, v := pair.Key, pair.Value
			c.Visit(ctx+k, v.Base.Shape)
		}
	}
	return nil
}

func (c *AnnotationsCollector) VisitArrayShape(ctx string, s *raml.ArrayShape) any {
	if ctx == "." {
		ctx += "#"
	} else {
		ctx += ".#"
	}

	if s.Items != nil {
		c.Visit(ctx, s.Items.Shape)
	}
	return nil
}

func (c *AnnotationsCollector) VisitUnionShape(ctx string, s *raml.UnionShape) any {
	for _, item := range s.AnyOf {
		c.Visit(ctx, item.Shape)
	}
	return nil
}

func (c *AnnotationsCollector) collectAnnotations(ctx string, s *raml.BaseShape) {
	filtered := make([]*raml.DomainExtension, 0)
	for pair := s.CustomDomainProperties.Oldest(); pair != nil; pair = pair.Next() {
		annotation := pair.Value
		if strings.HasPrefix(annotation.Name, MetadataPrefix) {
			filtered = append(filtered, annotation)
		}
	}
	if len(filtered) == 0 {
		return
	}
	item := c.annotations[metadata.GJsonPath(ctx)]
	for _, annotation := range filtered {
		switch annotation.Name {
		case metadata.Cti:
			item.Cti = annotation.Extension.Value
		case metadata.Final:
			v := annotation.Extension.Value.(bool)
			item.Final = &v
		case metadata.Resilient:
			v := annotation.Extension.Value.(bool)
			item.Resilient = &v
		case metadata.ID:
			v := annotation.Extension.Value.(bool)
			item.ID = &v
		case metadata.L10n:
			v := annotation.Extension.Value.(bool)
			item.L10N = &v
		case metadata.Asset:
			v := annotation.Extension.Value.(bool)
			item.Asset = &v
		case metadata.Overridable:
			v := annotation.Extension.Value.(bool)
			item.Overridable = &v
		case metadata.Reference:
			item.Reference = annotation.Extension.Value
		case metadata.Schema:
			item.Schema = annotation.Extension.Value
		case metadata.Meta:
			item.Meta = annotation.Extension.Value.(string)
		case metadata.DisplayName:
			v := annotation.Extension.Value.(bool)
			item.DisplayName = &v
		case metadata.Description:
			v := annotation.Extension.Value.(bool)
			item.Description = &v
		case metadata.PropertyNames:
			item.PropertyNames = annotation.Extension.Value.(map[string]interface{})
		}
	}
	c.annotations[metadata.GJsonPath(ctx)] = item
}

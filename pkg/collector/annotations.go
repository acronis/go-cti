package collector

import (
	"strings"

	"github.com/acronis/go-raml"

	"github.com/acronis/go-cti/pkg/cti"
)

type AnnotationsCollector struct {
	annotations map[cti.GJsonPath]cti.Annotations
}

func NewAnnotationsCollector() *AnnotationsCollector {
	return &AnnotationsCollector{}
}

func (c *AnnotationsCollector) Collect(s raml.Shape) map[cti.GJsonPath]cti.Annotations {
	c.annotations = make(map[cti.GJsonPath]cti.Annotations)
	c.Visit("", s)
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
	if ctx != "" {
		ctx += "."
	}

	if s.Properties != nil {
		for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
			v := pair.Value
			c.Visit(ctx+v.Name, *v.Shape)
		}
	}
	if s.PatternProperties != nil {
		for pair := s.PatternProperties.Oldest(); pair != nil; pair = pair.Next() {
			k, v := pair.Key, pair.Value
			c.Visit(ctx+k, *v.Shape)
		}
	}
	return nil
}

func (c *AnnotationsCollector) VisitArrayShape(ctx string, s *raml.ArrayShape) any {
	if ctx == "" {
		ctx += "#"
	} else {
		ctx += ".#"
	}

	if s.Items != nil {
		c.Visit(ctx, *s.Items)
	}
	return nil
}

func (c *AnnotationsCollector) VisitUnionShape(ctx string, s *raml.UnionShape) any {
	for _, item := range s.AnyOf {
		c.Visit(ctx, *item)
	}
	return nil
}

func (c *AnnotationsCollector) collectAnnotations(ctx string, s *raml.BaseShape) {
	filtered := make([]*raml.DomainExtension, 0)
	for pair := s.CustomDomainProperties.Oldest(); pair != nil; pair = pair.Next() {
		annotation := pair.Value
		if strings.HasPrefix(annotation.Name, "cti.") {
			filtered = append(filtered, annotation)
		}
	}
	if len(filtered) == 0 {
		return
	}
	item := c.annotations[cti.GJsonPath(ctx)]
	for _, annotation := range filtered {
		switch annotation.Name {
		case cti.Cti:
			item.Cti = annotation.Extension.Value
		case cti.Final:
			v := annotation.Extension.Value.(bool)
			item.Final = &v
		case cti.ID:
			v := annotation.Extension.Value.(bool)
			item.ID = &v
		case cti.L10n:
			v := annotation.Extension.Value.(bool)
			item.L10N = &v
		case cti.Asset:
			v := annotation.Extension.Value.(bool)
			item.Asset = &v
		case cti.Overridable:
			v := annotation.Extension.Value.(bool)
			item.Overridable = &v
		case cti.Reference:
			item.Reference = annotation.Extension.Value
		case cti.Schema:
			item.Schema = annotation.Extension.Value
		case cti.Meta:
			item.Meta = annotation.Extension.Value.(string)
		case cti.DisplayName:
			v := annotation.Extension.Value.(bool)
			item.DisplayName = &v
		case cti.Description:
			v := annotation.Extension.Value.(bool)
			item.Description = &v
		case cti.PropertyNames:
			item.PropertyNames = annotation.Extension.Value.(map[string]interface{})
		}
	}
	c.annotations[cti.GJsonPath(ctx)] = item
}

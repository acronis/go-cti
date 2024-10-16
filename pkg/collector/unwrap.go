package collector

import (
	"fmt"

	"github.com/acronis/go-raml"

	"github.com/acronis/go-cti/pkg/cti"
)

func (c *Collector) unwrapCtiType(base *raml.BaseShape) (*raml.BaseShape, error) {
	s := base.Shape
	if s == nil {
		return nil, fmt.Errorf("shape is nil")
	}
	objShape, ok := s.(*raml.ObjectShape)
	if !ok {
		return nil, fmt.Errorf("CTI type must be object shape")
	}

	if base.ShapeVisited {
		base.SetUnwrapped()
		return base, nil
	}
	base.ShapeVisited = true

	var source *raml.BaseShape
	if base.Alias != nil {
		us, err := c.unwrapCtiType(base.Alias)
		if err != nil {
			return nil, fmt.Errorf("alias unwrap: %w", err)
		}
		return base.AliasTo(us), nil
	}

	if base.Link != nil {
		us, err := c.unwrapCtiType(base.Link.Shape)
		if err != nil {
			return nil, fmt.Errorf("link unwrap: %w", err)
		}
		source = us

		base.Link = nil
	} else if len(base.Inherits) > 0 {
		inherits := base.Inherits
		if len(inherits) > 1 {
			return nil, fmt.Errorf("multiple inheritance is not supported")
		}
		parent := inherits[0]
		if _, ok := parent.CustomDomainProperties.Get(cti.Cti); !ok {
			ss, err := c.unwrapCtiType(parent)
			if err != nil {
				return nil, fmt.Errorf("parent unwrap: %w", err)
			}
			source = ss
		}
	}

	if objShape.Properties != nil {
		for pair := objShape.Properties.Oldest(); pair != nil; pair = pair.Next() {
			prop := pair.Value
			us, err := c.raml.UnwrapShape(prop.Shape)
			if err != nil {
				return nil, fmt.Errorf("object property unwrap: %w", err)
			}
			prop.Shape = us
			objShape.Properties.Set(pair.Key, prop)
		}
	}

	for pair := base.CustomShapeFacetDefinitions.Oldest(); pair != nil; pair = pair.Next() {
		prop := pair.Value
		us, err := c.raml.UnwrapShape(prop.Shape)
		if err != nil {
			return nil, fmt.Errorf("custom shape facet definition unwrap: %w", err)
		}
		prop.Shape = us
		base.CustomShapeFacetDefinitions.Set(pair.Key, prop)
	}

	if source != nil {
		is, errInherit := base.Inherit(source)
		if errInherit != nil {
			return nil, fmt.Errorf("merge shapes: %w", errInherit)
		}
		base.ShapeVisited = false
		base.SetUnwrapped()
		return is, nil
	}
	base.ShapeVisited = false
	base.SetUnwrapped()
	return base, nil
}

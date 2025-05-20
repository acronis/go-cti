package collector

import (
	"fmt"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-raml"
)

func (c *Collector) unwrapMetadataType(base *raml.BaseShape) (*raml.BaseShape, error) {
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
		us, err := c.unwrapMetadataType(base.Alias)
		if err != nil {
			return nil, fmt.Errorf("alias unwrap: %w", err)
		}
		return base.AliasTo(us)
	}

	switch {
	case base.Link != nil:
		us, err := c.unwrapMetadataType(base.Link.Shape)
		if err != nil {
			return nil, fmt.Errorf("link unwrap: %w", err)
		}
		source = us

		base.Link = nil
	case len(base.Inherits) == 1:
		parent := base.Inherits[0]
		// Continue unwrapping non-CTI type
		if _, ok := parent.CustomDomainProperties.Get(metadata.Cti); !ok {
			ss, err := c.unwrapMetadataType(parent)
			if err != nil {
				return nil, fmt.Errorf("parent unwrap: %w", err)
			}
			source = ss
		}
	case len(base.Inherits) > 1:
		inherits := base.Inherits
		ctiInherits, ramlInherits := c.splitMultipleInherits(inherits)
		if len(ctiInherits) > 1 {
			return nil, fmt.Errorf("multiple CTI inheritance is not supported")
		}
		if len(ramlInherits) > 0 {
			ss, err := c.inheritMultipleRamlParents(ramlInherits)
			if err != nil {
				return nil, fmt.Errorf("inherit multiple raml parents: %w", err)
			}
			source = ss
		}
	}

	if objShape.Properties != nil {
		for pair := objShape.Properties.Oldest(); pair != nil; pair = pair.Next() {
			prop := pair.Value
			us, err := c.raml.UnwrapShape(prop.Base)
			if err != nil {
				return nil, fmt.Errorf("object property unwrap: %w", err)
			}
			prop.Base = us
			objShape.Properties.Set(pair.Key, prop)
		}
	}

	for pair := base.CustomShapeFacetDefinitions.Oldest(); pair != nil; pair = pair.Next() {
		prop := pair.Value
		us, err := c.raml.UnwrapShape(prop.Base)
		if err != nil {
			return nil, fmt.Errorf("custom shape facet definition unwrap: %w", err)
		}
		prop.Base = us
		base.CustomShapeFacetDefinitions.Set(pair.Key, prop)
	}

	if source != nil {
		is, errInherit := base.Inherit(source)
		if errInherit != nil {
			return nil, fmt.Errorf("merge shapes: %w", errInherit)
		}
		is.ShapeVisited = false
		is.SetUnwrapped()
		return is, nil
	}
	base.ShapeVisited = false
	base.SetUnwrapped()
	return base, nil
}

func (c *Collector) inheritMultipleRamlParents(inherits []*raml.BaseShape) (*raml.BaseShape, error) {
	ss, err := c.unwrapMetadataType(inherits[0])
	if err != nil {
		return nil, fmt.Errorf("parent unwrap: %w", err)
	}
	for i := 1; i < len(inherits); i++ {
		us, err := c.unwrapMetadataType(inherits[i])
		if err != nil {
			return nil, fmt.Errorf("parent unwrap: %w", err)
		}
		_, err = ss.Inherit(us)
		if err != nil {
			return nil, fmt.Errorf("inherit shapes: %w", err)
		}
	}
	return ss, nil
}

func (c *Collector) splitMultipleInherits(inherits []*raml.BaseShape) ([]*raml.BaseShape, []*raml.BaseShape) {
	ramlInherits := make([]*raml.BaseShape, 0, len(inherits))
	ctiInherits := make([]*raml.BaseShape, 0, len(inherits))
	// Multiple parents are aliased
	for _, inherit := range inherits {
		if _, ok := inherit.Alias.CustomDomainProperties.Get(metadata.Cti); ok {
			ctiInherits = append(ctiInherits, inherit.Alias)
		} else {
			ramlInherits = append(ramlInherits, inherit.Alias)
		}
	}
	return ctiInherits, ramlInherits
}

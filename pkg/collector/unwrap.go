package collector

import (
	"fmt"

	"github.com/acronis/go-raml"

	"github.com/acronis/go-cti/pkg/cti"
)

func (c *Collector) unwrapCtiType(s *raml.ObjectShape, history []raml.Shape) (raml.Shape, error) {
	if s == nil {
		return nil, fmt.Errorf("shape is nil")
	}

	base := s.Base()
	for _, item := range history {
		if item.Base().Id == base.Id {
			return nil, fmt.Errorf("CTI type cannot be recursive")
		}
	}
	history = append(history, s)

	var source raml.Shape
	if base.Alias != nil {
		us, err := c.unwrapCtiType((*base.Alias).(*raml.ObjectShape), history)
		if err != nil {
			return nil, fmt.Errorf("alias unwrap: %w", err)
		}
		// Alias simply points to another shape, so we just change the name and return it as is.
		us.Base().Name = base.Name
		return us, nil
	}
	if base.Link != nil {
		us, err := c.unwrapCtiType((*base.Link.Shape).(*raml.ObjectShape), history)
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
		parent := (*inherits[0]).Clone().(*raml.ObjectShape)
		if _, ok := parent.Base().CustomDomainProperties.Get(cti.Cti); !ok {
			ss, err := c.unwrapCtiType(parent, history)
			if err != nil {
				return nil, fmt.Errorf("parent unwrap: %w", err)
			}
			source = ss
		}
	}

	if s.Properties != nil {
		for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
			prop := pair.Value
			us, err := c.raml.UnwrapShape(prop.Shape, history)
			if err != nil {
				return nil, fmt.Errorf("object property unwrap: %w", err)
			}
			*prop.Shape = us
		}
	}

	for pair := base.CustomShapeFacetDefinitions.Oldest(); pair != nil; pair = pair.Next() {
		prop := pair.Value
		us, err := c.raml.UnwrapShape(prop.Shape, history)
		if err != nil {
			return nil, fmt.Errorf("custom shape facet definition unwrap: %w", err)
		}
		*prop.Shape = us
	}

	if source != nil {
		ms, err := c.raml.Inherit(source, s)
		if err != nil {
			return nil, fmt.Errorf("merge shapes: %w", err)
		}
		return ms, nil
	}
	return s, nil
}

package bunman

import (
	"fmt"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/validator"
)

func Validate(bd *bundle.Bundle) error {
	r, err := ParseWithCache(bd)
	if err != nil {
		return fmt.Errorf("parse with cache: %w", err)
	}
	validator := validator.MakeCtiValidator()
	validator.LoadFromRegistry(r)

	if err := validator.ValidateAll(); err != nil {
		return fmt.Errorf("validate all: %w", err)
	}

	return nil
}

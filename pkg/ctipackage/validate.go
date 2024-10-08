package ctipackage

import (
	"fmt"

	"github.com/acronis/go-cti/pkg/validator"
)

func (pkg *Package) Validate() error {
	r, err := pkg.ParseWithCache()
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

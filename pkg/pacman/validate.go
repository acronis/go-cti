package pacman

import (
	"fmt"

	"github.com/acronis/go-cti/pkg/ctipackage"
	"github.com/acronis/go-cti/pkg/validator"
)

func Validate(pkg *ctipackage.Package) error {
	r, err := ParseWithCache(pkg)
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

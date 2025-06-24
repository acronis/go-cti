package ctipackage

import (
	"fmt"

	"github.com/acronis/go-cti/metadata/validator"
)

func (pkg *Package) Validate() error {
	// TODO: Validate must use cache.
	err := pkg.Parse()
	if err != nil {
		return fmt.Errorf("parse with cache: %w", err)
	}

	validator := validator.MakeMetadataValidator(pkg.GlobalRegistry, pkg.LocalRegistry)
	if err = validator.ValidateAll(); err != nil {
		return fmt.Errorf("validate all: %w", err)
	}

	return nil
}

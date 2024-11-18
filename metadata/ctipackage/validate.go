package ctipackage

import (
	"fmt"

	"github.com/acronis/go-cti/metadata/validator"
)

func (pkg *Package) Validate() error {
	r, err := pkg.ParseWithCache()
	if err != nil {
		return fmt.Errorf("parse with cache: %w", err)
	}
	validator := validator.MakeMetadataValidator()
	validator.LoadFromRegistry(r)

	if err := validator.ValidateAll(); err != nil {
		return fmt.Errorf("validate all: %w", err)
	}

	return nil
}

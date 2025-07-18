package ctipackage

import (
	"fmt"

	"github.com/acronis/go-cti/metadata/validator"
)

func (pkg *Package) Validate() error {
	err := pkg.Parse()
	if err != nil {
		return fmt.Errorf("parse with cache: %w", err)
	}

	validator := validator.New(pkg.Index.Vendor, pkg.Index.Pkg, pkg.GlobalRegistry, pkg.LocalRegistry)
	if err = validator.ValidateAll(); err != nil {
		return fmt.Errorf("validate all: %w", err)
	}

	return nil
}

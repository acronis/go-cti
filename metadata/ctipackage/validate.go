package ctipackage

import (
	"fmt"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/validator"
)

func ValidateEntity(v *validator.MetadataValidator, e *metadata.EntityType) error {
	traits := e.FindTraitsInChain()
	// Find the schema for traits in the chain.
	schema := e.FindTraitsSchemaInChain()

	// Extract ref schema
	schema, _, err := schema.GetRefSchema()
	if err != nil {
		return err
	}
	// Get the sub-schema for error_reporting trait
	errorReportingSchema, ok := schema.Properties.Get("error_reporting")
	if !ok {
		return fmt.Errorf("error_reporting property not found in schema")
	}

	if traits == nil {
		return nil
	}

	// Traits schema is map[string]any.
	val, ok := traits.(map[string]any)
	if !ok {
		return fmt.Errorf("traits is not a map[string]any, got %T", traits)
	}

	errorReportingVal, ok := val["error_reporting"]
	// Assuming that default is a string and is a legacy value
	if !ok || ok && errorReportingVal == errorReportingSchema.Default {
		_, errorOk := val["error"]
		if errorOk {
			return fmt.Errorf("entity has legacy error_reporting property but error property is also present")
		}
	}

	return nil
}

func (pkg *Package) Validate() error {
	// TODO: Validate must use cache.
	err := pkg.Parse()
	if err != nil {
		return fmt.Errorf("parse with cache: %w", err)
	}

	validator := validator.MakeMetadataValidator(pkg.Index.Vendor, pkg.Index.Pkg, pkg.GlobalRegistry, pkg.LocalRegistry)
	validator.OnType("cti.a.p.dts.func.v1.0", ValidateEntity)
	if err = validator.ValidateAll(); err != nil {
		return fmt.Errorf("validate all: %w", err)
	}

	return nil
}

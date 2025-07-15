package compatibility

import (
	"testing"

	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/acronis/go-cti/metadata/testsupp"
	"github.com/stretchr/testify/require"
)

const oldPackageIndexJSON = `{
    "package_id": "x.y",
    "ramlx_version": "1.0",
    "entities": [
      "types.raml"
    ]
}`

const oldPackageTypesRAML = `#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    properties:
      val:
        type: cti.CTI
`

const newPackageIndexJSON = `{
    "package_id": "x.y",
    "ramlx_version": "1.0",
    "entities": [
      "types.raml"
    ]
}`

const newPackageTypesRAML = `#%RAML 1.0 Library

uses:
  cti: .ramlx/cti.raml

types:
  SampleEntity:
    (cti.cti): cti.x.y.sample_entity.v1.0
    properties:
      val:
        type: string
        pattern: "^[a-zA-Z0-9]+$"
`

func TestCheckPackagesCompatibility(t *testing.T) {
	oldPackagePath := testsupp.InitTestPackageFiles(t, testsupp.PackageTestCase{
		Name:     "old_package",
		PkgId:    "x.y",
		Entities: []string{"types.raml"},
		Files: map[string]string{
			"index.json": oldPackageIndexJSON,
			"types.raml": oldPackageTypesRAML,
		},
	})
	newPackagePath := testsupp.InitTestPackageFiles(t, testsupp.PackageTestCase{
		Name:     "new_package",
		PkgId:    "x.y",
		Entities: []string{"types.raml"},
		Files: map[string]string{
			"index.json": newPackageIndexJSON,
			"types.raml": newPackageTypesRAML,
		},
	})

	// Create and parse the old package
	oldPkg, err := ctipackage.New(oldPackagePath)
	if err != nil {
		t.Fatalf("Failed to create old package: %v", err)
	}
	if err = oldPkg.Read(); err != nil {
		t.Fatalf("Failed to read old package: %v", err)
	}
	if err = oldPkg.Parse(); err != nil {
		t.Fatalf("Failed to parse old package: %v", err)
	}

	// Create and parse the new package
	newPkg, err := ctipackage.New(newPackagePath)
	if err != nil {
		t.Fatalf("Failed to create new package: %v", err)
	}
	if err = newPkg.Read(); err != nil {
		t.Fatalf("Failed to read new package: %v", err)
	}
	if err = newPkg.Parse(); err != nil {
		t.Fatalf("Failed to parse new package: %v", err)
	}

	// Create a compatibility checker
	checker := NewCompatibilityChecker()

	// Check compatibility between the packages
	err = checker.CheckPackagesCompatibility(oldPkg, newPkg)
	require.NoError(t, err)

	// We expect an error because the type of 'val' property changed from string to integer
	if checker.Pass {
		t.Errorf("Expected compatibility check to fail, but it succeeded")
	} else {
		t.Logf("Compatibility check failed as expected: %v", checker.Messages)
	}

	// Test with nil packages
	if err = checker.CheckPackagesCompatibility(nil, newPkg); err == nil {
		t.Errorf("Expected error when old package is nil, but got nil")
	}

	if err = checker.CheckPackagesCompatibility(oldPkg, nil); err == nil {
		t.Errorf("Expected error when new package is nil, but got nil")
	}

	// Test with unparsed packages
	unparsedPkg, err := ctipackage.New(oldPackagePath)
	if err != nil {
		t.Fatalf("Failed to create unparsed package: %v", err)
	}
	if err = unparsedPkg.Read(); err != nil {
		t.Fatalf("Failed to read unparsed package: %v", err)
	}

	if err = checker.CheckPackagesCompatibility(unparsedPkg, newPkg); err == nil {
		t.Errorf("Expected error when old package is not parsed, but got nil")
	}

	if err = checker.CheckPackagesCompatibility(oldPkg, unparsedPkg); err == nil {
		t.Errorf("Expected error when new package is not parsed, but got nil")
	}
}

func TestCheckPackagesCompatibilityWithSamePackages(t *testing.T) {
	oldPackagePath := testsupp.InitTestPackageFiles(t, testsupp.PackageTestCase{
		Name:     "old_package",
		PkgId:    "x.y",
		Entities: []string{"types.raml"},
		Files: map[string]string{
			"index.json": oldPackageIndexJSON,
			"types.raml": oldPackageTypesRAML,
		},
	})

	// Create and parse the old package twice
	oldPkg1, err := ctipackage.New(oldPackagePath)
	if err != nil {
		t.Fatalf("Failed to create old package 1: %v", err)
	}
	if err := oldPkg1.Read(); err != nil {
		t.Fatalf("Failed to read old package 1: %v", err)
	}
	if err := oldPkg1.Parse(); err != nil {
		t.Fatalf("Failed to parse old package 1: %v", err)
	}

	oldPkg2, err := ctipackage.New(oldPackagePath)
	if err != nil {
		t.Fatalf("Failed to create old package 2: %v", err)
	}
	if err := oldPkg2.Read(); err != nil {
		t.Fatalf("Failed to read old package 2: %v", err)
	}
	if err := oldPkg2.Parse(); err != nil {
		t.Fatalf("Failed to parse old package 2: %v", err)
	}

	// Create a compatibility checker
	checker := NewCompatibilityChecker()

	// Check compatibility between the same packages
	err = checker.CheckPackagesCompatibility(oldPkg1, oldPkg2)
	require.NoError(t, err)

	// We expect no error because the packages are identical
	if !checker.Pass {
		t.Errorf("Expected compatibility check to succeed, but it failed: %v", checker.Messages)
	} else {
		t.Logf("Compatibility check succeeded as expected")
	}
}

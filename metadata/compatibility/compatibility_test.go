package compatibility

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/stretchr/testify/require"
)

func TestCheckPackagesCompatibility(t *testing.T) {
	// Get absolute paths to the old and new package directories
	oldPackagePath, err := filepath.Abs("old_package")
	if err != nil {
		t.Fatalf("Failed to get absolute path for old_package: %v", err)
	}

	newPackagePath, err := filepath.Abs("new_package")
	if err != nil {
		t.Fatalf("Failed to get absolute path for new_package: %v", err)
	}

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
	checker := &CompatibilityChecker{}

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
	// Get absolute path to the old package directory
	oldPackagePath, err := filepath.Abs("old_package")
	if err != nil {
		t.Fatalf("Failed to get absolute path for old_package: %v", err)
	}

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
	checker := &CompatibilityChecker{}

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

func TestValidationSummaryTemplate(t *testing.T) {
	// Create a compatibility checker with some test messages
	checker := &CompatibilityChecker{
		Messages: []Message{
			{Severity: SeverityError, Message: "Error message 1"},
			{Severity: SeverityError, Message: "Error message 2"},
			{Severity: SeverityWarning, Message: "Warning message 1"},
			{Severity: SeverityInfo, Message: "Info message 1"},
		},
		Pass: false,
	}

	// Get the validation failed template
	template := checker.ValidationSummaryTemplate()

	// Check that the template contains the expected content
	if !strings.Contains(template, "# Compatibility Check Failed ❌") {
		t.Errorf("Expected template to contain '## Validation Failed', but it didn't")
	}
	if !strings.Contains(template, "**Errors:** 2") {
		t.Errorf("Expected template to contain '**Errors:** 2', but it didn't")
	}
	if !strings.Contains(template, "**Warnings:** 1") {
		t.Errorf("Expected template to contain '**Warnings:** 1', but it didn't")
	}
	if !strings.Contains(template, "**Info:** 1") {
		t.Errorf("Expected template to contain '**Info:** 1', but it didn't")
	}
	if !strings.Contains(template, "Error message 1") {
		t.Errorf("Expected template to contain 'Error message 1', but it didn't")
	}
	if !strings.Contains(template, "Error message 2") {
		t.Errorf("Expected template to contain 'Error message 2', but it didn't")
	}
}

func TestDiffReportTemplate(t *testing.T) {
	// Create entity types for testing
	newEntity1, err := metadata.NewEntityType("cti.vendor.package.new.entity.1.v1.0", map[string]interface{}{}, map[metadata.GJsonPath]*metadata.Annotations{})
	if err != nil {
		t.Fatalf("Failed to create new entity 1: %v", err)
	}

	newEntity2, err := metadata.NewEntityType("cti.vendor.package.new.entity.2.v1.0", map[string]interface{}{}, map[metadata.GJsonPath]*metadata.Annotations{})
	if err != nil {
		t.Fatalf("Failed to create new entity 2: %v", err)
	}

	removedEntity, err := metadata.NewEntityType("cti.vendor.package.removed.entity.1.v1.0", map[string]interface{}{}, map[metadata.GJsonPath]*metadata.Annotations{})
	if err != nil {
		t.Fatalf("Failed to create removed entity: %v", err)
	}

	modifiedEntity, err := metadata.NewEntityType("cti.vendor.package.modified.entity.1.v1.0", map[string]interface{}{}, map[metadata.GJsonPath]*metadata.Annotations{})
	if err != nil {
		t.Fatalf("Failed to create modified entity: %v", err)
	}

	// Create a compatibility checker with test data
	checker := &CompatibilityChecker{
		NewEntities: []metadata.Entity{
			newEntity1,
			newEntity2,
		},
		RemovedEntities: []metadata.Entity{
			removedEntity,
		},
		ModifiedEntities: []EntityDiff{
			{
				Entity:   modifiedEntity,
				Messages: []string{"Changed property X", "Removed property Y"},
			},
		},
		Messages: []Message{
			{Severity: SeverityError, Message: "Error message 1"},
			{Severity: SeverityWarning, Message: "Warning message 1"},
			{Severity: SeverityInfo, Message: "Info message 1"},
		},
	}

	// Get the diff report template
	template := checker.DiffReportTemplate()

	// Check that the template contains the expected content
	if !strings.Contains(template, "# Compatibility Diff Report") {
		t.Errorf("Expected template to contain '# Compatibility Diff Report', but it didn't")
	}
	if !strings.Contains(template, "**Errors:** 1") {
		t.Errorf("Expected template to contain '**Errors:** 1', but it didn't")
	}
	if !strings.Contains(template, "**Warnings:** 1") {
		t.Errorf("Expected template to contain '**Warnings:** 1', but it didn't")
	}
	if !strings.Contains(template, "**Info:** 1") {
		t.Errorf("Expected template to contain '**Info:** 1', but it didn't")
	}
	if !strings.Contains(template, "cti.vendor.package.new.entity.1.v1.0") {
		t.Errorf("Expected template to contain 'cti.vendor.package.new.entity.1.v1.0', but it didn't")
	}
	if !strings.Contains(template, "cti.vendor.package.removed.entity.1.v1.0") {
		t.Errorf("Expected template to contain 'cti.vendor.package.removed.entity.1.v1.0', but it didn't")
	}
	if !strings.Contains(template, "cti.vendor.package.modified.entity.1.v1.0") {
		t.Errorf("Expected template to contain 'cti.vendor.package.modified.entity.1.v1.0', but it didn't")
	}
	if !strings.Contains(template, "Changed property X") {
		t.Errorf("Expected template to contain 'Changed property X', but it didn't")
	}
	if !strings.Contains(template, "Error message 1") {
		t.Errorf("Expected template to contain 'Error message 1', but it didn't")
	}
}

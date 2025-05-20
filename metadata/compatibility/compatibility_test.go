package compatibility

import (
	"path/filepath"
	"testing"

	"github.com/acronis/go-cti/metadata/ctipackage"
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
	if err := oldPkg.Read(); err != nil {
		t.Fatalf("Failed to read old package: %v", err)
	}
	if err := oldPkg.Parse(); err != nil {
		t.Fatalf("Failed to parse old package: %v", err)
	}

	// Create and parse the new package
	newPkg, err := ctipackage.New(newPackagePath)
	if err != nil {
		t.Fatalf("Failed to create new package: %v", err)
	}
	if err := newPkg.Read(); err != nil {
		t.Fatalf("Failed to read new package: %v", err)
	}
	if err := newPkg.Parse(); err != nil {
		t.Fatalf("Failed to parse new package: %v", err)
	}

	// Create a compatibility checker
	checker := &CompatibilityChecker{}

	// Check compatibility between the packages
	ok := checker.CheckPackagesCompatibility(oldPkg, newPkg)

	// We expect an error because the type of 'val' property changed from string to integer
	if ok {
		t.Errorf("Expected compatibility check to fail, but it succeeded")
	} else {
		t.Logf("Compatibility check failed as expected: %v", checker.Messages)
	}

	// Test with nil packages
	if ok = checker.CheckPackagesCompatibility(nil, newPkg); ok {
		t.Errorf("Expected error when old package is nil, but got nil")
	}

	if ok = checker.CheckPackagesCompatibility(oldPkg, nil); ok {
		t.Errorf("Expected error when new package is nil, but got nil")
	}

	// Test with unparsed packages
	unparsedPkg, err := ctipackage.New(oldPackagePath)
	if err != nil {
		t.Fatalf("Failed to create unparsed package: %v", err)
	}
	if err := unparsedPkg.Read(); err != nil {
		t.Fatalf("Failed to read unparsed package: %v", err)
	}

	if ok = checker.CheckPackagesCompatibility(unparsedPkg, newPkg); ok {
		t.Errorf("Expected error when old package is not parsed, but got nil")
	}

	if ok = checker.CheckPackagesCompatibility(oldPkg, unparsedPkg); ok {
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
	ok := checker.CheckPackagesCompatibility(oldPkg1, oldPkg2)

	// We expect no error because the packages are identical
	if !ok {
		t.Errorf("Expected compatibility check to succeed, but it failed: %v", err)
	} else {
		t.Logf("Compatibility check succeeded as expected")
	}
}

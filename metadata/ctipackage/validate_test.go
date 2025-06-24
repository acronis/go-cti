package ctipackage

import (
	"testing"

	"github.com/acronis/go-cti/metadata/testsupp"
)

func TestValidateManual(t *testing.T) {
	testsupp.ManualTest(t)

	packagePath := ``

	// Create and parse the package
	pkg, err := New(packagePath)
	if err != nil {
		t.Fatalf("Failed to create package: %v", err)
	}
	if err = pkg.Read(); err != nil {
		t.Fatalf("Failed to read package: %v", err)
	}
	if err = pkg.Validate(); err != nil {
		t.Fatalf("Failed to validate package: %v", err)
	}
}

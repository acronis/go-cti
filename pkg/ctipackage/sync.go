package ctipackage

import (
	"fmt"
	"path/filepath"
)

func (pkg *Package) Sync() error {
	if err := extractRAMLxSpec(filepath.Join(pkg.BaseDir, RamlxDirName)); err != nil {
		return fmt.Errorf("extract raml files: %w", err)
	}

	return nil
}

package ctipackage

import (
	"fmt"
	"path/filepath"
)

func (b *Package) Sync() error {
	if err := extractRAMLxSpec(filepath.Join(b.BaseDir, RamlxDirName)); err != nil {
		return fmt.Errorf("extract raml files: %w", err)
	}

	return nil
}

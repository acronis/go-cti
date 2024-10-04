package bundle

import (
	"fmt"
	"path/filepath"
)

func (b *Bundle) Sync() error {
	if err := extractRAMLxSpec(filepath.Join(b.BaseDir, RamlxDirName)); err != nil {
		return fmt.Errorf("extract raml files: %w", err)
	}

	return nil
}

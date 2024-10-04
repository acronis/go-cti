package ctipackage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/filesys"
	"github.com/acronis/go-cti/pkg/ramlx"
)

const (
	defaultRamlxVersion = "1"
	RamlxDirName        = ".ramlx"
)

// extractRAMLxSpec extracts the embedded RAML files to the destination directory.
func extractRAMLxSpec(dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("remove destination directory: %w", err)
	}
	err := filesys.CopyFS(ramlx.RamlFiles, dst,
		filesys.WithRoot("spec_v"+defaultRamlxVersion),
		filesys.WithOverwrite(true))

	if err != nil {
		return fmt.Errorf("copy RAMLx specification: %w", err)
	}
	return nil
}

func (b *Package) ValidateRamlxSpec() error {

	return nil
}

func (b *Package) Initialize() error {
	if err := extractRAMLxSpec(filepath.Join(b.BaseDir, RamlxDirName)); err != nil {
		return fmt.Errorf("extract raml files: %w", err)
	}

	if err := b.SaveIndex(); err != nil {
		return fmt.Errorf("save index: %w", err)
	}
	if err := b.SaveIndexLock(); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}

	return nil
}

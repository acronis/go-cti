package ctipackage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/metadata/ramlx"

	"github.com/acronis/go-cti/metadata/filesys"
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

	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	err := filesys.CopyFS(ramlx.RamlFiles, dst,
		filesys.WithRoot("spec_v"+defaultRamlxVersion),
	)

	if err != nil {
		return fmt.Errorf("copy RAMLx specification: %w", err)
	}
	return nil
}

func (pkg *Package) ValidateRamlxSpec() error {
	return nil
}

func (pkg *Package) Initialize() error {
	if err := extractRAMLxSpec(filepath.Join(pkg.BaseDir, RamlxDirName)); err != nil {
		return fmt.Errorf("extract raml files: %w", err)
	}

	if err := pkg.SaveIndex(); err != nil {
		return fmt.Errorf("save index: %w", err)
	}
	// Save empty lock file
	if err := pkg.SaveIndexLock(NewIndexLock()); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}

	return nil
}

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
	if err := pkg.SaveIndexLock(); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}

	return nil
}

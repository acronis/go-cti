package ctipackage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/collector"
	"github.com/acronis/go-raml"
)

const (
	MetadataCacheFile = ".cache.json"
)

func (b *Package) Parse() error {
	baseDir := b.BaseDir

	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}

	r, err := raml.ParseFromString(b.Index.GenerateIndexRaml(false), "index.raml", absPath, raml.OptWithValidate())
	if err != nil {
		return fmt.Errorf("parse index.raml: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return fmt.Errorf("collect from package: %w", err)
	}

	b.Registry = c.Registry

	return nil
}

func (b *Package) DumpCache() error {
	bytes, err := json.Marshal(b.Registry.Total)
	if err != nil {
		return fmt.Errorf("serialize entities: %w", err)
	}
	return os.WriteFile(filepath.Join(b.BaseDir, MetadataCacheFile), bytes, 0600)
}

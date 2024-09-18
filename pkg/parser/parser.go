package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-raml"

	"github.com/acronis/go-cti/pkg/collector"
	_package "github.com/acronis/go-cti/pkg/package"

	"github.com/acronis/go-cti/pkg/validator"
)

const (
	MetadataCacheFile = ".cache.json"
)

// TODO: Maybe need to initialize one package parser instance and reuse it for all the parsing
// This could possibly simplify caching strategy for external clients
type Parser struct {
	BaseDir string

	Registry *collector.CtiRegistry

	RAML *raml.RAML
}

func ParsePackage(path string) (*Parser, error) {
	pkg, err := _package.New(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}

	baseDir := pkg.BaseDir

	r, err := raml.ParseFromString(pkg.Index.GenerateIndexRaml(false), "index.raml", baseDir, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("failed to parse index raml: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("failed to collect from package: %w", err)
	}

	return &Parser{
		BaseDir: baseDir,

		Registry: c.Registry,

		RAML: r,
	}, nil
}

// Parse parses a single entity file
// Parser will take a path for example "/home/app-package/test.raml".
func Parse(path string) (*Parser, error) {
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}
	baseDir := filepath.Dir(path)

	r, err := raml.ParseFromPath(path, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("failed to parse entity file: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("failed to collect from raml file: %w", err)
	}

	return &Parser{
		BaseDir: baseDir,

		Registry: c.Registry,

		RAML: r,
	}, nil
}

func ParseString(content string, fileName string, baseDir string) (*Parser, error) {
	r, err := raml.ParseFromString(content, fileName, baseDir, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("failed to parse entity file: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("failed to collect from raml string: %w", err)
	}

	return &Parser{
		BaseDir: baseDir,

		Registry: c.Registry,

		RAML: r,
	}, nil
}

func BuildPackageCache(path string) error {
	p, err := ParsePackage(path)
	if err != nil {
		return fmt.Errorf("failed to parse package: %w", err)
	}
	if err := p.DumpCache(); err != nil {
		return fmt.Errorf("failed to dump cache: %w", err)
	}
	return nil
}

func (p *Parser) Validate() []error {
	validator := validator.MakeCtiValidator()
	validator.LoadFromRegistry(p.Registry)
	return validator.ValidateAll()
}

func (p *Parser) DumpCache() error {
	bytes, err := json.Marshal(p.Registry.Total)
	if err != nil {
		return fmt.Errorf("failed to serialize entities: %w", err)
	}
	return os.WriteFile(filepath.Join(p.BaseDir, MetadataCacheFile), bytes, 0o644)
}

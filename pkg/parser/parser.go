package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-raml"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/collector"
)

const (
	MetadataCacheFile = ".cache.json"
)

// TODO: Maybe need to initialize one package parser instance and reuse it for all the parsing
// This could possibly simplify caching strategy for external clients
type Parser interface {
	DumpCache() error
	GetRegistry() *collector.CtiRegistry
}

type parserImpl struct {
	BaseDir string

	Registry *collector.CtiRegistry

	RAML *raml.RAML
}

func ParsePackage(path string) (Parser, error) {
	b, err := bundle.New(path)
	if err != nil {
		return nil, fmt.Errorf("create bundle: %w", err)
	}

	baseDir := b.BaseDir

	r, err := raml.ParseFromString(b.Index.GenerateIndexRaml(false), "index.raml", baseDir, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("parse index.raml: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("collect from bundle: %w", err)
	}

	return &parserImpl{
		BaseDir:  baseDir,
		Registry: c.Registry,
		RAML:     r,
	}, nil
}

// Parse parses a single entity file
// Parser will take a path for example "/home/app-package/test.raml".
func Parse(fPath string) (Parser, error) {
	if !filepath.IsAbs(fPath) {
		fPath, _ = filepath.Abs(fPath)
	}
	baseDir := filepath.Dir(fPath)

	r, err := raml.ParseFromPath(fPath, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("parse entity file: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("collect from raml file: %w", err)
	}

	return &parserImpl{
		BaseDir:  baseDir,
		Registry: c.Registry,
		RAML:     r,
	}, nil
}

func ParseString(content string, fileName string, baseDir string) (Parser, error) {
	r, err := raml.ParseFromString(content, fileName, baseDir, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("parse entity file: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("collect from raml string: %w", err)
	}

	return &parserImpl{
		BaseDir:  baseDir,
		Registry: c.Registry,
		RAML:     r,
	}, nil
}

func BuildPackageCache(path string) error {
	p, err := ParsePackage(path)
	if err != nil {
		return fmt.Errorf("parse package: %w", err)
	}
	if err := p.DumpCache(); err != nil {
		return fmt.Errorf("dump cache: %w", err)
	}
	return nil
}

func (p *parserImpl) DumpCache() error {
	bytes, err := json.Marshal(p.Registry.Total)
	if err != nil {
		return fmt.Errorf("serialize entities: %w", err)
	}
	return os.WriteFile(filepath.Join(p.BaseDir, MetadataCacheFile), bytes, 0600)
}

func (p *parserImpl) GetRegistry() *collector.CtiRegistry {
	return p.Registry
}

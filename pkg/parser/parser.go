package parser

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/collector"
	"github.com/acronis/go-cti/pkg/cti"
	"github.com/acronis/go-cti/pkg/index"
	"github.com/acronis/go-raml"
)

const (
	MetadataCacheFile = ".cache.json"
)

// TODO: Maybe need to initialize one package parser instance and reuse it for all the parsing
// This could possibly simplify caching strategy for external clients
type PackageParser struct {
	BaseDir string

	Index    *index.Index
	Registry *collector.CtiRegistry

	RAML *raml.RAML
}

// ParseAll cti type, instances, dictionaries in one single output
// PackageParser will take a path for example "/home/app-package/index.json".
func ParsePackage(path string) (*PackageParser, error) {
	idx, err := index.ReadIndexFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	baseDir := idx.BaseDir

	r, err := raml.ParseFromString(idx.GenerateIndexRaml(false), "index.raml", baseDir, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("failed to parse index raml: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("failed to collect from index: %w", err)
	}

	return &PackageParser{
		BaseDir: baseDir,

		Index:    idx,
		Registry: c.Registry,

		RAML: r,
	}, nil
}

// Parse parses a single entity file
// PackageParser will take a path for example "/home/app-package/test.raml".
func ParseEntity(path string) (*PackageParser, error) {
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}
	baseDir := filepath.Dir(path)

	r, err := raml.ParseFromPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse entity file: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("failed to collect from index: %w", err)
	}

	return &PackageParser{
		BaseDir: baseDir,

		Index:    nil,
		Registry: c.Registry,

		RAML: r,
	}, nil
}

func ParseEntityString(content string, fileName string, baseDir string) (*PackageParser, error) {
	r, err := raml.ParseFromString(content, fileName, baseDir, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("failed to parse entity file: %w", err)
	}

	c := collector.New(r, baseDir)
	if err := c.Collect(); err != nil {
		return nil, fmt.Errorf("failed to collect from index: %w", err)
	}

	return &PackageParser{
		BaseDir: baseDir,

		Index:    nil,
		Registry: c.Registry,

		RAML: r,
	}, nil
}

func (p *PackageParser) Serialize() error {
	if p.Index == nil {
		return fmt.Errorf("index is not set")
	}
	bytes, err := json.Marshal(p.Registry.Total)
	if err != nil {
		return fmt.Errorf("failed to serialize package: %w", err)
	}
	p.Index.PutSerialized(MetadataCacheFile)
	p.Index.Save()
	return os.WriteFile(filepath.Join(p.BaseDir, MetadataCacheFile), bytes, 0o644)
}

func (p *PackageParser) Bundle() error {
	if err := p.Serialize(); err != nil {
		return err
	}
	archive, err := os.Create(filepath.Join(p.BaseDir, "bundle.zip"))
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	for _, entity := range p.Registry.Instances {
		typ, ok := p.Registry.Types[cti.GetParentCti(entity.Cti)]
		if !ok {
			return fmt.Errorf("type %s not found", entity.Cti)
		}
		// TODO: Collect annotations from entire chain of CTI types
		for key, annotation := range typ.Annotations {
			if annotation.Asset == nil {
				continue
			}
			value := key.GetValue(entity.Values)
			assetPath := value.String()
			if assetPath == "" {
				break
			}
			err := func() error {
				asset, err := os.OpenFile(filepath.Join(p.BaseDir, assetPath), os.O_RDONLY, 0o644)
				if err != nil {
					return fmt.Errorf("failed to open asset %s: %w", assetPath, err)
				}
				defer asset.Close()

				w, err := zipWriter.Create(assetPath)
				if err != nil {
					return fmt.Errorf("failed to create asset %s in bundle: %w", assetPath, err)
				}
				if _, err = io.Copy(w, asset); err != nil {
					return fmt.Errorf("failed to write asset %s to bundle: %w", assetPath, err)
				}
				return nil
			}()
			if err != nil {
				return fmt.Errorf("failed to bundle asset %s: %w", assetPath, err)
			}
		}
	}

	w, err := zipWriter.Create("index.json")
	if err != nil {
		return fmt.Errorf("failed to create index in bundle: %w", err)
	}
	if _, err = w.Write(p.Index.ToBytes()); err != nil {
		return fmt.Errorf("failed to write index to bundle: %w", err)
	}

	for _, metadata := range p.Index.Serialized {
		f, err := os.OpenFile(filepath.Join(p.BaseDir, metadata), os.O_RDONLY, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open serialized metadata %s: %w", metadata, err)
		}
		defer f.Close()

		w, err := zipWriter.Create(metadata)
		if err != nil {
			return fmt.Errorf("failed to create serialized metadata %s in bundle: %w", metadata, err)
		}
		if _, err = io.Copy(w, f); err != nil {
			return fmt.Errorf("failed to write serialized metadata %s to bundle: %w", metadata, err)
		}
	}

	return nil
}

package collector

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/registry"
	"gopkg.in/yaml.v3"
)

type CTIMetadataCollector struct {
	collector.BaseCollector

	// entryPoint is a map of YAML fragments, where each key is a fragment name
	// and value is JSON Schema which is expressed as raw bytes.
	Entry map[string][]byte
}

func NewCTIMetadataCollector(entry map[string][]byte, baseDir string) *CTIMetadataCollector {
	return &CTIMetadataCollector{
		BaseCollector: collector.BaseCollector{
			CTIParser: cti.NewParser(),
			Registry:  registry.New(),
			BaseDir:   baseDir,
		},
		Entry: entry,
	}
}

func (c *CTIMetadataCollector) Collect() (*registry.MetadataRegistry, error) {
	if c.Entry == nil {
		return nil, errors.New("entry point is not set")
	}

	for fragmentName, raw := range c.Entry {
		r := bytes.NewReader(raw)
		head, err := c.readHead(r)
		if err != nil {
			return nil, fmt.Errorf("read head from fragment %s: %w", fragmentName, err)
		}
		var entity metadata.Entity
		switch head {
		case "#%CTI Type 1.0":
			var typ metadata.EntityType
			if err = yaml.Unmarshal(raw, &typ); err != nil {
				return nil, fmt.Errorf("unmarshal type %s: %w", fragmentName, err)
			}
			typ.SetSourceMap(&metadata.TypeSourceMap{
				DocumentSourceMap: metadata.DocumentSourceMap{
					OriginalPath: filepath.ToSlash(fragmentName),
					SourcePath:   filepath.ToSlash(fragmentName),
				},
			})
			entity = &typ
		case "#%CTI Instance 1.0":
			var instance metadata.EntityInstance
			if err = yaml.Unmarshal(raw, &instance); err != nil {
				return nil, fmt.Errorf("unmarshal instance %s: %w", fragmentName, err)
			}
			instance.SetSourceMap(&metadata.InstanceSourceMap{
				AnnotationType: metadata.AnnotationType{
					Name: instance.CTI,
				},
				DocumentSourceMap: metadata.DocumentSourceMap{
					OriginalPath: filepath.ToSlash(fragmentName),
					SourcePath:   filepath.ToSlash(fragmentName),
				},
			})
			entity = &instance
		default:
			return nil, fmt.Errorf("unknown fragment kind: head: %s", head)
		}
		err = c.Registry.Add(entity)
		if err != nil {
			return nil, fmt.Errorf("add cti entity: %w", err)
		}
	}

	return c.Registry, nil
}

// ReadHead reads, reset file and returns the trimmed first line of a file.
func (c *CTIMetadataCollector) readHead(f io.ReadSeeker) (string, error) {
	r := bufio.NewReader(f)
	head, err := r.ReadBytes('\n')
	if err != nil {
		return "", fmt.Errorf("read fragment head: %w", err)
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("seek to start: %w", err)
	}

	head = bytes.TrimRight(head, "\r\n ")
	return string(head), nil
}

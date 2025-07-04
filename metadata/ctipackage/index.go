package ctipackage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/acronis/go-cti/metadata/filesys"
)

const (
	IndexFileName = "index.json"
	RAMLExt       = ".raml"
	YAMLExt       = ".yaml"
)

type Index struct {
	PackageID            string            `json:"package_id"`
	RamlxVersion         string            `json:"ramlx_version,omitempty"`
	Apis                 []string          `json:"apis,omitempty"`
	Entities             []string          `json:"entities,omitempty"`
	Assets               []string          `json:"assets,omitempty"`
	Dictionaries         []string          `json:"dictionaries,omitempty"`
	Depends              map[string]string `json:"depends,omitempty"`
	Examples             []string          `json:"examples,omitempty"`
	AdditionalProperties interface{}       `json:"additional_properties,omitempty"`
	Serialized           []string          `json:"serialized,omitempty"`
}

func ReadIndex(dirPath string) (*Index, error) {
	return ReadIndexFile(path.Join(dirPath, IndexFileName))
}

func ReadIndexFile(fPath string) (*Index, error) {
	file, err := os.Open(fPath)
	if err != nil {
		return nil, fmt.Errorf("open index file: %w", err)
	}
	defer file.Close()

	idx, err := DecodeIndex(file)
	if err != nil {
		return nil, fmt.Errorf("decode index file: %w", err)
	}
	if err := idx.Check(); err != nil {
		return nil, fmt.Errorf("check index file: %w", err)
	}

	return idx, nil
}

func DecodeIndex(input io.Reader) (*Index, error) {
	var idx *Index
	decoder := json.NewDecoder(input)
	if err := decoder.Decode(&idx); err != nil {
		return nil, fmt.Errorf("error decoding index file: %w", err)
	}

	return idx, nil
}

func (idx *Index) Check() error {
	if idx.PackageID == "" {
		return errors.New("package id is missing")
	}
	for i, p := range idx.Apis {
		if p == "" {
			return fmt.Errorf("$.apis[%d]: api path cannot be empty", i)
		}
		if ext := filepath.Ext(p); ext != RAMLExt {
			return fmt.Errorf("$.apis[%d]: invalid api path extension: %s", i, ext)
		}
	}
	for i, p := range idx.Entities {
		if p == "" {
			return fmt.Errorf("$.entities[%d]: entity path cannot be empty", i)
		}
		if ext := filepath.Ext(p); ext != RAMLExt && ext != YAMLExt {
			return fmt.Errorf("$.entities[%d]: invalid entity extension: %s", i, ext)
		}
	}
	for i, p := range idx.Examples {
		if p == "" {
			return fmt.Errorf("$.examples[%d]: example path cannot be empty", i)
		}
		if ext := filepath.Ext(p); ext != RAMLExt && ext != YAMLExt {
			return fmt.Errorf("$.examples[%d]: invalid example extension: %s", i, ext)
		}
	}
	return nil
}

func (idx *Index) Clone() *Index {
	c := *idx
	return &c
}

func (idx *Index) ToBytes() []byte {
	bytes, _ := json.Marshal(idx)
	return bytes
}

func (idx *Index) Save(baseDir string) error {
	return filesys.WriteJSON(filepath.Join(baseDir, IndexFileName), idx)
}

func (idx *Index) PutSerialized(fName string) {
	for _, f := range idx.Serialized {
		if f == fName {
			return
		}
	}
	idx.Serialized = append(idx.Serialized, fName)
}

func (idx *Index) GetEntities() ([]Entity, error) {
	var entities []Entity
	for _, entity := range idx.Entities {
		name := filesys.GetBaseName(entity)
		entities = append(entities, Entity{
			Name: name,
			Path: entity,
		})
	}

	return entities, nil
}

func (idx *Index) GetAssets() []string {
	return idx.Assets
}

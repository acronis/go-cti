package ctipackage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/metadata/filesys"
	"github.com/blang/semver/v4"
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
	Depends              map[string]string `json:"depends,omitempty"`
	Examples             []string          `json:"examples,omitempty"`
	AdditionalProperties interface{}       `json:"additional_properties,omitempty"`
	Serialized           []string          `json:"serialized,omitempty"`

	Vendor string `json:"-"`
	Pkg    string `json:"-"`
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
	packageIDChunks := strings.Split(idx.PackageID, ".")
	idx.Vendor = packageIDChunks[0]
	idx.Pkg = packageIDChunks[1]

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
	if err := ValidatePackageID(idx.PackageID); err != nil {
		return fmt.Errorf("validate package ID: %w", err)
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
	for name, version := range idx.Depends {
		if err := ValidateDependencyName(name); err != nil {
			return fmt.Errorf("$.depends[%s]: %w", name, err)
		}
		if _, err := semver.Parse(version); err != nil {
			return fmt.Errorf("$.depends[%s]: invalid version %s: %w", name, version, err)
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

func (idx *Index) Hash() string {
	h := sha256.New()
	h.Write(idx.ToBytes())

	return fmt.Sprintf("%x", h.Sum(nil))
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

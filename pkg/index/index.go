package index

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/pkg/filesys"
)

const (
	PackageType = "cti.a.p.app.package.v1.0"
)

type Index struct {
	Type                 string      `json:"type"`
	AppCode              string      `json:"app_code"`
	Apis                 []string    `json:"apis,omitempty"`
	Entities             []string    `json:"entities,omitempty"`
	Assets               []string    `json:"assets,omitempty"`
	Dictionaries         []string    `json:"dictionaries,omitempty"`
	Depends              []string    `json:"depends,omitempty"`
	Examples             []string    `json:"examples,omitempty"`
	AdditionalProperties interface{} `json:"additional_properties,omitempty"`
	Serialized           []string    `json:"serialized,omitempty"`

	BaseDir string `json:"-"`
}

func ReadIndexFile(path string) (*Index, error) {
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening index file: %w", err)
	}
	defer file.Close()

	idx, err := DecodeIndexFile(file)
	if err != nil {
		return nil, fmt.Errorf("error decoding index file: %w", err)
	}

	idx.BaseDir = filepath.Dir(path)
	if err := idx.Check(); err != nil {
		return nil, fmt.Errorf("error checking index file: %w", err)
	}

	return idx, nil
}

func UnmarshalIndexFIle(v []byte) (*Index, error) {
	var idx *Index
	if err := json.Unmarshal(v, &idx); err != nil {
		return nil, fmt.Errorf("error unmarshalling index file: %w", err)
	}

	return idx, nil
}

func DecodeIndexFile(file io.Reader) (*Index, error) {
	var idx *Index
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&idx); err != nil {
		return nil, fmt.Errorf("error decoding index file: %w", err)
	}

	return idx, nil
}

func (idx *Index) Check() error {
	if idx.Type != PackageType {
		return fmt.Errorf("invalid index file type: %s", idx.Type)
	}
	for i, path := range idx.Apis {
		if path == "" {
			return fmt.Errorf("$.apis[%d]: api path cannot be empty", i)
		}
		ext := filepath.Ext(path)
		if ext != ".raml" {
			return fmt.Errorf("$.apis[%d]: invalid api path extension: %s", i, ext)
		}
	}
	for i, path := range idx.Entities {
		if path == "" {
			return fmt.Errorf("$.entities[%d]: entity path cannot be empty", i)
		}
		ext := filepath.Ext(path)
		if ext != ".raml" {
			return fmt.Errorf("$.entities[%d]: invalid entity extension: %s", i, ext)
		}
	}
	for i, path := range idx.Examples {
		if path == "" {
			return fmt.Errorf("$.examples[%d]: example path cannot be empty", i)
		}
		ext := filepath.Ext(path)
		if ext != ".raml" {
			return fmt.Errorf("$.examples[%d]: invalid example extension: %s", i, ext)
		}
	}
	if idx.AppCode == "" {
		return fmt.Errorf("missing app code")
	}
	return nil
}

func (idx *Index) GenerateIndexRaml(includeExamples bool) string {
	// TODO: Maybe it is possible to avoid index.raml generation and reuse RAML parser instance to parse each entity file instead.
	// Could have something like PackageParser.Initialize(path string) (maybe even in go-raml itself).
	// This would also allow employing per-fragment cache strategy based on project configuration.
	var sb strings.Builder
	sb.WriteString("#%RAML 1.0 Library\nuses:")
	for i, entity := range idx.Entities {
		sb.WriteString(fmt.Sprintf("\n  e%d: %s", i+1, entity))
	}
	if includeExamples {
		for i, example := range idx.Examples {
			sb.WriteString(fmt.Sprintf("\n  x%d: %s", i+1, example))
		}
	}
	return sb.String()
}

func (idx *Index) ToBytes() []byte {
	bytes, _ := json.Marshal(idx)
	return bytes
}

func (idx *Index) Save() error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(idx.BaseDir, "index.json"), data, 0755)
}

func (idx *Index) PutSerialized(file string) {
	for _, f := range idx.Serialized {
		if f == file {
			return
		}
	}
	idx.Serialized = append(idx.Serialized, file)
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

func (idx *Index) GetDictionaries() (Dictionaries, error) {
	dictionaries := Dictionaries{
		Dictionaries: make(map[LangCode]Entry),
	}

	for _, dict := range idx.Dictionaries {
		file, err := os.Open(path.Join(idx.BaseDir, dict))
		if err != nil {
			return Dictionaries{}, err
		}
		defer file.Close()

		entry, err := ValidateDictionary(file)
		if err != nil {
			return Dictionaries{}, err
		}
		lang := filesys.GetBaseName(file.Name())
		dictionaries.Dictionaries[LangCode(lang)] = entry
	}

	return dictionaries, nil
}

func ValidateDictionary(file *os.File) (Entry, error) {
	var config Entry
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error decoding dictionary file: %w", err)
	}

	return config, nil
}

package bundle

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
	IndexFileName = "index.json"
	RAMLExt       = ".raml"
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

	BaseDir  string `json:"-"`
	FilePath string `json:"-"`
}

func ReadIndexFile(idxPath string) (*Index, error) {
	if !filepath.IsAbs(idxPath) {
		fixedPath, err := filepath.Abs(idxPath)
		if err != nil {
			return nil, fmt.Errorf("get absolute path: %w", err)
		}
		idxPath = fixedPath
	}

	file, err := os.Open(idxPath)
	if err != nil {
		return nil, fmt.Errorf("open index file: %w", err)
	}
	defer file.Close()

	idx, err := DecodeIndexFile(file)
	if err != nil {
		return nil, fmt.Errorf("decode index file: %w", err)
	}

	idx.BaseDir = filepath.Dir(idxPath)
	idx.FilePath = idxPath
	if err := idx.Check(); err != nil {
		return nil, fmt.Errorf("check index file: %w", err)
	}

	return idx, nil
}

func UnmarshalIndexFile(v []byte) (*Index, error) {
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
	for i, p := range idx.Apis {
		if p == "" {
			return fmt.Errorf("$.apis[%d]: api path cannot be empty", i)
		}
		ext := filepath.Ext(p)
		if ext != RAMLExt {
			return fmt.Errorf("$.apis[%d]: invalid api path extension: %s", i, ext)
		}
	}
	for i, p := range idx.Entities {
		if p == "" {
			return fmt.Errorf("$.entities[%d]: entity path cannot be empty", i)
		}
		ext := filepath.Ext(p)
		if ext != RAMLExt {
			return fmt.Errorf("$.entities[%d]: invalid entity extension: %s", i, ext)
		}
	}
	for i, p := range idx.Examples {
		if p == "" {
			return fmt.Errorf("$.examples[%d]: example path cannot be empty", i)
		}
		ext := filepath.Ext(p)
		if ext != RAMLExt {
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

func (idx *Index) Clone() *Index {
	c := *idx
	return &c
}

func (idx *Index) ToBytes() []byte {
	bytes, _ := json.Marshal(idx)
	return bytes
}

func (idx *Index) Save() error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling index file: %w", err)
	}
	return os.WriteFile(idx.FilePath, data, 0600)
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

func (idx *Index) GetDictionaries() (Dictionaries, error) {
	dictionaries := Dictionaries{
		Dictionaries: make(map[LangCode]Entry),
	}

	for _, dict := range idx.Dictionaries {
		file, err := os.Open(path.Join(idx.BaseDir, dict))
		if err != nil {
			return Dictionaries{}, fmt.Errorf("open dictionary file: %w", err)
		}
		defer file.Close()

		entry, err := ValidateDictionary(file)
		if err != nil {
			return Dictionaries{}, fmt.Errorf("validate dictionary: %w", err)
		}
		lang := filesys.GetBaseName(file.Name())
		dictionaries.Dictionaries[LangCode(lang)] = entry
	}

	return dictionaries, nil
}

func ValidateDictionary(file *os.File) (Entry, error) {
	decoder := json.NewDecoder(file)

	var config Entry
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode dictionary file: %w", err)
	}

	return config, nil
}

package ctipackage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/filesys"
)

const (
	DependencyDirName = ".dep"
)

type Package struct {
	Index     *Index
	IndexLock *IndexLock

	Registry *collector.MetadataRegistry

	BaseDir string
}

// New creates a new package from the specified path.
// If the path is empty, the current working directory is used.
func New(baseDir string, options ...InitializeOption) (*Package, error) {
	b := &Package{
		BaseDir: filepath.ToSlash(path.Clean(baseDir)),
		Index:   &Index{},
		IndexLock: &IndexLock{
			Version:           IndexLockVersion,
			DependentPackages: make(map[string]string),
			SourceInfo:        make(map[string]Info),
		},
	}

	for _, opt := range options {
		if err := opt(b); err != nil {
			return nil, err
		}
	}

	return b, nil
}

type InitializeOption func(*Package) error

func WithID(id string) InitializeOption {
	return func(pkg *Package) error {
		if ValidateID(id) != nil {
			return fmt.Errorf("validate id: %w", ValidateID(id))
		}
		pkg.Index.PackageID = id
		return nil
	}
}

func WithRamlxVersion(version string) InitializeOption {
	return func(pkg *Package) error {
		// TODO validate that version is a valid ramlx version and supported by the current version of tool
		pkg.Index.RamlxVersion = version
		return nil
	}
}
func WithEntities(entities []string) InitializeOption {
	return func(pkg *Package) error {
		if entities != nil {
			pkg.Index.Entities = entities
		}
		return nil
	}
}

func (pkg *Package) Read() error {
	idx, err := ReadIndex(pkg.BaseDir)
	if err != nil {
		return fmt.Errorf("read index file: %w", err)
	}
	idxLock, err := ReadIndexLock(pkg.BaseDir)
	if err != nil {
		return fmt.Errorf("read index lock: %w", err)
	}

	pkg.Index = idx
	pkg.IndexLock = idxLock
	return nil
}

func (pkg *Package) SaveIndexLock() error {
	if err := pkg.IndexLock.Save(pkg.BaseDir); err != nil {
		return fmt.Errorf("save index lock: %w", err)
	}
	return nil
}

func (pkg *Package) SaveIndex() error {
	if err := pkg.Index.Save(pkg.BaseDir); err != nil {
		return fmt.Errorf("save index: %w", err)
	}
	return nil
}

func (pkg *Package) GetDictionaries() (Dictionaries, error) {
	dictionaries := Dictionaries{
		Dictionaries: make(map[LangCode]Entry),
	}

	for _, dict := range pkg.Index.Dictionaries {
		file, err := os.Open(path.Join(pkg.BaseDir, dict))
		if err != nil {
			return Dictionaries{}, fmt.Errorf("open dictionary file: %w", err)
		}
		defer file.Close()

		entry, err := validateDictionary(file)
		if err != nil {
			return Dictionaries{}, fmt.Errorf("validate dictionary: %w", err)
		}
		lang := filesys.GetBaseName(file.Name())
		dictionaries.Dictionaries[LangCode(lang)] = entry
	}

	return dictionaries, nil
}

func validateDictionary(input io.Reader) (Entry, error) {
	decoder := json.NewDecoder(input)

	var config Entry
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode dictionary: %w", err)
	}

	return config, nil
}

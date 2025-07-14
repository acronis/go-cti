package ctipackage

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/acronis/go-cti/metadata/registry"
)

const (
	DependencyDirName = ".dep"
)

type Package struct {
	Index     *Index
	IndexLock *IndexLock

	LocalRegistry  *registry.MetadataRegistry
	GlobalRegistry *registry.MetadataRegistry

	Parsed bool

	BaseDir string
}

// New creates a new package from the specified path.
// If the path is empty, the current working directory is used.
func New(baseDir string, options ...InitializeOption) (*Package, error) {
	absPath, err := filepath.Abs(path.Clean(baseDir))
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}
	b := &Package{
		BaseDir: filepath.ToSlash(absPath),
		Index:   &Index{},
		IndexLock: &IndexLock{
			Version:     IndexLockVersion,
			Hash:        "",
			Depends:     make(map[string]string),
			DependsInfo: make(map[string]Info),
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
		if ValidatePackageID(id) != nil {
			return fmt.Errorf("validate id: %w", ValidatePackageID(id))
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
	if pkg.Index == nil {
		return fmt.Errorf("index is not initialized")
	}

	// make sure that index hash in lock file is up to date
	pkg.IndexLock.Hash = pkg.Index.Hash()

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

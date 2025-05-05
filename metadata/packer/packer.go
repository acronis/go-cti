package packer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/archiver"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/ctipackage"
)

const (
	ArchiveExtension = ".cti"
)

var (
	SkipFile   error = errors.New("skip this file")
	SkipDir    error = filepath.SkipDir
	SkipChecks error = errors.New("skip other checks")
)

type Packer struct {
	IncludeSources     bool
	Archiver           archiver.Archiver
	AnnotationHandlers []AnnotationHandler
	// ExcludeFunction is called for each file in the package.
	// If SkipFile is returned, the file will be excluded from the archive.
	// If SkipDir is returned, whole directory will be excluded from the archive.
	ExcludeFunction func(fsPath string, e os.DirEntry) error
	// WhitelistFunction is called for each file in the package.
	// If SkipChecks is returned, all other checks are skipped and the file will be added to the archive.
	WhitelistFunction func(fsPath string, e os.DirEntry) error
}

type Option func(*Packer) error

func WithSources() Option {
	return func(p *Packer) error {
		p.IncludeSources = true
		return nil
	}
}

func WithArchiver(w archiver.Archiver) Option {
	return func(p *Packer) error {
		p.Archiver = w
		return nil
	}
}

func WithAnnotationHandler(h AnnotationHandler) Option {
	return func(p *Packer) error {
		if p.Archiver == nil {
			return fmt.Errorf("writer is not set")
		}
		p.AnnotationHandlers = append(p.AnnotationHandlers, h)
		return nil
	}
}

func WithExcludeFunction(f func(fsPath string, e os.DirEntry) error) Option {
	return func(p *Packer) error {
		p.ExcludeFunction = f
		return nil
	}
}

func WithWhitelistFunction(f func(fsPath string, e os.DirEntry) error) Option {
	return func(p *Packer) error {
		p.WhitelistFunction = f
		return nil
	}
}

type AnnotationHandler func(baseDir string, writer archiver.Archiver,
	key metadata.GJsonPath, entity *metadata.Entity, a metadata.Annotations) error

func New(opts ...Option) (*Packer, error) {
	pkr := &Packer{}

	for _, opt := range opts {
		if err := opt(pkr); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	return pkr, nil
}

func (p *Packer) Pack(pkg *ctipackage.Package, destination string) error {
	if p.Archiver == nil {
		return fmt.Errorf("writer is not set")
	}

	if err := pkg.Read(); err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	if err := pkg.Parse(); err != nil {
		return fmt.Errorf("parse package: %w", err)
	}

	zipWriter, err := p.Archiver.Init(destination)
	if err != nil {
		return fmt.Errorf("create zip writer: %w", err)
	}
	defer zipWriter.Close()

	idx := pkg.Index.Clone()
	idx.PutSerialized(ctipackage.MetadataCacheFile)

	if err := p.Archiver.WriteBytes(ctipackage.IndexFileName, idx.ToBytes()); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	for _, metadata := range idx.Serialized {
		if err := p.Archiver.WriteFile(pkg.BaseDir, metadata); err != nil {
			return fmt.Errorf("write metadata %s: %w", metadata, err)
		}
	}

	if p.IncludeSources {
		if err := p.Archiver.WriteDirectory(pkg.BaseDir, func(fsPath string, e os.DirEntry) error {

			// Support custom whitelist function
			if p.WhitelistFunction != nil {
				if err := p.WhitelistFunction(fsPath, e); err != nil {
					if errors.Is(err, SkipChecks) {
						return nil
					}

					return err
				}
			}

			if e.IsDir() {
				switch e.Name() {
				case ctipackage.DependencyDirName:
					return archiver.SkipDir
				case ctipackage.RamlxDirName:
					return archiver.SkipDir
				}

				if strings.HasPrefix(e.Name(), ".") {
					return archiver.SkipDir
				}
			} else {
				if strings.HasPrefix(e.Name(), ".") {
					return archiver.SkipFile
				}

				// file already written
				if e.Name() == ctipackage.IndexFileName {
					return archiver.SkipFile
				}
			}

			// Support custom exclude function
			if p.ExcludeFunction != nil {
				switch err := p.ExcludeFunction(fsPath, e); {
				case errors.Is(err, SkipFile):
					return archiver.SkipFile
				case errors.Is(err, SkipDir):
					return archiver.SkipDir
				default:
					return err
				}
			}

			return nil
		}); err != nil {
			return fmt.Errorf("write sources: %w", err)
		}
	}

	r := pkg.GlobalRegistry
	for _, entity := range r.Instances {
		if err := p.WriteEntity(pkg.BaseDir, r, entity); err != nil {
			return fmt.Errorf("write entity: %w", err)
		}
	}

	return nil
}

func (p *Packer) WriteEntity(baseDir string, r *collector.MetadataRegistry, entity *metadata.Entity) error {
	tID := metadata.GetParentCti(entity.Cti)
	typ, ok := r.Types[tID]
	if !ok {
		return fmt.Errorf("parent type %s not found", tID)
	}
	// TODO: Collect annotations from the entire chain of CTI types
	for _, handler := range p.AnnotationHandlers {
		for key, annotation := range typ.Annotations {
			if err := handler(baseDir, p.Archiver, key, entity, annotation); err != nil {
				return fmt.Errorf("handle annotation: %w", err)
			}
		}
	}

	return nil
}

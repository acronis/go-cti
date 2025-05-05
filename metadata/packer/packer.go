package packer

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/archiver"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/ctipackage"
)

const (
	ArchiveExtension = ".cti"
)

type Packer struct {
	arch archiver.Archiver

	IncludeSources     bool
	AnnotationHandlers []AnnotationHandler
	// ExcludeFunction is called for each file in the package.
	// If SkipFile is returned, the file will be excluded from the archive.
	// If SkipDir is returned, whole directory will be excluded from the archive.
	ExcludeFunction func(fsPath string, e os.DirEntry) error
	//
	ExtraFiles []string
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
		p.arch = w
		return nil
	}
}

func WithAnnotationHandler(h AnnotationHandler) Option {
	return func(p *Packer) error {
		if p.arch == nil {
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

func WithExtraFiles(files ...string) Option {
	return func(p *Packer) error {
		p.ExtraFiles = append(p.ExtraFiles, files...)
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
	if p.arch == nil {
		return fmt.Errorf("writer is not set")
	}

	if err := pkg.Read(); err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	if err := pkg.Parse(); err != nil {
		return fmt.Errorf("parse package: %w", err)
	}

	zipWriter, err := p.arch.Init(destination)
	if err != nil {
		return fmt.Errorf("create zip writer: %w", err)
	}
	defer zipWriter.Close()

	idx := pkg.Index.Clone()
	idx.PutSerialized(ctipackage.MetadataCacheFile)

	if err := p.arch.WriteBytes(ctipackage.IndexFileName, idx.ToBytes()); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	// Direct write files
	files := slices.Concat(idx.Serialized, p.ExtraFiles)
	slices.Sort(files)
	files = slices.Compact(files)

	for _, f := range files {
		if err := p.arch.WriteFile(pkg.BaseDir, f); err != nil {
			return fmt.Errorf("write file %s: %w", f, err)
		}
	}

	if p.IncludeSources {
		if err := p.arch.WriteDirectory(pkg.BaseDir, func(fsPath string, e os.DirEntry) error {
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
				case errors.Is(err, archiver.SkipFile):
					return archiver.SkipFile
				case errors.Is(err, archiver.SkipDir):
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

	// Allow callback to write file dependent to annotations
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
			if err := handler(baseDir, p.arch, key, entity, annotation); err != nil {
				return fmt.Errorf("handle annotation: %w", err)
			}
		}
	}

	return nil
}

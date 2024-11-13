package packer

import (
	"fmt"
	"os"
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
	IncludeSources     bool
	Archiver           archiver.Archiver
	AnnotationHandlers []AnnotationHandler
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
			// , err := filepath.Rel(pkg.BaseDir, fsPath)
			// if err != nil {
			// 	return err
			// }

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

			return nil
		}); err != nil {
			return fmt.Errorf("write sources: %w", err)
		}
	}

	r, err := pkg.ParseWithCache()
	if err != nil {
		return fmt.Errorf("parse with cache: %w", err)
	}
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

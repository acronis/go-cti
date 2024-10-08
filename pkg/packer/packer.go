package packer

import (
	"fmt"
	"strings"

	"github.com/acronis/go-cti/pkg/collector"
	"github.com/acronis/go-cti/pkg/cti"
	"github.com/acronis/go-cti/pkg/ctipackage"
)

const (
	ArchiveExtension = ".cti"
)

type Packer struct {
	IncludeSources     bool
	Writer             Writer
	AnnotationHandlers []AnnotationHandler
}

type Option func(*Packer) error

func WithSources() Option {
	return func(p *Packer) error {
		p.IncludeSources = true
		return nil
	}
}

func WithWriter(w Writer) Option {
	return func(p *Packer) error {
		p.Writer = w
		return nil
	}
}

func WithAnnotationHandler(h AnnotationHandler) Option {
	return func(p *Packer) error {
		if p.Writer == nil {
			return fmt.Errorf("writer is not set")
		}
		p.AnnotationHandlers = append(p.AnnotationHandlers, h)
		return nil
	}
}

type AnnotationHandler func(baseDir string, writer Writer, key cti.GJsonPath, entity *cti.Entity, a cti.Annotations) error

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
	if p.Writer == nil {
		return fmt.Errorf("writer is not set")
	}

	if err := pkg.Read(); err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	zipWriter, err := p.Writer.Init(destination)
	if err != nil {
		return fmt.Errorf("create zip writer: %w", err)
	}
	defer zipWriter.Close()

	idx := pkg.Index.Clone()
	idx.PutSerialized(ctipackage.MetadataCacheFile)

	if err := p.Writer.WriteBytes(ctipackage.IndexFileName, idx.ToBytes()); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	for _, metadata := range idx.Serialized {
		if err := p.Writer.WriteFile(pkg.BaseDir, metadata); err != nil {
			return fmt.Errorf("write metadata %s: %w", metadata, err)
		}
	}

	if p.IncludeSources {
		if err := p.Writer.WriteDirectory(pkg.BaseDir, func(dirName string, fName string) bool {
			// exclude all hidden files
			if dirName != "." && strings.HasPrefix(dirName, ".") {
				return true
			}
			if strings.HasPrefix(fName, ".") {
				return true
			}

			if dirName == ctipackage.DependencyDirName {
				return true
			}
			if dirName == ctipackage.RamlxDirName {
				return true
			}
			// file already written
			if fName == ctipackage.IndexFileName {
				return true
			}
			return false
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

func (p *Packer) WriteEntity(baseDir string, r *collector.CtiRegistry, entity *cti.Entity) error {
	tID := cti.GetParentCti(entity.Cti)
	typ, ok := r.Types[tID]
	if !ok {
		return fmt.Errorf("parent type %s not found", tID)
	}
	// TODO: Collect annotations from the entire chain of CTI types
	for _, handler := range p.AnnotationHandlers {
		for key, annotation := range typ.Annotations {
			if err := handler(baseDir, p.Writer, key, entity, annotation); err != nil {
				return fmt.Errorf("handle annotation: %w", err)
			}
		}
	}

	return nil
}

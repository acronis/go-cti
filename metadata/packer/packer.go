package packer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/archiver"
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
	key metadata.GJsonPath, object *metadata.EntityInstance, a *metadata.Annotations) error

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
		// Collect relative paths of all parsed source files, excluding files
		// outside the package root (e.g. dependency files) and the root index
		// (already written above via WriteBytes).
		rootIndexPath := filepath.Join(pkg.BaseDir, ctipackage.IndexFileName)
		sourceFiles := make([]string, 0, len(pkg.SourceFiles))
		for _, absPath := range pkg.SourceFiles {
			relPath, err := filepath.Rel(pkg.BaseDir, absPath)
			if err != nil || strings.HasPrefix(relPath, "..") {
				continue
			}
			if absPath == rootIndexPath {
				continue
			}
			sourceFiles = append(sourceFiles, relPath)
		}
		slices.Sort(sourceFiles)
		sourceFiles = slices.Compact(sourceFiles)

		for _, relPath := range sourceFiles {
			absPath := filepath.Join(pkg.BaseDir, relPath)
			if p.ExcludeFunction != nil {
				info, err := os.Lstat(absPath)
				if err != nil {
					return fmt.Errorf("stat source file %s: %w", absPath, err)
				}
				switch err := p.ExcludeFunction(absPath, fs.FileInfoToDirEntry(info)); {
				case errors.Is(err, archiver.SkipFile):
					continue
				case err != nil:
					return fmt.Errorf("exclude function: %w", err)
				}
			}
			if err := p.arch.WriteFile(pkg.BaseDir, relPath); err != nil {
				return fmt.Errorf("write source file %s: %w", relPath, err)
			}
		}
	}

	// Allow callback to write file dependent to annotations
	r := pkg.GlobalRegistry
	for _, entity := range r.Instances {
		if err := p.WriteEntity(pkg.BaseDir, entity); err != nil {
			return fmt.Errorf("write entity: %w", err)
		}
	}

	return nil
}

func (p *Packer) WriteEntity(baseDir string, object *metadata.EntityInstance) error {
	typ := object.Parent()
	if typ == nil {
		return fmt.Errorf("%s has no parent type", object.CTI)
	}
	// TODO: Collect annotations from the entire chain of CTI types
	for _, handler := range p.AnnotationHandlers {
		for key, annotation := range typ.Annotations {
			if err := handler(baseDir, p.arch, key, object, annotation); err != nil {
				return fmt.Errorf("handle annotation: %w", err)
			}
		}
	}

	return nil
}

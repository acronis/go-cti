package bunman

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/cti"
	"github.com/acronis/go-cti/pkg/parser"
)

const (
	BundleName = "bundle.zip"
)

func Pack(bd *bundle.Bundle, includeSource bool) (string, error) {
	baseDir := bd.BaseDir

	r, err := ParseWithCache(bd)
	if err != nil {
		return "", fmt.Errorf("parse with cache: %w", err)
	}
	fileName := filepath.Join(baseDir, BundleName)
	archive, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	w, err := zipWriter.Create("index.json")
	if err != nil {
		return "", fmt.Errorf("create index in bundle: %w", err)
	}

	idx := bd.Index.Clone()
	idx.PutSerialized(parser.MetadataCacheFile)

	if _, err = w.Write(idx.ToBytes()); err != nil {
		return "", fmt.Errorf("write index to bundle: %w", err)
	}

	for _, metadata := range idx.Serialized {
		f, err := os.OpenFile(filepath.Join(baseDir, metadata), os.O_RDONLY, 0o644)
		if err != nil {
			return "", fmt.Errorf("open serialized metadata %s: %w", metadata, err)
		}
		defer f.Close()

		w, err := zipWriter.Create(metadata)
		if err != nil {
			return "", fmt.Errorf("create serialized metadata %s in bundle: %w", metadata, err)
		}
		if _, err = io.Copy(w, f); err != nil {
			return "", fmt.Errorf("write serialized metadata %s to bundle: %w", metadata, err)
		}
	}

	if includeSource {
		err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
			rel := strings.TrimPrefix(path, baseDir)
			if rel == "" || d.IsDir() {
				return nil
			}
			rel = filepath.ToSlash(rel[1:])
			if err != nil {
				return fmt.Errorf("walk directory: %w", err)
			}
			if rel[0] == '.' || rel == BundleName || rel == bundle.IndexFileName {
				return nil
			}
			f, err := os.OpenFile(path, os.O_RDONLY, 0o644)
			if err != nil {
				return fmt.Errorf("open index: %w", err)
			}
			w, err := zipWriter.Create(rel)
			if err != nil {
				return fmt.Errorf("create file in bundle: %w", err)
			}
			if _, err = io.Copy(w, f); err != nil {
				return fmt.Errorf("copy file in bundle: %w", err)
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("walk directory: %w", err)
		}
		return "", nil
	}

	for _, entity := range r.Instances {
		tID := cti.GetParentCti(entity.Cti)
		typ, ok := r.Types[tID]
		if !ok {
			return "", fmt.Errorf("type %s not found", tID)
		}
		// TODO: Collect annotations from the entire chain of CTI types
		for key, annotation := range typ.Annotations {
			if annotation.Asset == nil {
				continue
			}
			value := key.GetValue(entity.Values)
			assetPath := value.String()
			if assetPath == "" {
				break
			}
			err := func() error {
				asset, err := os.OpenFile(filepath.Join(baseDir, assetPath), os.O_RDONLY, 0o644)
				if err != nil {
					return fmt.Errorf("open asset %s: %w", assetPath, err)
				}
				defer asset.Close()

				w, err := zipWriter.Create(assetPath)
				if err != nil {
					return fmt.Errorf("create asset %s in bundle: %w", assetPath, err)
				}
				if _, err = io.Copy(w, asset); err != nil {
					return fmt.Errorf("write asset %s to bundle: %w", assetPath, err)
				}
				return nil
			}()
			if err != nil {
				return "", fmt.Errorf("bundle asset %s: %w", assetPath, err)
			}
		}
	}

	return fileName, nil
}

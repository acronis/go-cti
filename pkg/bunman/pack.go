package bunman

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/acronis/go-cti/pkg/collector"
	"github.com/acronis/go-cti/pkg/cti"
)

const (
	BundleName = "bundle.zip"
)

func writeMetadata(bd *bundle.Bundle, metadata string, zipWriter *zip.Writer) error {
	f, err := os.OpenFile(filepath.Join(bd.BaseDir, metadata), os.O_RDONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open serialized metadata %s: %w", metadata, err)
	}
	defer f.Close()

	w, err := zipWriter.Create(metadata)
	if err != nil {
		return fmt.Errorf("create serialized metadata %s in bundle: %w", metadata, err)
	}
	if _, err = io.Copy(w, f); err != nil {
		return fmt.Errorf("write serialized metadata %s to bundle: %w", metadata, err)
	}
	return nil
}

func writeIndex(bd *bundle.Bundle, zipWriter *zip.Writer) error {
	w, err := zipWriter.Create(bundle.IndexFileName)
	if err != nil {
		return fmt.Errorf("create index in bundle: %w", err)
	}

	idx := bd.Index.Clone()
	idx.PutSerialized(bundle.MetadataCacheFile)

	if _, err = w.Write(idx.ToBytes()); err != nil {
		return fmt.Errorf("write index to bundle: %w", err)
	}

	for _, metadata := range idx.Serialized {
		if err := writeMetadata(bd, metadata, zipWriter); err != nil {
			return fmt.Errorf("write metadata %s: %w", metadata, err)
		}
	}

	return nil
}

func writeSources(baseDir string, zipWriter *zip.Writer) error {
	if err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
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
	}); err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}
	return nil
}

func writeAsset(assetPath string, zipWriter *zip.Writer) error {
	asset, err := os.OpenFile(assetPath, os.O_RDONLY, 0o644)
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
}

func writeEntity(r *collector.CtiRegistry, entity *cti.Entity, baseDir string, zipWriter *zip.Writer) error {
	tID := cti.GetParentCti(entity.Cti)
	typ, ok := r.Types[tID]
	if !ok {
		return fmt.Errorf("parent type %s not found", tID)
	}
	// TODO: Collect annotations from the entire chain of CTI types
	for key, annotation := range typ.Annotations {
		if annotation.Asset == nil {
			continue
		}
		value := key.GetValue(entity.Values)
		assetPath := value.String()
		if assetPath == "" {
			slog.Warn("Empty asset path", slog.String("entity", entity.Cti), slog.String("key", value.Str))
			break
		}
		if err := writeAsset(filepath.Join(baseDir, assetPath), zipWriter); err != nil {
			return fmt.Errorf("write asset %s: %w", assetPath, err)
		}
	}

	return nil
}

func Pack(bd *bundle.Bundle, includeSource bool) (string, error) {
	fileName := filepath.Join(bd.BaseDir, BundleName)
	archive, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	// write index
	if err := writeIndex(bd, zipWriter); err != nil {
		return "", fmt.Errorf("write index: %w", err)
	}

	if includeSource {
		if err := writeSources(bd.BaseDir, zipWriter); err != nil {
			return "", fmt.Errorf("write sources: %w", err)
		}
	}

	r, err := ParseWithCache(bd)
	if err != nil {
		return "", fmt.Errorf("parse with cache: %w", err)
	}
	for _, entity := range r.Instances {
		if err := writeEntity(r, entity, bd.BaseDir, zipWriter); err != nil {
			return "", fmt.Errorf("write entity: %w", err)
		}
	}

	return fileName, nil
}

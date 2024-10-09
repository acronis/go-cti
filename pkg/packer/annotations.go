package packer

import (
	"fmt"
	"log/slog"

	"github.com/acronis/go-cti/pkg/cti"
)

func defaultAnnotationHandler(baseDir string, writer Writer, key cti.GJsonPath, entity *cti.Entity, a cti.Annotations) error {
	// process asset annotation
	if a.Asset != nil {
		value := key.GetValue(entity.Values)
		assetPath := value.String()
		if assetPath == "" {
			slog.Warn("Empty asset path", slog.String("entity", entity.Cti), slog.String("key", value.Str))
			return nil
		}
		if err := writer.WriteFile(baseDir, assetPath); err != nil {
			return fmt.Errorf("write asset %s: %w", assetPath, err)
		}
	}
	return nil
}

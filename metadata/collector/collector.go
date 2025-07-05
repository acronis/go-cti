package collector

import (
	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/registry"
)

var AnnotationsToMove = []string{consts.Reference, consts.Schema}

type Collector interface {
	// Collect collects metadata from the source and returns a map of collected entities.
	Collect() (*registry.MetadataRegistry, error)
}

type BaseCollector struct {
	BaseDir string

	CTIParser *cti.Parser

	// Local Registry holds entities that are declared by the package.
	Registry *registry.MetadataRegistry
}

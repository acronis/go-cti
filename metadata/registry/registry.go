package registry

import (
	"fmt"

	"github.com/acronis/go-cti/metadata"
)

// MetadataRegistry is a storage for CTI types and instances.
type MetadataRegistry struct {
	Types     metadata.EntityTypeMap
	Instances metadata.EntityInstanceMap
	Index     metadata.EntityMap
}

func (r *MetadataRegistry) Add(entity metadata.Entity) error {
	cti := entity.GetCTI()
	if _, ok := r.Index[cti]; ok {
		return fmt.Errorf("duplicate cti entity %s", cti)
	}

	switch e := entity.(type) {
	case *metadata.EntityInstance:
		r.Instances[cti] = e
	case *metadata.EntityType:
		r.Types[cti] = e
	default:
		return fmt.Errorf("invalid entity: %s", cti)
	}

	r.Index[cti] = entity
	return nil
}

func (r *MetadataRegistry) CopyFrom(registry *MetadataRegistry) error {
	for _, entity := range registry.Types {
		cti := entity.GetCTI()
		if _, ok := r.Index[cti]; ok {
			return fmt.Errorf("duplicate cti entity %s", cti)
		}
		r.Types[cti] = entity
		r.Index[cti] = entity
	}
	for _, entity := range registry.Instances {
		cti := entity.GetCTI()
		if _, ok := r.Index[cti]; ok {
			return fmt.Errorf("duplicate cti entity %s", cti)
		}
		r.Instances[cti] = entity
		r.Index[cti] = entity
	}
	return nil
}

func (r *MetadataRegistry) Clone() *MetadataRegistry {
	c := *r
	return &c
}

func New() *MetadataRegistry {
	return &MetadataRegistry{
		Types:     make(metadata.EntityTypeMap),
		Instances: make(metadata.EntityInstanceMap),
		Index:     make(metadata.EntityMap),
	}
}

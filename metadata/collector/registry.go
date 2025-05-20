package collector

import (
	"fmt"

	"github.com/acronis/go-cti/metadata"
)

type MetadataRegistry struct {
	// TODO: Too many indexes that are not efficient for operations other than add.
	Types     metadata.EntityTypeMap
	Instances metadata.EntityInstanceMap
	Index     metadata.EntityMap
}

func (r *MetadataRegistry) Add(_ string, entity metadata.Entity) error {
	cti := entity.GetCti()
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

func (r *MetadataRegistry) Clone() *MetadataRegistry {
	c := *r
	return &c
}

func NewMetadataRegistry() *MetadataRegistry {
	return &MetadataRegistry{
		Types:     make(metadata.EntityTypeMap),
		Instances: make(metadata.EntityInstanceMap),
		Index:     make(metadata.EntityMap),
	}
}

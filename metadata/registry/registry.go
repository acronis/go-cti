package registry

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

func (r *MetadataRegistry) Add(entity metadata.Entity) error {
	entityID := entity.GetEntityID()
	if _, ok := r.Index[entityID]; ok {
		return fmt.Errorf("duplicate cti entity %s", entity.GetCti())
	}

	switch e := entity.(type) {
	case *metadata.EntityInstance:
		r.Instances[entityID] = e
	case *metadata.EntityType:
		r.Types[entityID] = e
	default:
		return fmt.Errorf("invalid entity: %s", entity.GetCti())
	}

	r.Index[entityID] = entity
	return nil
}

func (r *MetadataRegistry) CopyFrom(registry *MetadataRegistry) error {
	for _, entity := range registry.Types {
		entityID := entity.GetEntityID()
		if _, ok := r.Index[entityID]; ok {
			return fmt.Errorf("duplicate cti entity %s", entity.GetCti())
		}
		r.Types[entityID] = entity
		r.Index[entityID] = entity
	}
	for _, entity := range registry.Instances {
		entityID := entity.GetEntityID()
		if _, ok := r.Index[entityID]; ok {
			return fmt.Errorf("duplicate cti entity %s", entity.GetCti())
		}
		r.Instances[entityID] = entity
		r.Index[entityID] = entity
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

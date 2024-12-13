package collector

import (
	"fmt"

	"github.com/acronis/go-cti/metadata"
)

type MetadataRegistry struct {
	// TODO: Too many indexes that are not efficient on operations other than add.
	Types            metadata.EntitiesMap
	Instances        metadata.EntitiesMap
	FragmentEntities map[string]metadata.Entities
	Index            metadata.EntitiesMap
}

func (r *MetadataRegistry) Add(originalPath string, entity *metadata.Entity) error {
	if _, ok := r.Index[entity.Cti]; ok {
		return fmt.Errorf("duplicate cti entity %s", entity.Cti)
	}

	switch {
	case entity.Values != nil:
		r.Instances[entity.Cti] = entity
	case entity.Schema != nil:
		r.Types[entity.Cti] = entity
	default:
		return fmt.Errorf("invalid entity: %s", entity.Cti)
	}

	r.FragmentEntities[originalPath] = append(r.FragmentEntities[originalPath], entity)
	r.Index[entity.Cti] = entity
	return nil
}

func (r *MetadataRegistry) Clone() *MetadataRegistry {
	c := *r
	return &c
}

func NewMetadataRegistry() *MetadataRegistry {
	return &MetadataRegistry{
		Types:            make(metadata.EntitiesMap),
		Instances:        make(metadata.EntitiesMap),
		Index:            make(metadata.EntitiesMap),
		FragmentEntities: make(map[string]metadata.Entities),
	}
}

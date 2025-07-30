package metadata

import (
	"encoding/json"
	"fmt"

	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

type UntypedEntities []UntypedEntity

// UntypedEntity is an interface for CTI entity that doesn't have a specific type defined at the time of creation.
// Objects that implement this interface can be converted to the typed Entity using ConvertUntypedEntityToTypedEntity function.
// See more information about CTI entities in the [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#metadata-structure
type UntypedEntity interface {
	GetFinal() bool
	GetCTI() string
	GetResilient() bool
	GetAccess() consts.AccessModifier
	GetDisplayName() string
	GetDescription() string
	GetDictionaries() map[string]any
	GetValues() json.RawMessage
	GetSchema() json.RawMessage
	GetTraitsSchema() json.RawMessage
	GetTraitsAnnotations() json.RawMessage
	GetTraits() json.RawMessage
	GetAnnotations() json.RawMessage
	GetSourceMap() UntypedSourceMap
}

type UntypedSourceMap struct {
	TypeAnnotationReference
	InstanceAnnotationReference

	// SourcePath is a relative path to the RAML file where the CTI parent is defined.
	SourcePath string `json:"$sourcePath,omitempty"`

	// OriginalPath is a relative path to RAML fragment where the CTI entity is defined.
	OriginalPath string `json:"$originalPath,omitempty"`
}

type TypeAnnotationReference struct {
	Name string `json:"$name,omitempty"`
}

type InstanceAnnotationReference struct {
	AnnotationType AnnotationType `json:"$annotationType,omitempty"`
}

// ConvertUntypedEntityToTypedEntity converts an UntypedEntity to a typed Entity.
// It checks if the entity has a schema or values, and creates either an EntityType or EntityInstance accordingly.
// If the entity has both schema and values, or neither, it returns an error.
func ConvertUntypedEntityToTypedEntity(untypedEntity UntypedEntity) (Entity, error) {
	rawSchema := untypedEntity.GetSchema()
	rawValues := untypedEntity.GetValues()
	if rawSchema == nil && rawValues == nil {
		return nil, fmt.Errorf("untyped entity %s has neither schema nor values", untypedEntity.GetCTI())
	}
	if rawValues != nil && rawSchema != nil {
		return nil, fmt.Errorf("untyped entity %s has both schema and values, only one is allowed", untypedEntity.GetCTI())
	}

	// If the entity has values but no schema, we treat it as an instance of an entity type.
	if rawValues != nil {
		var values any
		if err := json.Unmarshal(rawValues, &values); err != nil {
			return nil, fmt.Errorf("unmarshal values for %s: %w", untypedEntity.GetCTI(), err)
		}
		e, err := NewEntityInstance(untypedEntity.GetCTI(), values)
		if err != nil {
			return nil, fmt.Errorf("make entity instance: %w", err)
		}
		if !untypedEntity.GetFinal() {
			return nil, fmt.Errorf("untyped entity %s is not final, cannot convert to typed entity instance", untypedEntity.GetCTI())
		}
		if untypedEntity.GetTraitsSchema() != nil {
			return nil, fmt.Errorf("untyped entity %s has traits schema, but it is not allowed for entity instances", untypedEntity.GetCTI())
		}
		if untypedEntity.GetTraits() != nil {
			return nil, fmt.Errorf("untyped entity %s has traits, but it is not allowed for entity instances", untypedEntity.GetCTI())
		}
		if untypedEntity.GetTraitsAnnotations() != nil {
			return nil, fmt.Errorf("untyped entity %s has traits annotations, but it is not allowed for entity instances", untypedEntity.GetCTI())
		}
		if untypedEntity.GetAnnotations() != nil {
			return nil, fmt.Errorf("untyped entity %s has annotations, but it is not allowed for entity instances", untypedEntity.GetCTI())
		}
		e.SetFinal(true)
		e.SetResilient(untypedEntity.GetResilient())
		e.SetAccess(untypedEntity.GetAccess())
		e.SetDisplayName(untypedEntity.GetDisplayName())
		e.SetDescription(untypedEntity.GetDescription())
		untypedSourceMap := untypedEntity.GetSourceMap()
		e.SetSourceMap(EntityInstanceSourceMap{
			AnnotationType: untypedSourceMap.AnnotationType,
			EntitySourceMap: EntitySourceMap{
				OriginalPath: untypedSourceMap.OriginalPath,
				SourcePath:   untypedSourceMap.SourcePath,
			},
		})
		return e, nil
	}

	// If the entity has a schema, we treat it as an entity type.
	var schema jsonschema.JSONSchemaCTI
	if err := json.Unmarshal(rawSchema, &schema); err != nil {
		return nil, fmt.Errorf("unmarshal schema for %s: %w", untypedEntity.GetCTI(), err)
	}
	var annotations map[GJsonPath]*Annotations
	if rawAnnotations := untypedEntity.GetAnnotations(); rawAnnotations != nil {
		if err := json.Unmarshal(rawAnnotations, &annotations); err != nil {
			return nil, fmt.Errorf("unmarshal annotations for %s: %w", untypedEntity.GetCTI(), err)
		}
	}
	e, err := NewEntityType(untypedEntity.GetCTI(), &schema, annotations)
	if err != nil {
		return nil, fmt.Errorf("make entity type: %w", err)
	}
	e.SetFinal(untypedEntity.GetFinal())
	e.SetResilient(untypedEntity.GetResilient())
	e.SetAccess(untypedEntity.GetAccess())
	e.SetDisplayName(untypedEntity.GetDisplayName())
	e.SetDescription(untypedEntity.GetDescription())
	if rawTraitsSchema := untypedEntity.GetTraitsSchema(); rawTraitsSchema != nil {
		var traitsSchema jsonschema.JSONSchemaCTI
		if err = json.Unmarshal(rawTraitsSchema, &traitsSchema); err != nil {
			return nil, fmt.Errorf("unmarshal traits schema for %s: %w", untypedEntity.GetCTI(), err)
		}
		var traitsAnnotations map[GJsonPath]*Annotations
		if rawTraitsAnnotations := untypedEntity.GetTraitsAnnotations(); rawTraitsAnnotations != nil {
			if err = json.Unmarshal(rawTraitsAnnotations, &traitsAnnotations); err != nil {
				return nil, fmt.Errorf("unmarshal traits annotations for %s: %w", untypedEntity.GetCTI(), err)
			}
		}
		e.SetTraitsSchema(&traitsSchema, traitsAnnotations)
	}
	if rawTraits := untypedEntity.GetTraits(); rawTraits != nil {
		var traits map[string]any
		if err = json.Unmarshal(rawTraits, &traits); err != nil {
			return nil, fmt.Errorf("unmarshal traits for %s: %w", untypedEntity.GetCTI(), err)
		}
		e.SetTraits(traits)
	}
	untypedSourceMap := untypedEntity.GetSourceMap()
	e.SetSourceMap(EntityTypeSourceMap{
		Name: untypedSourceMap.Name,
		EntitySourceMap: EntitySourceMap{
			OriginalPath: untypedSourceMap.OriginalPath,
			SourcePath:   untypedSourceMap.SourcePath,
		},
	})
	return e, nil
}

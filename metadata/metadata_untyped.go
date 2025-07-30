package metadata

import (
	"encoding/json"
	"fmt"

	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

// TypedEntityConverter is an interface for converting an entity of other type to Entity.
type TypedEntityConverter interface {
	// AsTypedEntity converts the entity to a typed Entity interface.
	AsTypedEntity() (Entity, error)
}

type UntypedEntities []UntypedEntity

// UntypedEntity represents an untyped CTI entity. It can be used to parse an entity of unknown type
// and then convert it to a typed entity later.
// Follows the same metadata structure as defined in the [CTI specification].
//
// [CTI specification]: https://github.com/acronis/go-cti/blob/main/cti-spec/SPEC.md#metadata-structure
type UntypedEntity struct {
	Final             bool                       `json:"final"`
	CTI               string                     `json:"cti"`
	Resilient         bool                       `json:"resilient"`
	Access            consts.AccessModifier      `json:"access"`
	DisplayName       string                     `json:"display_name,omitempty"`
	Description       string                     `json:"description,omitempty"`
	Dictionaries      map[string]any             `json:"dictionaries,omitempty"`
	Values            json.RawMessage            `json:"values,omitempty"`
	Schema            json.RawMessage            `json:"schema,omitempty"`
	TraitsSchema      json.RawMessage            `json:"traits_schema,omitempty"`
	TraitsAnnotations map[GJsonPath]*Annotations `json:"traits_annotations,omitempty"`
	Traits            json.RawMessage            `json:"traits,omitempty"`
	Annotations       map[GJsonPath]*Annotations `json:"annotations,omitempty"`
	SourceMap         UntypedSourceMap           `json:"source_map,omitempty"`
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
	AnnotationType *AnnotationType `json:"$annotationType,omitempty"`
}

// AsTypedEntity converts the UntypedEntity to a typed Entity interface.
func (ue *UntypedEntity) AsTypedEntity() (Entity, error) {
	switch {
	case ue.Schema != nil:
		var schema jsonschema.JSONSchemaCTI
		if err := json.Unmarshal(ue.Schema, &schema); err != nil {
			return nil, fmt.Errorf("unmarshal schema for %s: %w", ue.CTI, err)
		}
		e, err := NewEntityType(ue.CTI, &schema, ue.Annotations)
		if err != nil {
			return nil, fmt.Errorf("make entity type: %w", err)
		}
		e.SetFinal(ue.Final)
		e.SetResilient(ue.Resilient)
		e.SetAccess(ue.Access)
		e.SetDisplayName(ue.DisplayName)
		e.SetDescription(ue.Description)
		if ue.TraitsSchema != nil {
			var traitsSchema jsonschema.JSONSchemaCTI
			if err = json.Unmarshal(ue.TraitsSchema, &traitsSchema); err != nil {
				return nil, fmt.Errorf("unmarshal traits schema for %s: %w", ue.CTI, err)
			}
			e.SetTraitsSchema(&traitsSchema, ue.TraitsAnnotations)
		}
		if ue.Traits != nil {
			var traits map[string]any
			if err = json.Unmarshal(ue.Traits, &traits); err != nil {
				return nil, fmt.Errorf("unmarshal traits for %s: %w", ue.CTI, err)
			}
			e.SetTraits(traits)
		}
		e.SetSourceMap(EntityTypeSourceMap{
			Name: ue.SourceMap.Name,
			EntitySourceMap: EntitySourceMap{
				OriginalPath: ue.SourceMap.OriginalPath,
				SourcePath:   ue.SourceMap.SourcePath,
			},
		})
		return e, nil
	case ue.Values != nil:
		e, err := NewEntityInstance(ue.CTI, ue.Values)
		if err != nil {
			return nil, fmt.Errorf("make entity instance: %w", err)
		}
		e.SetFinal(true)
		e.SetResilient(ue.Resilient)
		e.SetAccess(ue.Access)
		e.SetDisplayName(ue.DisplayName)
		e.SetDescription(ue.Description)
		e.SetSourceMap(EntityInstanceSourceMap{
			AnnotationType: *ue.SourceMap.AnnotationType,
			EntitySourceMap: EntitySourceMap{
				OriginalPath: ue.SourceMap.OriginalPath,
				SourcePath:   ue.SourceMap.SourcePath,
			},
		})
		return e, nil
	default:
		return nil, fmt.Errorf("untyped entity %s has neither schema nor values", ue.CTI)
	}
}

package metadata

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

type UntypedEntities []UntypedEntity

// UntypedEntity is an interface for CTI entity that doesn't have a specific type defined at the time of creation.
// Objects that implement this interface can be converted to the typed Entity using ConvertUntypedEntityToEntity function.
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
	GetTraitsSourceMap() UntypedSourceMap
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

// GetSourceAnnotations extracts source maps from legacy annotations format in raw annotations JSON.
// It returns a map of GJsonPath to Annotations, a boolean indicating if legacy source annotations were found,
// and an error if any occurred during unmarshalling.
//
// Note that this method is subject to deprecation.
// Source maps must be returned using GetTraitsSourceMap and GetSourceMap methods.
func GetSourceAnnotations(rawAnnotations json.RawMessage) (map[GJsonPath]*Annotations, bool, error) {
	if rawAnnotations == nil {
		return nil, false, nil
	}

	var annotations map[string]json.RawMessage
	if err := json.Unmarshal(rawAnnotations, &annotations); err != nil {
		return nil, false, fmt.Errorf("unmarshal annotations: %w", err)
	}

	entityAnnotations := map[GJsonPath]*Annotations{}
	hasLegacySourceAnnotations := false
	for k, v := range annotations {
		if v == nil {
			continue
		}
		if strings.HasPrefix(k, "$") {
			hasLegacySourceAnnotations = true
			continue
		}
		var ann Annotations
		if err := json.Unmarshal(v, &ann); err != nil {
			return nil, false, fmt.Errorf("unmarshal annotation %s: %w", k, err)
		}
		entityAnnotations[GJsonPath(k)] = &ann
	}
	return entityAnnotations, hasLegacySourceAnnotations, nil
}

// ConvertUntypedEntityToEntity converts an UntypedEntity to a typed Entity.
// It checks if the entity has a schema or values, and creates either an EntityType or EntityInstance accordingly.
// If the entity has both schema and values, or neither, it returns an error.
func ConvertUntypedEntityToEntity(untypedEntity UntypedEntity) (Entity, error) {
	rawSchema := untypedEntity.GetSchema()
	rawValues := untypedEntity.GetValues()
	if rawSchema == nil && rawValues == nil {
		return nil, fmt.Errorf("untyped entity %s has neither schema nor values", untypedEntity.GetCTI())
	}
	if rawValues != nil && rawSchema != nil {
		return nil, fmt.Errorf("untyped entity %s has both schema and values, only one is allowed", untypedEntity.GetCTI())
	}

	rawAnnotations := untypedEntity.GetAnnotations()
	annotations, hasLegacySourceAnnotations, err := GetSourceAnnotations(rawAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get annotations for %s: %w", untypedEntity.GetCTI(), err)
	}

	// If the entity has values but no schema, we treat it as an instance of an entity type.
	if rawValues != nil {
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

		var sourceMap InstanceSourceMap
		if !hasLegacySourceAnnotations {
			if untypedEntity.GetAnnotations() != nil {
				return nil, fmt.Errorf("untyped entity %s has annotations, but it is not allowed for entity instances", untypedEntity.GetCTI())
			}
			untypedSourceMap := untypedEntity.GetSourceMap()
			sourceMap = InstanceSourceMap{
				AnnotationType: untypedSourceMap.AnnotationType,
				DocumentSourceMap: DocumentSourceMap{
					OriginalPath: untypedSourceMap.OriginalPath,
					SourcePath:   untypedSourceMap.SourcePath,
				},
			}
		} else {
			if len(annotations) != 0 {
				return nil, fmt.Errorf("untyped entity %s has annotations, but it is not allowed for entity instances", untypedEntity.GetCTI())
			}
			if err := json.Unmarshal(rawAnnotations, &sourceMap); err != nil {
				return nil, fmt.Errorf("unmarshal source map for %s: %w", untypedEntity.GetCTI(), err)
			}
		}

		var values any
		if err := json.Unmarshal(rawValues, &values); err != nil {
			return nil, fmt.Errorf("unmarshal values for %s: %w", untypedEntity.GetCTI(), err)
		}
		e, err := NewEntityInstance(untypedEntity.GetCTI(), values)
		if err != nil {
			return nil, fmt.Errorf("make entity instance: %w", err)
		}
		e.SetFinal(true)
		e.SetResilient(untypedEntity.GetResilient())
		e.SetAccess(untypedEntity.GetAccess())
		e.SetDisplayName(untypedEntity.GetDisplayName())
		e.SetDescription(untypedEntity.GetDescription())
		e.SetSourceMap(&sourceMap)
		return e, nil
	}

	// If the entity has a schema, we treat it as an entity type.
	var schema jsonschema.JSONSchemaCTI
	if err := json.Unmarshal(rawSchema, &schema); err != nil {
		return nil, fmt.Errorf("unmarshal schema for %s: %w", untypedEntity.GetCTI(), err)
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
		rawTraitsAnnotations := untypedEntity.GetTraitsAnnotations()
		traitsAnnotations, hasLegacySourceAnnotations, err := GetSourceAnnotations(rawTraitsAnnotations)
		if err != nil {
			return nil, fmt.Errorf("get traits annotations for %s: %w", untypedEntity.GetCTI(), err)
		}
		e.SetTraitsSchema(&traitsSchema, traitsAnnotations)

		var sourceMap TypeSourceMap
		if !hasLegacySourceAnnotations {
			untypedSourceMap := untypedEntity.GetTraitsSourceMap()
			sourceMap = TypeSourceMap{
				Name: untypedSourceMap.Name,
				DocumentSourceMap: DocumentSourceMap{
					OriginalPath: untypedSourceMap.OriginalPath,
					SourcePath:   untypedSourceMap.SourcePath,
				},
			}
		} else {
			if err := json.Unmarshal(rawTraitsAnnotations, &sourceMap); err != nil {
				return nil, fmt.Errorf("unmarshal source map for %s: %w", untypedEntity.GetCTI(), err)
			}
		}

		e.SetTraitsSourceMap(&sourceMap)
	}
	if rawTraits := untypedEntity.GetTraits(); rawTraits != nil {
		var traits map[string]any
		if err = json.Unmarshal(rawTraits, &traits); err != nil {
			return nil, fmt.Errorf("unmarshal traits for %s: %w", untypedEntity.GetCTI(), err)
		}
		e.SetTraits(traits)
	}
	var sourceMap TypeSourceMap
	if !hasLegacySourceAnnotations {
		untypedSourceMap := untypedEntity.GetSourceMap()
		sourceMap = TypeSourceMap{
			Name: untypedSourceMap.Name,
			DocumentSourceMap: DocumentSourceMap{
				OriginalPath: untypedSourceMap.OriginalPath,
				SourcePath:   untypedSourceMap.SourcePath,
			},
		}
	} else {
		if err := json.Unmarshal(rawAnnotations, &sourceMap); err != nil {
			return nil, fmt.Errorf("unmarshal source map for %s: %w", untypedEntity.GetCTI(), err)
		}
	}
	e.SetSourceMap(&sourceMap)
	return e, nil
}

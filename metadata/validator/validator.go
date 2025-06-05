package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-stacktrace"
	"github.com/xeipuuv/gojsonschema"
)

const (
	TrueStr = "true"
)

type MetadataValidator struct {
	registry  *collector.MetadataRegistry
	ctiParser *cti.Parser
}

func MakeMetadataValidator(r *collector.MetadataRegistry) *MetadataValidator {
	return &MetadataValidator{
		ctiParser: cti.NewParser(),
		registry:  r,
	}
}

func (v *MetadataValidator) ValidateAll() error {
	st := stacktrace.StackTrace{}
	for _, object := range v.registry.Index {
		if err := v.Validate(object); err != nil {
			_ = st.Append(stacktrace.NewWrapped("validation failed", err, stacktrace.WithInfo("cti", object.GetCti()), stacktrace.WithType("validation")))
		}
	}
	if len(st.List) > 0 {
		return &st
	}

	return nil
}

func (v *MetadataValidator) Validate(object metadata.Entity) error {
	if err := v.validateBaseProperties(object); err != nil {
		return fmt.Errorf("%s: %w", object.GetCti(), err)
	}
	switch entity := object.(type) {
	case *metadata.EntityType:
		if err := v.ValidateType(entity); err != nil {
			return fmt.Errorf("%s: %w", object.GetCti(), err)
		}
	case *metadata.EntityInstance:
		if err := v.ValidateInstance(entity); err != nil {
			return fmt.Errorf("%s: %w", object.GetCti(), err)
		}
	default:
		return fmt.Errorf("%s: invalid type", object.GetCti())
	}
	return nil
}

func (v *MetadataValidator) validateBaseProperties(object metadata.Entity) error {
	currentCti := object.GetCti()
	parent := object.Parent()
	// TODO: Check presence of parents in chain according to expression.
	if parent != nil {
		parentCti := parent.GetCti()
		ok, err := parent.Match(object)
		if err != nil {
			return fmt.Errorf("failed to match %s with parent %s: %w", currentCti, parentCti, err)
		}
		if !ok {
			return fmt.Errorf("%s doesn't match %s", currentCti, parentCti)
		}
		// if parent.Access.Integer() > object.GetAccess().Integer() {
		// 	return fmt.Errorf("%s access is less restrictive than parent %s", currentCti, parentCti)
		// }
		// if err = parent.IsAccessibleBy(object); err != nil {
		// 	return fmt.Errorf("%s is not accessible by %s: %w", currentCti, parentCti, err)
		// }
	}
	return nil
}

func (v *MetadataValidator) ValidateType(entity *metadata.EntityType) error {
	if entity == nil {
		return errors.New("entity is nil")
	}

	if entity.Schema == nil {
		return fmt.Errorf("%s type has no schema", entity.Cti)
	}

	currentCti := entity.GetCti()
	parent := entity.Parent()
	if parent != nil && parent.IsFinal() {
		return fmt.Errorf("%s is derived from final type %s", currentCti, parent.GetCti())
	}

	if err := validateJSONSchema(entity.Schema); err != nil {
		return fmt.Errorf("%s contains invalid schema: %w", entity.Cti, err)
	}

	if entity.Traits != nil {
		if parent == nil {
			return fmt.Errorf("%s type has no parent type", entity.Cti)
		}
		parentTraitsSchema := parent.FindTraitsSchemaInChain()
		if parentTraitsSchema == nil {
			return fmt.Errorf("%s type specifies traits but none of the parents define traits schema", entity.Cti)
		}
		if err := validateJSONValues(parentTraitsSchema, entity.Traits); err != nil {
			return fmt.Errorf("%s contains invalid values: %w", entity.Cti, err)
		}
	}

	if entity.TraitsSchema != nil {
		if err := validateJSONSchema(entity.TraitsSchema); err != nil {
			return fmt.Errorf("%s contains invalid schema: %w", entity.Cti, err)
		}
		if entity.TraitsAnnotations != nil {
			for key, annotation := range entity.TraitsAnnotations {
				if err := v.validateTypeReference(key, annotation, entity, parent); err != nil {
					return fmt.Errorf("%s@%s: %w", currentCti, key, err)
				}
				if err := v.validateCtiSchema(key, annotation, entity, parent); err != nil {
					return fmt.Errorf("%s@%s: %w", currentCti, key, err)
				}
			}
		}
	}

	if entity.Annotations != nil {
		for key, annotation := range entity.Annotations {
			if err := v.validateTypeReference(key, annotation, entity, parent); err != nil {
				return fmt.Errorf("%s@%s: %w", currentCti, key, err)
			}
			if err := v.validateCtiSchema(key, annotation, entity, parent); err != nil {
				return fmt.Errorf("%s@%s: %w", currentCti, key, err)
			}
		}
	}

	return nil
}

func (v *MetadataValidator) validateCtiSchema(_ metadata.GJsonPath, annotation *metadata.Annotations, child, _ *metadata.EntityType) error {
	currentRefs := annotation.ReadCtiSchema()
	for _, currentRef := range currentRefs {
		refObject := v.registry.Index[currentRef]
		if refObject == nil {
			return fmt.Errorf("cti schema %s not found", currentRef)
		}
		// if err := refObject.IsAccessibleBy(child); err != nil {
		// 	return fmt.Errorf("cti schema %s is not accessible by %s: %w", currentRef, child.GetCti(), err)
		// }
	}
	return nil
}

func (v *MetadataValidator) validateTypeReference(key metadata.GJsonPath, annotation *metadata.Annotations, child *metadata.EntityType, parent *metadata.EntityType) error {
	currentRef := annotation.ReadReference()
	if currentRef == "" {
		return nil
	}
	if parent != nil {
		parentAnnotations := parent.FindAnnotationsByPredicateInChain(key, func(a *metadata.Annotations) bool {
			return a.Reference != nil
		})
		if parentAnnotations != nil {
			parentRef := parentAnnotations.ReadReference()
			if parentRef != TrueStr && currentRef == TrueStr {
				return errors.New("parent cti.reference defines a specific CTI, but child specifies true")
			}
			if parentRef != TrueStr {
				expr, err := v.ctiParser.Parse(parentRef)
				if err != nil {
					return fmt.Errorf("failed to parse parent cti.reference %s: %w", parentRef, err)
				}
				if err = v.matchCti(&expr, currentRef); err != nil {
					return fmt.Errorf("cti.reference %s does not match parent reference %s: %w", currentRef, parentRef, err)
				}
			}
		}
		if currentRef == TrueStr {
			return nil
		}
	} else {
		if currentRef == TrueStr {
			return nil
		}
		if _, err := v.ctiParser.Parse(currentRef); err != nil {
			return fmt.Errorf("failed to parse cti.reference %s: %w", currentRef, err)
		}
	}
	refObject := v.registry.Index[currentRef]
	if refObject == nil {
		return fmt.Errorf("reference %s not found", currentRef)
	}
	// if err := refObject.IsAccessibleBy(child); err != nil {
	// 	return fmt.Errorf("reference %s is not accessible by %s: %w", currentRef, child.GetCti(), err)
	// }
	return nil
}

func (v *MetadataValidator) ValidateInstance(entity *metadata.EntityInstance) error {
	if entity == nil {
		return errors.New("entity is nil")
	}

	if entity.Values == nil {
		return fmt.Errorf("%s instance has no values", entity.Cti)
	}

	currentCti := entity.GetCti()
	parent := entity.Parent()
	if parent == nil {
		return fmt.Errorf("%s instance has no parent type", currentCti)
	}
	// TODO: Move to entity.Validate()
	mergedSchema, err := parent.GetMergedSchema()
	if err != nil {
		return fmt.Errorf("failed to get merged schema for %s: %w", parent.GetCti(), err)
	}
	if err = validateJSONValues(mergedSchema, entity.Values); err != nil {
		return fmt.Errorf("%s contains invalid values: %w", currentCti, err)
	}
	if parent.Annotations != nil {
		values, err := entity.GetRawValues()
		if err != nil {
			return fmt.Errorf("failed to get raw values for %s: %w", currentCti, err)
		}
		for key := range parent.Annotations {
			if err = v.validateInstanceReference(key, entity, parent, values); err != nil {
				return fmt.Errorf("%s@%s: %w", currentCti, key, err)
			}
		}
	}
	return nil
}

func (v *MetadataValidator) validateInstanceReference(key metadata.GJsonPath, child *metadata.EntityInstance, parent *metadata.EntityType, values []byte) error {
	annotation := parent.FindAnnotationsByPredicateInChain(key, func(a *metadata.Annotations) bool {
		return a.Reference != nil
	})
	if annotation == nil {
		return nil
	}

	ref := annotation.ReadReference()
	if ref == TrueStr {
		return nil
	}

	refObject := v.registry.Index[ref]
	if refObject == nil {
		return fmt.Errorf("reference %s not found", ref)
	}
	// if err := refObject.IsAccessibleBy(child); err != nil {
	// 	return fmt.Errorf("reference %s is not accessible by %s: %w", ref, child.Cti, err)
	// }

	expr, err := v.ctiParser.Parse(ref)
	if err != nil {
		return fmt.Errorf("failed to parse cti.reference %s: %w", ref, err)
	}
	value := key.GetValue(values)
	for _, val := range value.Array() {
		if err = v.matchCti(&expr, val.Str); err != nil {
			return fmt.Errorf("cti.reference %s does not match value %s: %w", ref, val.Str, err)
		}
	}
	return nil
}

func (v *MetadataValidator) matchCti(ref *cti.Expression, id string) error {
	val, err := v.ctiParser.Parse(id)
	if err != nil {
		return fmt.Errorf("failed to parse cti %s: %w", id, err)
	}
	if ok, err := ref.Match(val); !ok {
		if err != nil {
			return fmt.Errorf("%s doesn't match: %w", id, err)
		}
		return fmt.Errorf("%s doesn't match", id)
	}
	return nil
}

func validateJSONSchema(schema interface{}) error {
	sl := gojsonschema.NewSchemaLoader()
	sl.Validate = true
	return sl.AddSchemas(gojsonschema.NewGoLoader(schema))
}

func validateJSONValues(schema interface{}, document interface{}) error {
	sl := gojsonschema.NewGoLoader(schema)
	dl := gojsonschema.NewGoLoader(document)
	res, err := gojsonschema.Validate(sl, dl)
	if err != nil {
		return fmt.Errorf("failed to validate JSON values: %w", err)
	}
	if !res.Valid() {
		errs := res.Errors()
		var b strings.Builder
		for _, err := range errs {
			b.WriteString("\n- ")
			b.WriteString(err.Description())
		}
		return errors.New(b.String())
	}
	return nil
}

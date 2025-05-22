package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/merger"
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
		return fmt.Errorf("%s %s", object.GetCti(), err.Error())
	}
	switch entity := object.(type) {
	case *metadata.EntityType:
		if err := v.ValidateType(entity); err != nil {
			return fmt.Errorf("%s %s", object.GetCti(), err.Error())
		}
	case *metadata.EntityInstance:
		if err := v.ValidateInstance(entity); err != nil {
			return fmt.Errorf("%s %s", object.GetCti(), err.Error())
		}
	default:
		return fmt.Errorf("%s: invalid type", object.GetCti())
	}
	return nil
}

func (v *MetadataValidator) validateBaseProperties(object metadata.Entity) error {
	currentCti := object.GetCti()
	parent := object.Parent()
	if parent != nil {
		parentCti := parent.GetCti()
		ok, err := parent.Expression().Match(*object.Expression())
		if err != nil {
			return fmt.Errorf("%s %s", currentCti, err.Error())
		}
		if !ok {
			return fmt.Errorf("%s doesn't match %s", currentCti, parentCti)
		}
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
	if parent != nil {
		parentCti := parent.GetCti()
		if parent.IsFinal() {
			return fmt.Errorf("%s is derived from final type %s", currentCti, parentCti)
		}
	}

	if err := validateJSONSchema(entity.Schema); err != nil {
		return fmt.Errorf("%s contains invalid schema: %s", entity.Cti, err)
	}

	if entity.Traits != nil {
		if parent == nil {
			return fmt.Errorf("%s type has no parent type", entity.Cti)
		}
		parentTraitsSchema := parent.FindTraitsSchemaInChain()
		if parentTraitsSchema == nil {
			return fmt.Errorf("%s type is derived from type that does not define traits schema", entity.Cti)
		}
		if err := validateJSONValues(parentTraitsSchema, entity.Traits); err != nil {
			return fmt.Errorf("%s contains invalid values: %w", entity.Cti, err)
		}
	}

	if entity.TraitsSchema != nil {
		if err := validateJSONSchema(entity.TraitsSchema); err != nil {
			return fmt.Errorf("%s contains invalid schema: %w", entity.Cti, err)
		}
	}

	if entity.Annotations != nil {
		for key, annotation := range entity.Annotations {
			currentRef := annotation.ReadReference()
			if currentRef == "" {
				continue
			}
			parentAnnotations := entity.FindAnnotationsKeyInChain(key)
			if parentAnnotations == nil {
				if currentRef == TrueStr {
					continue
				}
				if _, err := v.ctiParser.Parse(currentRef); err != nil {
					return fmt.Errorf("%s@%s: %s", entity.Cti, key, err.Error())
				}
				continue
			}
			parentRef := parentAnnotations.ReadReference()
			if parentRef != TrueStr && currentRef == TrueStr {
				return fmt.Errorf("%s@%s: parent cti.reference defines a specific CTI, but child specifies true", entity.Cti, key)
			}
			// If either the parent or the current reference is true, then we don't need to validate the reference
			if currentRef == TrueStr || parentRef == TrueStr {
				continue
			}
			expr, err := v.ctiParser.Parse(parentRef)
			if err != nil {
				return fmt.Errorf("%s@%s: %s", entity.Cti, key, err.Error())
			}
			if err = v.matchCti(&expr, currentRef); err != nil {
				return fmt.Errorf("%s@%s: %s", entity.Cti, key, err.Error())
			}
		}
	}

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
	parentCti := parent.GetCti()
	mergedSchema, err := merger.GetMergedCtiSchema(parentCti, v.registry)
	if err != nil {
		return err
	}
	if err = validateJSONValues(mergedSchema, entity.Values); err != nil {
		return fmt.Errorf("%s contains invalid values: %w", currentCti, err)
	}
	if parent.Annotations != nil {
		values, _ := json.Marshal(entity.Values)
		for key, annotation := range parent.Annotations {
			if ref := annotation.ReadReference(); ref != "" && ref != TrueStr {
				value := key.GetValue(values)
				if ref, err := v.ctiParser.Parse(ref); err == nil {
					for _, val := range value.Array() {
						if err = v.matchCti(&ref, val.Str); err != nil {
							return fmt.Errorf("%s@%s: %s in %s", currentCti, key, err.Error(), val.Str)
						}
					}
				} else {
					return fmt.Errorf("%s@%s: failed to parse cti.reference. Reason: %s", currentCti, key, err.Error())
				}
			}
			// if l10n := annotation.L10N; l10n != nil {
			// 	fmt.Printf("key: [%s][cti.l10n]: %t\n", key, *l10n)
			// }
		}
	}
	return nil
}

func (v *MetadataValidator) matchCti(ref *cti.Expression, id string) error {
	val, err := v.ctiParser.Parse(id)
	if err != nil {
		return fmt.Errorf("%s %s", id, err.Error())
	}
	if ok, err := ref.Match(val); !ok {
		if err != nil {
			return fmt.Errorf("%s doesn't match. Reason: %s", id, err.Error())
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
		return err
	}
	if !res.Valid() {
		errs := res.Errors()
		str := make([]string, len(res.Errors()))
		for i, err := range errs {
			str[i] = err.Description()
		}
		return errors.New(strings.Join(str, "\n-"))
	}
	return nil
}

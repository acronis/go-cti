package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/acronis/go-cti/metadata/merger"
	"github.com/acronis/go-stacktrace"
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
	for _, entity := range v.registry.Index {
		if err := v.Validate(entity); err != nil {
			_ = st.Append(stacktrace.NewWrapped("validation failed", err, stacktrace.WithInfo("cti", entity.Cti), stacktrace.WithType("validation")))
		}
	}
	if len(st.List) > 0 {
		return &st
	}

	return nil
}

func (v *MetadataValidator) Validate(current *metadata.Entity) error {
	// TODO: Pre-parse all CTIs into expressions
	currentCtiExpr, err := v.ctiParser.Parse(current.Cti)
	if err != nil {
		return fmt.Errorf("%s %s", current.Cti, err.Error())
	}

	parentCti := metadata.GetParentCti(current.Cti)
	if parentCti == current.Cti {
		if current.Schema != nil {
			schema := []byte(current.Schema)
			if err := validateBytesJsonSchema(schema); err != nil {
				return fmt.Errorf("%s contains invalid schema: %s", current.Cti, err)
			}
		}
		if current.TraitsSchema != nil {
			schema := []byte(current.TraitsSchema)
			if err := validateBytesJsonSchema(schema); err != nil {
				return fmt.Errorf("%s contains invalid schema: %s", current.Cti, err)
			}
		}
		return nil
	}

	parent, ok := v.registry.Index[parentCti]
	if !ok {
		return fmt.Errorf("%s failed to find parent type", current.Cti)
	}
	if parent.Final && current.Values == nil {
		return fmt.Errorf("%s is derived from final type", current.Cti)
	}
	// TODO: Need to memoize validated schemas and values for better performance
	if current.Values != nil {
		if parent.Schema == nil {
			return fmt.Errorf("%s instance is derived from non-type CTI", current.Cti)
		}
		mergedSchema, err := merger.GetMergedCtiSchema(parent.Cti, v.registry)
		if err != nil {
			return err
		}
		values := []byte(current.Values)
		if err := validateGoJsonValues(mergedSchema, values); err != nil {
			return fmt.Errorf("%s contains invalid values: %s", current.Cti, err)
		}
		if parent.Annotations != nil {
			// TODO: Ensure correct cti.id field is used
			for key, annotation := range parent.Annotations {
				// if ctis := annotation.ReadCti(); len(ctis) > 0 {
				// 	for _, id := range ctis {
				// 		fmt.Printf("key: [%s][cti.cti]: %s", key, id)
				// 	}
				// }
				if parent, err := v.ctiParser.Parse(parent.Cti); err == nil {
					if ok, err := parent.Match(currentCtiExpr); !ok {
						if err != nil {
							return fmt.Errorf("%s: invalid inheritance. Reason: %s", current.Cti, err.Error())
						}

						return fmt.Errorf("%s: invalid inheritance", current.Cti)
					}
				}
				if ref := annotation.ReadReference(); ref != "" && ref != TrueStr {
					value := key.GetValue(values)
					if ref, err := v.ctiParser.Parse(ref); err == nil {
						for _, val := range value.Array() {
							err := v.matchCti(&ref, val.Str)
							if err != nil {
								return fmt.Errorf("%s@%s: %s in %s", current.Cti, key, err.Error(), val.Str)
							}
						}
					} else {
						return fmt.Errorf("%s@%s: failed to parse cti.reference. Reason: %s", current.Cti, key, err.Error())
					}
				}
				// if l10n := annotation.L10N; l10n != nil {
				// 	fmt.Printf("key: [%s][cti.l10n]: %t\n", key, *l10n)
				// }
			}
		} else {
			return fmt.Errorf("%s does not have any annotations", current.Cti)
		}
	}
	if current.Traits != nil {
		id := metadata.GetBaseCti(parentCti)
		base, ok := v.registry.Index[id]
		if !ok {
			return fmt.Errorf("%s failed to find base type", current.Cti)
		}
		// FIXME: Need to obtain traits from the parent
		if base.TraitsSchema == nil {
			return fmt.Errorf("%s type is derived from type that does not define traits", current.Cti)
		}
		schema, values := []byte(base.TraitsSchema), []byte(current.Traits)
		if err := validateBytesJsonValues(schema, values); err != nil {
			return fmt.Errorf("%s contains invalid values: %s", current.Cti, err)
		}
	}
	if current.Schema != nil {
		schema := []byte(current.Schema)
		if err := validateBytesJsonSchema(schema); err != nil {
			return fmt.Errorf("%s contains invalid schema: %s", current.Cti, err)
		}
	}
	if current.TraitsSchema != nil {
		schema := []byte(current.TraitsSchema)
		if err := validateBytesJsonSchema(schema); err != nil {
			return fmt.Errorf("%s contains invalid schema: %s", current.Cti, err)
		}
	}
	if current.Annotations != nil {
		for key, annotation := range current.Annotations {
			currentRef := annotation.ReadReference()
			if currentRef == "" {
				continue
			}
			parentAnnotations := v.FindInheritedAnnotation(current.Cti, key, func(a *metadata.Annotations) bool { return a.Reference != nil })
			if parentAnnotations == nil {
				if currentRef == TrueStr {
					continue
				}
				if _, err := v.ctiParser.Parse(currentRef); err != nil {
					return fmt.Errorf("%s@%s: %s", current.Cti, key, err.Error())
				}
				continue
			}
			parentRef := parentAnnotations.ReadReference()
			if parentRef != TrueStr && currentRef == TrueStr {
				return fmt.Errorf("%s@%s: parent cti.reference defines a specific CTI, but child specifies true", current.Cti, key)
			}
			// If either the parent or the current reference is true, then we don't need to validate the reference
			if currentRef == TrueStr || parentRef == TrueStr {
				continue
			}
			expr, err := v.ctiParser.Parse(parentRef)
			if err != nil {
				return fmt.Errorf("%s@%s: %s", current.Cti, key, err.Error())
			}
			if err := v.matchCti(&expr, currentRef); err != nil {
				return fmt.Errorf("%s@%s: %s", current.Cti, key, err.Error())
			}
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

func (v *MetadataValidator) FindInheritedAnnotation(
	id string, key metadata.GJsonPath, predicate func(*metadata.Annotations) bool,
) *metadata.Annotations {
	root := id
	for {
		parentCti := metadata.GetParentCti(root)

		entity, ok := v.registry.Index[parentCti]
		if !ok {
			return nil
		}
		if val, ok := entity.Annotations[key]; ok && predicate(&val) {
			return &val
		}
		if parentCti == entity.Cti {
			break
		}
		root = parentCti
	}
	return nil
}

func validateBytesJsonSchema(schema []byte) error {
	sl := gojsonschema.NewSchemaLoader()
	sl.Validate = true
	return sl.AddSchemas(gojsonschema.NewBytesLoader(schema))
}

func validateBytesJsonValues(schema []byte, document []byte) error {
	sl := gojsonschema.NewBytesLoader(schema)
	dl := gojsonschema.NewBytesLoader(document)
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

func validateGoJsonValues(schema map[string]interface{}, document []byte) error {
	sl := gojsonschema.NewGoLoader(schema)
	dl := gojsonschema.NewBytesLoader(document)
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

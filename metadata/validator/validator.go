package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/acronis/go-cti/metadata/registry"
	"github.com/acronis/go-stacktrace"
	"github.com/xeipuuv/gojsonschema"
)

const (
	TrueStr = "true"
)

type TypeHook func(v *MetadataValidator, e *metadata.EntityType) error
type InstanceHook func(v *MetadataValidator, e *metadata.EntityInstance) error

type MetadataValidator struct {
	localRegistry  *registry.MetadataRegistry
	globalRegistry *registry.MetadataRegistry
	ctiParser      *cti.Parser
	vendor         string
	pkg            string

	typeHooks     map[string][]TypeHook
	instanceHooks map[string][]InstanceHook

	typeCache     map[string][]TypeHook
	instanceCache map[string][]InstanceHook

	// CustomData is a map of custom data that can be used by hooks.
	CustomData map[string]any

	metaSchema              *gojsonschema.Schema
	schemaLoaderCache       map[string]*gojsonschema.Schema
	traitsSchemaLoaderCache map[string]*gojsonschema.Schema

	// TODO: Probably need global expressions cache.
	expressionsCache map[string]*cti.Expression
}

func MakeMetadataValidator(vendor, pkg string, gr, lr *registry.MetadataRegistry) *MetadataValidator {
	return &MetadataValidator{
		ctiParser:      cti.NewParser(),
		globalRegistry: gr,
		localRegistry:  lr,
		vendor:         vendor,
		pkg:            pkg,

		typeHooks:     make(map[string][]TypeHook),
		instanceHooks: make(map[string][]InstanceHook),

		typeCache:     make(map[string][]TypeHook),
		instanceCache: make(map[string][]InstanceHook),

		CustomData: make(map[string]any),

		// NOTE: gojsonschema loads and compiles the meta-schema from the URL without caching on each validation.
		// Here we pre-compile the meta-schema from local source to avoid network calls and recompilation.
		metaSchema:              MustCompileSchema(jsonschema.MetaSchemaDraft07),
		schemaLoaderCache:       make(map[string]*gojsonschema.Schema),
		traitsSchemaLoaderCache: make(map[string]*gojsonschema.Schema),
	}
}

func (v *MetadataValidator) ValidateAll() error {
	st := stacktrace.StackTrace{}
	for _, object := range v.localRegistry.Index {
		if err := v.Validate(object); err != nil {
			_ = st.Append(stacktrace.NewWrapped("validation failed", err, stacktrace.WithInfo("cti", object.GetCTI()), stacktrace.WithType("validation")))
		}
	}
	if len(st.List) > 0 {
		return &st
	}

	return nil
}

func (v *MetadataValidator) OnType(id string, h TypeHook) error {
	if _, ok := v.localRegistry.Types[id]; !ok {
		return fmt.Errorf("type %s not found in local registry", id)
	}
	v.typeHooks[id] = append(v.typeHooks[id], h)
	delete(v.typeCache, id)
	return nil
}

func (v *MetadataValidator) OnInstanceOfType(id string, h InstanceHook) error {
	if _, ok := v.localRegistry.Instances[id]; !ok {
		return fmt.Errorf("instance %s not found in local registry", id)
	}
	v.instanceHooks[id] = append(v.instanceHooks[id], h)
	delete(v.instanceCache, id)
	return nil
}

func (v *MetadataValidator) collectTypeHooks(entity *metadata.EntityType) []TypeHook {
	if hs, ok := v.typeCache[entity.CTI]; ok {
		return hs
	}
	var out []TypeHook
	for root := entity; root != nil; root = root.Parent() {
		out = append(out, v.typeHooks[root.CTI]...)
	}
	v.typeCache[entity.CTI] = out
	return out
}

func (v *MetadataValidator) collectInstanceHooks(entity *metadata.EntityInstance) []InstanceHook {
	if hs, ok := v.instanceCache[entity.CTI]; ok {
		return hs
	}
	var out []InstanceHook
	// TODO: This means we cannot put a hook directly on an instance.
	// But maybe it's not required anyway.
	for root := entity.Parent(); root != nil; root = root.Parent() {
		out = append(out, v.instanceHooks[root.CTI]...)
	}
	v.instanceCache[entity.CTI] = out
	return out
}

func (v *MetadataValidator) Validate(object metadata.Entity) error {
	if err := v.validateBaseProperties(object); err != nil {
		return fmt.Errorf("%s: %w", object.GetCTI(), err)
	}
	switch entity := object.(type) {
	case *metadata.EntityType:
		if err := v.ValidateType(entity); err != nil {
			return fmt.Errorf("%s: %w", object.GetCTI(), err)
		}
	case *metadata.EntityInstance:
		if err := v.ValidateInstance(entity); err != nil {
			return fmt.Errorf("%s: %w", object.GetCTI(), err)
		}
	default:
		return fmt.Errorf("%s: invalid type", object.GetCTI())
	}
	return nil
}

func (v *MetadataValidator) validateBaseProperties(object metadata.Entity) error {
	currentCti := object.GetCTI()
	parent := object.Parent()
	// TODO: Check presence of parents in chain according to expression.
	if object.Vendor() != v.vendor {
		return fmt.Errorf("%s vendor %s doesn't match expected %s", currentCti, object.Vendor(), v.vendor)
	}
	if object.Package() != v.pkg {
		return fmt.Errorf("%s package %s doesn't match expected %s", currentCti, object.Package(), v.pkg)
	}
	if parent != nil {
		parentCti := parent.GetCTI()
		if !object.IsA(parent) {
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
		return fmt.Errorf("%s type has no schema", entity.CTI)
	}

	for _, h := range v.collectTypeHooks(entity) {
		if err := h(v, entity); err != nil {
			return err
		}
	}

	currentCti := entity.GetCTI()
	parent := entity.Parent()
	if parent != nil && parent.IsFinal() {
		return fmt.Errorf("%s is derived from final type %s", currentCti, parent.GetCTI())
	}

	mergedSchema, err := entity.GetMergedSchema()
	if err != nil {
		return fmt.Errorf("failed to get merged schema for %s: %w", currentCti, err)
	}
	if _, err := v.getOrCacheSchema(entity.CTI, mergedSchema, v.schemaLoaderCache); err != nil {
		return fmt.Errorf("%s contains invalid schema: %w", entity.CTI, err)
	}

	if entity.Traits != nil {
		if parent == nil {
			return fmt.Errorf("%s type has no parent type", entity.CTI)
		}
		parentWithTraits := parent.FindEntityTypeByPredicate(func(e *metadata.EntityType) bool {
			return e.TraitsSchema != nil
		})
		if parentWithTraits == nil {
			return fmt.Errorf("%s type specifies traits but none of the parents define traits schema", entity.CTI)
		}
		s, err := v.getOrCacheSchema(parentWithTraits.CTI, parentWithTraits.TraitsSchema, v.traitsSchemaLoaderCache)
		if err != nil {
			return fmt.Errorf("%s traits schema is invalid: %w", parentWithTraits.CTI, err)
		}
		if err := v.validateJSONDocument(s, gojsonschema.NewGoLoader(entity.Traits)); err != nil {
			return fmt.Errorf("%s contains invalid values: %w", entity.CTI, err)
		}
		if parentWithTraits.TraitsAnnotations != nil {
			for key, annotations := range parentWithTraits.TraitsAnnotations {
				values, err := entity.GetRawTraits()
				if err != nil {
					return fmt.Errorf("failed to get raw trait values for %s: %w", currentCti, err)
				}
				if err := v.validateValueReference(key, entity, annotations, values); err != nil {
					return fmt.Errorf("%s@traits%s: %w", currentCti, key, err)
				}
			}
		}
	}

	if entity.TraitsSchema != nil {
		if _, err := v.getOrCacheSchema(entity.CTI, entity.TraitsSchema, v.traitsSchemaLoaderCache); err != nil {
			return fmt.Errorf("%s contains invalid schema: %w", entity.CTI, err)
		}
		if entity.TraitsAnnotations != nil {
			for key, annotation := range entity.TraitsAnnotations {
				// NOTE: Traits annotations are not inherited from parent.
				if err := v.validateTypeReference(key, annotation, entity, nil); err != nil {
					return fmt.Errorf("%s@%s: %w", currentCti, key, err)
				}
				if err := v.validateCtiSchema(key, annotation, entity, nil); err != nil {
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
			if err := v.validateCtiSchema(key, annotation, entity, nil); err != nil {
				return fmt.Errorf("%s@%s: %w", currentCti, key, err)
			}
		}
	}

	return nil
}

func (v *MetadataValidator) validateCtiSchema(_ metadata.GJsonPath, annotation *metadata.Annotations, child, _ *metadata.EntityType) error {
	schemaRefs := annotation.ReadCTISchema()
	for _, schemaRef := range schemaRefs {
		if schemaRef == "null" {
			continue
		}
		expr, err := v.getOrCacheExpression(schemaRef, v.ctiParser.Parse)
		if err != nil {
			return fmt.Errorf("failed to parse parent cti.schema %s: %w", schemaRef, err)
		}
		attributeSelector := string(expr.AttributeSelector)
		// Strip the attribute selector from the ID.
		if attributeSelector != "" {
			schemaRef = schemaRef[:len(schemaRef)-len(attributeSelector)-1]
		}
		refObject, ok := v.globalRegistry.Types[schemaRef]
		if !ok {
			return fmt.Errorf("cti schema %s not found", schemaRef)
		}
		if _, err = refObject.GetSchemaByAttributeSelectorInChain(attributeSelector); err != nil {
			return fmt.Errorf("cti schema %s does not contain attribute %s: %w", schemaRef, attributeSelector, err)
		}
		// if err := refObject.IsAccessibleBy(child); err != nil {
		// 	return fmt.Errorf("cti schema %s is not accessible by %s: %w", currentRef, child.GetCTI(), err)
		// }
	}
	return nil
}

func (v *MetadataValidator) validateTypeReference(key metadata.GJsonPath, annotation *metadata.Annotations, child *metadata.EntityType, parent *metadata.EntityType) error {
	currentRefs := annotation.ReadReference()
	if len(currentRefs) == 0 {
		return nil
	}
	if parent != nil {
		parentAnnotations := parent.FindAnnotationsByPredicateInChain(key, func(a *metadata.Annotations) bool {
			return a.Reference != nil
		})
		if parentAnnotations != nil {
			parentRefs := parentAnnotations.ReadReference()
			err := func() error {
				if len(parentRefs) == 0 {
					return nil
				}
				if parentRefs[0] != TrueStr && currentRefs[0] == TrueStr {
					return errors.New("parent cti.reference defines a specific CTI, but child specifies true")
				}
				if parentRefs[0] != TrueStr {
					for _, currentRef := range currentRefs {
						compatible := false
						for _, parentRef := range parentRefs {
							if err := v.matchRefAgainstRef(parentRef, currentRef); err == nil {
								compatible = true
								break
							}
						}
						if !compatible {
							return fmt.Errorf("cti.reference %s does not match parent reference %s", currentRefs, parentRefs)
						}
					}
				}
				return nil
			}()
			if err != nil {
				return fmt.Errorf("%s@%s: %w", child.CTI, key, err)
			}
		}
		if currentRefs[0] == TrueStr {
			return nil
		}
	} else {
		if currentRefs[0] == TrueStr {
			return nil
		}
		for _, currentRef := range currentRefs {
			if _, err := v.getOrCacheExpression(currentRef, v.ctiParser.ParseIdentifier); err != nil {
				return fmt.Errorf("failed to parse cti.reference %s: %w", currentRef, err)
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
		return fmt.Errorf("%s instance has no values", entity.CTI)
	}

	for _, h := range v.collectInstanceHooks(entity) {
		if err := h(v, entity); err != nil {
			return err
		}
	}

	currentCti := entity.GetCTI()
	parent := entity.Parent()
	if parent == nil {
		return fmt.Errorf("%s instance has no parent type", currentCti)
	}
	// TODO: Move to entity.Validate()
	mergedSchema, err := parent.GetMergedSchema()
	if err != nil {
		return fmt.Errorf("failed to get merged schema for %s: %w", parent.CTI, err)
	}
	s, err := v.getOrCacheSchema(parent.CTI, mergedSchema, v.schemaLoaderCache)
	if err != nil {
		return fmt.Errorf("%s contains invalid schema: %w", parent.CTI, err)
	}
	if err = v.validateJSONDocument(s, gojsonschema.NewGoLoader(entity.Values)); err != nil {
		return fmt.Errorf("%s contains invalid values: %w", currentCti, err)
	}
	if parent.Annotations != nil {
		values, err := entity.GetRawValues()
		if err != nil {
			return fmt.Errorf("failed to get raw values for %s: %w", currentCti, err)
		}
		for key := range parent.Annotations {
			annotation := parent.FindAnnotationsByPredicateInChain(key, func(a *metadata.Annotations) bool {
				return a.Reference != nil
			})
			if err = v.validateValueReference(key, entity, annotation, values); err != nil {
				return fmt.Errorf("%s@%s: %w", currentCti, key, err)
			}
		}
	}
	return nil
}

func (v *MetadataValidator) validateValueReference(key metadata.GJsonPath, child metadata.Entity, annotation *metadata.Annotations, values []byte) error {
	if annotation == nil {
		return nil
	}

	refs := annotation.ReadReference()
	if len(refs) == 0 {
		return nil
	}
	if refs[0] == TrueStr {
		return nil
	}

	value := key.GetValue(values)
	for _, val := range value.Array() {
		compatible := false
		for _, ref := range refs {
			if err := v.matchCTIAgainstRef(ref, val.Str); err == nil {
				compatible = true
				break
			}
		}
		if !compatible {
			return fmt.Errorf("cti.reference %s does not match any of the values in %s", refs, child.GetCTI())
		}
		if _, ok := v.globalRegistry.Index[val.Str]; !ok {
			return fmt.Errorf("referenced entity %s not found in registry", val.Str)
		}
	}

	return nil
}

func (v *MetadataValidator) getOrCacheExpression(cti string, parseFn func(string) (cti.Expression, error)) (*cti.Expression, error) {
	if expr, ok := v.expressionsCache[cti]; ok {
		return expr, nil
	}
	expr, err := parseFn(cti)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CTI %s: %w", cti, err)
	}
	return &expr, nil
}

func (v *MetadataValidator) matchCTIAgainstRef(refCTI, compCTI string) error {
	refExpr, err := v.getOrCacheExpression(refCTI, v.ctiParser.ParseReference)
	if err != nil {
		return fmt.Errorf("failed to parse cti %s: %w", refCTI, err)
	}
	compExpr, err := v.getOrCacheExpression(compCTI, v.ctiParser.Parse)
	if err != nil {
		return fmt.Errorf("failed to parse cti %s: %w", compCTI, err)
	}
	if ok, err := refExpr.Match(*compExpr); !ok {
		if err != nil {
			return fmt.Errorf("%s doesn't match: %w", compCTI, err)
		}
		return fmt.Errorf("%s doesn't match", compCTI)
	}
	return nil
}

func (v *MetadataValidator) matchRefAgainstRef(refCTI, compCTI string) error {
	refExpr, err := v.getOrCacheExpression(refCTI, v.ctiParser.ParseReference)
	if err != nil {
		return fmt.Errorf("failed to parse cti %s: %w", refCTI, err)
	}
	compExpr, err := v.getOrCacheExpression(compCTI, v.ctiParser.ParseReference)
	if err != nil {
		return fmt.Errorf("failed to parse cti %s: %w", compCTI, err)
	}
	if ok, err := refExpr.Match(*compExpr); !ok {
		if err != nil {
			return fmt.Errorf("%s doesn't match: %w", compCTI, err)
		}
		return fmt.Errorf("%s doesn't match", compCTI)
	}
	return nil
}

func (v *MetadataValidator) getOrCacheSchema(
	cti string,
	schema *jsonschema.JSONSchemaCTI,
	storage map[string]*gojsonschema.Schema,
) (*gojsonschema.Schema, error) {
	s, ok := storage[cti]
	if ok {
		return s, nil
	}
	data := schema.Map()
	// NewRawLoader() will load provided interface{} directly without Marshal/Unmarshal.
	// We use it to validate the JSON document against the meta-schema.
	err := v.validateJSONDocument(v.metaSchema, gojsonschema.NewRawLoader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to validate JSON document for %s: %w", cti, err)
	}
	// And then to compile the schema for validation.
	// TODO: To consider on-demand compilation.
	s, err = gojsonschema.NewSchemaLoader().Compile(gojsonschema.NewRawLoader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to make schema loader for %s: %w", cti, err)
	}
	storage[cti] = s
	return s, nil
}

func (v *MetadataValidator) validateJSONDocument(s *gojsonschema.Schema, dl gojsonschema.JSONLoader) error {
	res, err := s.Validate(dl)
	if err != nil {
		return fmt.Errorf("failed to validate JSON document: %w", err)
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

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

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

func NewValidationIssueWrapped(msg string, err error, severity Severity, opts ...stacktrace.Option) *stacktrace.StackTrace {
	opts = append(opts, stacktrace.WithSeverity(stacktrace.Severity(severity)))
	return stacktrace.NewWrapped(msg, err, opts...)
}

func NewValidationIssue(msg string, severity Severity, opts ...stacktrace.Option) *stacktrace.StackTrace {
	opts = append(opts, stacktrace.WithSeverity(stacktrace.Severity(severity)))
	return stacktrace.New(msg, opts...)
}

type ValidatorOption func(*MetadataValidator) error

// Rule defines a validation rule for a specific type or instance in the CTI metadata.
type Rule[T metadata.EntityType | metadata.EntityInstance] struct {
	ID         string
	Expression string
	Hook       func(v *MetadataValidator, e *T) error
}

// WithTypeHook registers a TypeHook for a specific CTI type.
func WithTypeRule(rule Rule[metadata.EntityType]) ValidatorOption {
	return func(v *MetadataValidator) error {
		if rule.Hook == nil {
			return fmt.Errorf("rule %s does not provide hook function", rule.ID)
		}
		return v.onType(rule)
	}
}

// WithInstanceHook registers an InstanceHook for a specific CTI type.
func WithInstanceRule(rule Rule[metadata.EntityInstance]) ValidatorOption {
	return func(v *MetadataValidator) error {
		if rule.Hook == nil {
			return fmt.Errorf("rule %s does not provide hook function", rule.ID)
		}
		return v.onInstanceOfType(rule)
	}
}

type MetadataValidator struct {
	ctiParser *cti.Parser
	vendor    string
	pkg       string

	typeRules     map[string][]Rule[metadata.EntityType]
	instanceRules map[string][]Rule[metadata.EntityInstance]

	aggregateTypeRules     map[*cti.Expression][]Rule[metadata.EntityType]
	aggregateInstanceRules map[*cti.Expression][]Rule[metadata.EntityInstance]

	LocalRegistry  *registry.MetadataRegistry
	GlobalRegistry *registry.MetadataRegistry
	// CustomData is a map of custom data that can be used by hooks.
	CustomData map[string]any

	metaSchema              *gojsonschema.Schema
	schemaLoaderCache       map[string]*gojsonschema.Schema
	traitsSchemaLoaderCache map[string]*gojsonschema.Schema

	// TODO: Probably need global expressions cache.
	expressionsCache map[string]*cti.Expression
}

// New creates a new MetadataValidator instance.
func New(vendor, pkg string, gr, lr *registry.MetadataRegistry, opts ...ValidatorOption) (*MetadataValidator, error) {
	v := &MetadataValidator{
		ctiParser:      cti.NewParser(),
		GlobalRegistry: gr,
		LocalRegistry:  lr,
		vendor:         vendor,
		pkg:            pkg,

		aggregateTypeRules:     make(map[*cti.Expression][]Rule[metadata.EntityType]),
		aggregateInstanceRules: make(map[*cti.Expression][]Rule[metadata.EntityInstance]),

		typeRules:     make(map[string][]Rule[metadata.EntityType]),
		instanceRules: make(map[string][]Rule[metadata.EntityInstance]),

		CustomData: make(map[string]any),

		// NOTE: gojsonschema loads and compiles the meta-schema from the URL without caching on each validation.
		// Here we pre-compile the meta-schema from local source to avoid network calls and recompilation.
		metaSchema:              MustCompileSchema(jsonschema.MetaSchemaDraft07),
		schemaLoaderCache:       make(map[string]*gojsonschema.Schema),
		traitsSchemaLoaderCache: make(map[string]*gojsonschema.Schema),
	}

	for _, opt := range opts {
		if err := opt(v); err != nil {
			return nil, fmt.Errorf("failed to apply validator option: %w", err)
		}
	}

	if err := v.registerRules(); err != nil {
		return nil, fmt.Errorf("failed to aggregate hooks: %w", err)
	}

	return v, nil
}

// ValidateAll validates well-formedness of all metadata entities in the local metadata registry.
func (v *MetadataValidator) ValidateAll() (bool, error) {
	pass := true
	st := stacktrace.New("validation failed", stacktrace.WithType("validation"))
	for _, object := range v.LocalRegistry.Index {
		err := v.Validate(object)
		if err == nil {
			continue
		}
		_ = st.Append(err)
		if err.Severity.String() == string(SeverityError) {
			pass = false
		}
	}
	if len(st.List) > 0 {
		return pass, st
	}
	return pass, nil
}

// registerRules takes aggregated hooks for types and instances and assigns them to corresponding
// CTI types and instances by matching their CTI expressions.
func (v *MetadataValidator) registerRules() error {
	for k, typ := range v.LocalRegistry.Types {
		secondExpr, err := typ.Expression()
		if err != nil {
			return fmt.Errorf("failed to get expression for type %s: %w", typ.CTI, err)
		}
		for expr, hooks := range v.aggregateTypeRules {
			if ok, _ := expr.Match(*secondExpr); !ok {
				continue
			}
			v.typeRules[k] = append(v.typeRules[k], hooks...)
		}
	}

	for k, typ := range v.LocalRegistry.Instances {
		secondExpr, err := typ.Expression()
		if err != nil {
			return fmt.Errorf("failed to get expression for type %s: %w", typ.CTI, err)
		}
		for expr, hooks := range v.aggregateInstanceRules {
			if ok, _ := expr.Match(*secondExpr); !ok {
				continue
			}
			v.instanceRules[k] = append(v.instanceRules[k], hooks...)
		}
	}
	return nil
}

// onType registers a hook by CTI expression (i.e., "cti.vendor.pkg.entity_name.v1.0" or "cti.vendor.pkg.entity_name.*").
// Does not support CTI query expressions.
func (v *MetadataValidator) onType(rule Rule[metadata.EntityType]) error {
	expr, err := v.getOrCacheExpression(rule.Expression, v.ctiParser.ParseReference)
	if err != nil {
		return fmt.Errorf("failed to parse expression %s: %w", rule.Expression, err)
	}
	v.aggregateTypeRules[expr] = append(v.aggregateTypeRules[expr], rule)
	return nil
}

// onInstanceOfType registers a hook by CTI expression (i.e., "cti.vendor.pkg.entity_name.v1.0" or "cti.vendor.pkg.entity_name.*").
// Does not support CTI query expressions and attribute selectors.
func (v *MetadataValidator) onInstanceOfType(rule Rule[metadata.EntityInstance]) error {
	expr, err := v.getOrCacheExpression(rule.Expression, v.ctiParser.ParseReference)
	if err != nil {
		return fmt.Errorf("failed to parse expression %s: %w", rule.Expression, err)
	}
	v.aggregateInstanceRules[expr] = append(v.aggregateInstanceRules[expr], rule)
	return nil
}

// Validate validates the well-formedness of a single metadata entity (type or instance).
func (v *MetadataValidator) Validate(object metadata.Entity) *stacktrace.StackTrace {
	err := v.validateBaseProperties(object)
	if err != nil {
		return NewValidationIssueWrapped("failed to validate base properties", err, SeverityError)
	}
	switch entity := object.(type) {
	case *metadata.EntityType:
		err = v.ValidateType(entity)
	case *metadata.EntityInstance:
		err = v.ValidateInstance(entity)
	default:
		return NewValidationIssue(
			"invalid type",
			SeverityError,
			stacktrace.WithInfo("expected", "EntityType or EntityInstance"),
			stacktrace.WithInfo("got", fmt.Sprintf("%T", object)),
		)
	}
	if err != nil {
		if vErr, ok := err.(*stacktrace.StackTrace); ok {
			return vErr
		}
		return NewValidationIssueWrapped("failed to validate entity", err, SeverityError)
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
		return errors.New("entity type has no schema")
	}

	for _, rule := range v.typeRules[entity.CTI] {
		if err := rule.Hook(v, entity); err != nil {
			if vErr, ok := err.(*stacktrace.StackTrace); ok {
				return stacktrace.NewWrapped(
					"validation rule",
					vErr,
					stacktrace.WithSeverity(*vErr.Severity),
					stacktrace.WithInfo("rule", rule.ID),
				)
			}
			return fmt.Errorf("validation rule %s: %w", rule.ID, err)
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
		parentWithTraits := parent.FindEntityTypeByPredicateInChain(func(e *metadata.EntityType) bool {
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
		refObject, ok := v.GlobalRegistry.Types[schemaRef]
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
		return errors.New("entity instance has no values")
	}

	for _, rule := range v.instanceRules[entity.CTI] {
		if err := rule.Hook(v, entity); err != nil {
			if vErr, ok := err.(*stacktrace.StackTrace); ok {
				return stacktrace.NewWrapped(
					"validation rule",
					vErr,
					stacktrace.WithSeverity(*vErr.Severity),
					stacktrace.WithInfo("rule", rule.ID),
				)
			}
			return fmt.Errorf("validation rule %s: %w", rule.ID, err)
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
		if _, ok := v.GlobalRegistry.Index[val.Str]; !ok {
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

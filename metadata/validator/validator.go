package validator

import (
	"errors"
	"fmt"

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

const (
	SeverityError   stacktrace.Severity = "error"
	SeverityWarning stacktrace.Severity = "warning"
	SeverityInfo    stacktrace.Severity = "info"
)

func NewValidationIssueWrapped(msg string, err error, severity stacktrace.Severity, opts ...stacktrace.Option) *stacktrace.StackTrace {
	opts = append(opts, stacktrace.WithSeverity(severity))
	return stacktrace.NewWrapped(msg, err, opts...)
}

func NewValidationIssue(msg string, severity stacktrace.Severity, opts ...stacktrace.Option) *stacktrace.StackTrace {
	opts = append(opts, stacktrace.WithSeverity(severity))
	return stacktrace.New(msg, opts...)
}

type ValidatorOption func(*MetadataValidator) error

// Rule defines a validation function for a specific type or instance in the CTI metadata.
type Rule[T metadata.EntityType | metadata.EntityInstance] struct {
	ID             string
	Expression     string
	ValidationHook func(v *MetadataValidator, e *T, customData any) error
	CustomDataHook func(v *MetadataValidator) (any, error)
}

type TypeRule Rule[metadata.EntityType]
type InstanceRule Rule[metadata.EntityInstance]

// WithTypeRule registers a TypeRule for a specific CTI type.
func WithTypeRule(rule TypeRule) ValidatorOption {
	return func(v *MetadataValidator) error {
		return v.onType(rule)
	}
}

// WithInstanceRule registers an InstanceRule for a specific CTI instance.
func WithInstanceRule(rule InstanceRule) ValidatorOption {
	return func(v *MetadataValidator) error {
		return v.onInstanceOfType(rule)
	}
}

type MetadataValidator struct {
	ctiParser *cti.Parser
	vendor    string
	pkg       string

	registeredRules map[string]struct{}

	typeRules     map[string][]TypeRule
	instanceRules map[string][]InstanceRule

	aggregateTypeRules     map[*cti.Expression][]TypeRule
	aggregateInstanceRules map[*cti.Expression][]InstanceRule

	// LocalRegistry is a metadata storage that contains only current package types and instances.
	LocalRegistry *registry.MetadataRegistry

	// GlobalRegistry is a metadata storage that contains both current package and dependent package
	// types and instances.
	GlobalRegistry *registry.MetadataRegistry

	// customData is a map of custom data that can be used by rules.
	customData map[string]any

	// TODO: Probably need global expressions cache.
	expressionsCache map[string]*cti.Expression
}

// New creates a new MetadataValidator instance.
func New(vendor, pkg string, gr, lr *registry.MetadataRegistry, opts ...ValidatorOption) (*MetadataValidator, error) {
	if gr == nil || lr == nil {
		return nil, errors.New("global and local metadata registries must not be nil")
	}
	v := &MetadataValidator{
		ctiParser:      cti.NewParser(),
		GlobalRegistry: gr,
		LocalRegistry:  lr,
		vendor:         vendor,
		pkg:            pkg,

		registeredRules: make(map[string]struct{}),

		aggregateTypeRules:     make(map[*cti.Expression][]TypeRule),
		aggregateInstanceRules: make(map[*cti.Expression][]InstanceRule),

		typeRules:     make(map[string][]TypeRule),
		instanceRules: make(map[string][]InstanceRule),

		customData: make(map[string]any),
	}

	for _, opt := range opts {
		if err := opt(v); err != nil {
			return nil, fmt.Errorf("failed to apply validator option: %w", err)
		}
	}

	if err := v.registerRules(); err != nil {
		return nil, fmt.Errorf("failed to aggregate rules: %w", err)
	}

	return v, nil
}

// ValidateAll validates well-formedness of all metadata entities in the local metadata registry.
func (v *MetadataValidator) ValidateAll() (bool, error) {
	pass := true
	stacktraces := make([]*stacktrace.StackTrace, 0)
	for _, object := range v.LocalRegistry.Index {
		err := v.Validate(object)
		if err == nil {
			continue
		}
		stacktraces = append(stacktraces, err)
		if err.Severity.String() == string(SeverityError) {
			pass = false
		}
	}
	if len(stacktraces) > 0 {
		st := stacktrace.New("validation issues")
		st.List = stacktraces
		return pass, st
	}
	return pass, nil
}

// registerRules takes aggregated rules for types and instances and assigns them to corresponding
// CTI types and instances by matching their CTI expressions.
func (v *MetadataValidator) registerRules() error {
	for k, typ := range v.LocalRegistry.Types {
		secondExpr, err := typ.Expression()
		if err != nil {
			return fmt.Errorf("failed to get expression for type %s: %w", typ.CTI, err)
		}
		for expr, rules := range v.aggregateTypeRules {
			if ok, _ := expr.Match(*secondExpr); !ok {
				continue
			}
			v.typeRules[k] = append(v.typeRules[k], rules...)
		}
	}

	for k, typ := range v.LocalRegistry.Instances {
		secondExpr, err := typ.Expression()
		if err != nil {
			return fmt.Errorf("failed to get expression for type %s: %w", typ.CTI, err)
		}
		for expr, rules := range v.aggregateInstanceRules {
			if ok, _ := expr.Match(*secondExpr); !ok {
				continue
			}
			v.instanceRules[k] = append(v.instanceRules[k], rules...)
		}
	}
	return nil
}

// onType registers a hook by CTI expression (i.e., "cti.vendor.pkg.entity_name.v1.0" or "cti.vendor.pkg.entity_name.*").
// Does not support CTI query expressions.
func (v *MetadataValidator) onType(rule TypeRule) error {
	if rule.ValidationHook == nil {
		return fmt.Errorf("rule '%s' does not provide hook function", rule.ID)
	}

	if _, ok := v.registeredRules[rule.ID]; ok {
		return fmt.Errorf("rule '%s' is already registered", rule.ID)
	}
	v.registeredRules[rule.ID] = struct{}{}

	expr, err := v.getOrCacheExpression(rule.Expression, v.ctiParser.ParseReference)
	if err != nil {
		return fmt.Errorf("failed to parse expression %s: %w", rule.Expression, err)
	}
	if rule.CustomDataHook != nil {
		data, err := rule.CustomDataHook(v)
		if err != nil {
			return fmt.Errorf("failed to execute custom data hook for rule '%s': %w", rule.ID, err)
		}
		v.customData[rule.ID] = data
	}
	v.aggregateTypeRules[expr] = append(v.aggregateTypeRules[expr], rule)
	return nil
}

// onInstanceOfType registers a hook by CTI expression (i.e., "cti.vendor.pkg.entity_name.v1.0" or "cti.vendor.pkg.entity_name.*").
// Does not support CTI query expressions and attribute selectors.
func (v *MetadataValidator) onInstanceOfType(rule InstanceRule) error {
	if rule.ValidationHook == nil {
		return fmt.Errorf("rule '%s' does not provide hook function", rule.ID)
	}

	if _, ok := v.registeredRules[rule.ID]; ok {
		return fmt.Errorf("rule '%s' is already registered", rule.ID)
	}
	v.registeredRules[rule.ID] = struct{}{}

	expr, err := v.getOrCacheExpression(rule.Expression, v.ctiParser.ParseReference)
	if err != nil {
		return fmt.Errorf("failed to parse expression %s: %w", rule.Expression, err)
	}
	if rule.CustomDataHook != nil {
		data, err := rule.CustomDataHook(v)
		if err != nil {
			return fmt.Errorf("failed to execute custom data hook for rule '%s': %w", rule.ID, err)
		}
		v.customData[rule.ID] = data
	}
	v.aggregateInstanceRules[expr] = append(v.aggregateInstanceRules[expr], rule)
	return nil
}

// Validate validates the well-formedness of a single metadata entity (type or instance).
func (v *MetadataValidator) Validate(object metadata.Entity) *stacktrace.StackTrace {
	err := v.validateBaseProperties(object)
	if err != nil {
		return NewValidationIssueWrapped(
			"validate base properties",
			err,
			SeverityError,
			stacktrace.WithInfo("cti", object.GetCTI()),
		)
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
			stacktrace.WithInfo("cti", object.GetCTI()),
		)
	}
	if err != nil {
		wErr := NewValidationIssueWrapped(
			"validate entity",
			err,
			SeverityError,
			stacktrace.WithInfo("cti", object.GetCTI()),
		)
		if st, ok := err.(*stacktrace.StackTrace); ok {
			wErr.SetSeverity(*st.Severity)
		}
		return wErr
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
		if !object.IsChildOf(parent) {
			return fmt.Errorf("%s is not a child of %s", currentCti, parentCti)
		}
		if parent.Access.Integer() > object.GetAccess().Integer() {
			return fmt.Errorf("%s access is less restrictive than parent %s", currentCti, parentCti)
		}
		if err := parent.IsAccessibleBy(object); err != nil {
			return fmt.Errorf("%s is not accessible by %s: %w", currentCti, parentCti, err)
		}
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

	currentCti := entity.GetCTI()
	parent := entity.Parent()
	if parent != nil && parent.IsFinal() {
		return fmt.Errorf("%s is derived from final type %s", currentCti, parent.GetCTI())
	}

	if _, err := entity.GetSchemaValidator(); err != nil {
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
		s, err := parentWithTraits.GetTraitsSchemaValidator()
		if err != nil {
			return fmt.Errorf("%s traits schema is invalid: %w", parentWithTraits.CTI, err)
		}
		if err := jsonschema.ValidateWrapper(s, gojsonschema.NewGoLoader(entity.Traits)); err != nil {
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
		if _, err := entity.GetTraitsSchemaValidator(); err != nil {
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

	stacktraces := make([]*stacktrace.StackTrace, 0)
	highestSeverity := SeverityWarning
	for _, rule := range v.typeRules[entity.CTI] {
		if err := rule.ValidationHook(v, entity, v.customData[rule.ID]); err != nil {
			severity := SeverityError
			if vErr, ok := err.(*stacktrace.StackTrace); ok {
				severity = *vErr.Severity
			}
			if severity == SeverityError {
				highestSeverity = SeverityError
			}
			stacktraces = append(stacktraces, stacktrace.NewWrapped(
				"validation rule",
				err,
				stacktrace.WithSeverity(severity),
				stacktrace.WithInfo("rule", rule.ID),
			))
		}
	}
	if len(stacktraces) > 0 {
		st := stacktrace.New("custom type validation rules", stacktrace.WithSeverity(highestSeverity))
		st.List = stacktraces
		return st
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
		if err := refObject.IsAccessibleBy(child); err != nil {
			return fmt.Errorf("cti schema %s is not accessible by %s: %w", schemaRef, child.GetCTI(), err)
		}
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
			if _, err := v.getOrCacheExpression(currentRef, v.ctiParser.ParseReference); err != nil {
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

	currentCti := entity.GetCTI()
	parent := entity.Parent()
	if parent == nil {
		return fmt.Errorf("%s instance has no parent type", currentCti)
	}
	if err := parent.Validate(entity.Values); err != nil {
		return fmt.Errorf("%s failed to validate values: %w", currentCti, err)
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

	stacktraces := make([]*stacktrace.StackTrace, 0)
	highestSeverity := SeverityWarning
	for _, rule := range v.instanceRules[entity.CTI] {
		if err := rule.ValidationHook(v, entity, v.customData[rule.ID]); err != nil {
			severity := SeverityError
			if vErr, ok := err.(*stacktrace.StackTrace); ok {
				severity = *vErr.Severity
			}
			if severity == SeverityError {
				highestSeverity = SeverityError
			}
			stacktraces = append(stacktraces, stacktrace.NewWrapped(
				"validation rule",
				err,
				stacktrace.WithSeverity(severity),
				stacktrace.WithInfo("rule", rule.ID),
			))
		}
	}
	if len(stacktraces) > 0 {
		st := stacktrace.New("custom instance validation rules", stacktrace.WithSeverity(highestSeverity))
		st.List = stacktraces
		return st
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

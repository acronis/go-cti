package compatibility

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/acronis/go-cti/metadata/jsonschema"
)

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "Error"
	case SeverityWarning:
		return "Warning"
	case SeverityInfo:
		return "Info"
	default:
		return "Unknown"
	}
}

// TODO: Configurable severity levels
type Message struct {
	Severity Severity
	Message  string
}

type MessageGroup map[context][]Message

type EntityDiff struct {
	Messages []string
	Entity   metadata.Entity
}

// CompatibilityChecker is a struct that provides methods to check compatibility between two CTI packages.
// It implements the compatibility check logic for CTI entities and their properties.
// CompatibilityChecker performs a check according to full compatibility rules.
type CompatibilityChecker struct {
	NewEntities      []metadata.Entity
	RemovedEntities  []metadata.Entity
	ModifiedEntities []EntityDiff

	Messages MessageGroup
	Pass     bool
}

func NewCompatibilityChecker() *CompatibilityChecker {
	return &CompatibilityChecker{
		Messages: make(MessageGroup),
	}
}

func cloneExpressionWithDecrement(expr *cti.Expression) *cti.Expression {
	if expr == nil || expr.Head == nil {
		return nil
	}
	root := *expr
	h := *root.Head
	root.Head = &h

	n := root.Head
	for n != nil && n.Child != nil {
		val := *n.Child
		n.Child = &val
		n = n.Child
	}
	// TODO: Probably need to support jump by more than one minor version.
	n.Version.Minor.Value--
	return &root
}

func (cc *CompatibilityChecker) CheckPackagesCompatibility(oldPkg, newPkg *ctipackage.Package) error {
	if oldPkg == nil || newPkg == nil {
		return errors.New("packages cannot be nil")
	}
	if !oldPkg.Parsed || !newPkg.Parsed {
		return errors.New("packages must be parsed before checking compatibility")
	}
	if oldPkg.Index.PackageID != newPkg.Index.PackageID {
		return fmt.Errorf("package IDs do not match: `%s` -> `%s`", oldPkg.Index.PackageID, newPkg.Index.PackageID)
	}

	// Set pass to true by default
	// It will be set to false if any errors are found when checking compatibility
	cc.Pass = true

	// Check compatibility of entities that are present in both packages.
	for _, oldObject := range oldPkg.LocalRegistry.Index {
		oldCti := oldObject.GetCTI()
		newObject, ok := newPkg.LocalRegistry.Index[oldCti]
		if !ok {
			cc.RemovedEntities = append(cc.RemovedEntities, oldObject)
			continue
		}
		if err := cc.checkEntitiesCompatibility(oldObject, newObject); err != nil {
			return fmt.Errorf("failed to check compatibility of %s: %w", oldCti, err)
		}
	}

	// Check compatibility of new versions that may not present in the old package.
	for _, newObject := range newPkg.LocalRegistry.Index {
		newCti := newObject.GetCTI()
		if _, ok := oldPkg.LocalRegistry.Index[newCti]; ok {
			continue
		}
		cc.NewEntities = append(cc.NewEntities, newObject)

		currentVersion := newObject.Version()
		if !currentVersion.Minor.Valid {
			continue
		}

		expr, err := newObject.Expression()
		if err != nil {
			return fmt.Errorf("get expression for new object %s: %w", newCti, err)
		}

		clonedExpr := cloneExpressionWithDecrement(expr)
		if clonedExpr == nil {
			return fmt.Errorf("clone expression for new object %s", newCti)
		}

		previousMinorVersionObject := newPkg.LocalRegistry.Index[clonedExpr.String()]
		if previousMinorVersionObject == nil {
			continue
		}
		if err = cc.checkEntitiesCompatibility(previousMinorVersionObject, newObject); err != nil {
			return fmt.Errorf("failed to check compatibility between %s and %s: %w", previousMinorVersionObject.GetCTI(), newObject.GetCTI(), err)
		}
	}
	return nil
}

type context struct {
	OldEntity metadata.Entity
	NewEntity metadata.Entity
}

func (cc *CompatibilityChecker) checkEntitiesCompatibility(oldObject, newObject metadata.Entity) error {
	if oldObject == nil || newObject == nil {
		return errors.New("entities cannot be nil")
	}
	ctx := context{OldEntity: oldObject, NewEntity: newObject}
	switch oldEntity := oldObject.(type) {
	case *metadata.EntityType:
		newEntity, ok := newObject.(*metadata.EntityType)
		if !ok {
			return fmt.Errorf("entity %s is not a valid EntityType", oldEntity.CTI)
		}
		if err := cc.checkJsonSchemaCompatibility(ctx, oldEntity.Schema, newEntity.Schema); err != nil {
			return fmt.Errorf("failed to check schema compatibility: %w", err)
		}
		if err := cc.checkJsonSchemaCompatibility(ctx, oldEntity.TraitsSchema, newEntity.TraitsSchema); err != nil {
			return fmt.Errorf("failed to check traits schema compatibility: %w", err)
		}
		if err := cc.checkValuesCompatibility(ctx, oldEntity.Traits, newEntity.Traits, "$"); err != nil {
			return fmt.Errorf("failed to check traits compatibility: %w", err)
		}
		if err := cc.checkAnnotationsCompatibility(ctx, oldEntity.TraitsAnnotations, newEntity.TraitsAnnotations); err != nil {
			return fmt.Errorf("failed to check traits annotations compatibility: %w", err)
		}
	case *metadata.EntityInstance:
		newEntity, ok := newObject.(*metadata.EntityInstance)
		if !ok {
			return fmt.Errorf("entity %s is not a valid EntityInstance", oldEntity.CTI)
		}
		if err := cc.checkValuesCompatibility(ctx, oldEntity.Values, newEntity.Values, "$"); err != nil {
			return fmt.Errorf("failed to check values compatibility: %w", err)
		}
	default:
		return fmt.Errorf("invalid entity type: %T", oldEntity)
	}
	if err := cc.checkAnnotationsCompatibility(ctx, oldObject.GetAnnotations(), newObject.GetAnnotations()); err != nil {
		return fmt.Errorf("failed to check annotations compatibility: %w", err)
	}
	return nil
}

func (cc *CompatibilityChecker) checkAnnotationsCompatibility(ctx context, oldAnnotations, newAnnotations map[metadata.GJsonPath]*metadata.Annotations) error {
	if oldAnnotations != nil && newAnnotations == nil {
		return errors.New("new values cannot be nil if old values are not nil")
	} else if oldAnnotations == nil || newAnnotations == nil {
		return nil
	}
	for path, oldAnnotation := range oldAnnotations {
		newAnnotation, ok := newAnnotations[path]
		if !ok {
			cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("annotations key `%s` not found", path))
			continue
		}
		if oldAnnotation.Reference != newAnnotation.Reference {
			cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.reference mismatch: `%v` -> `%v`", path, oldAnnotation.Reference, newAnnotation.Reference))
		}
		if oldAnnotation.Meta != newAnnotation.Meta {
			cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.meta mismatch: `%s` -> `%s`", path, oldAnnotation.Meta, newAnnotation.Meta))
		}
		if oldAnnotation.PropertyNames != nil && newAnnotation.PropertyNames != nil {
			for key, oldValue := range oldAnnotation.PropertyNames {
				newValue, ok := newAnnotation.PropertyNames[key]
				if !ok {
					cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.propertyNames not found", key))
				} else if oldValue != newValue {
					cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.propertyNames mismatch: `%v` -> `%v`", key, oldValue, newValue))
				}
			}
		}
		if oldAnnotation.ID != nil && newAnnotation.ID != nil {
			if *oldAnnotation.ID != *newAnnotation.ID {
				cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.id mismatch: `%v` -> `%v`", path, *oldAnnotation.ID, *newAnnotation.ID))
			}
		}
		if oldAnnotation.Asset != nil && newAnnotation.Asset != nil {
			if *oldAnnotation.Asset != *newAnnotation.Asset {
				cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.asset mismatch: `%v` -> `%v`", path, *oldAnnotation.Asset, *newAnnotation.Asset))
			}
		}
		if oldAnnotation.L10N != nil && newAnnotation.L10N != nil {
			if *oldAnnotation.L10N != *newAnnotation.L10N {
				cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.l10n mismatch: `%v` -> `%v`", path, *oldAnnotation.L10N, *newAnnotation.L10N))
			}
		}
		if oldAnnotation.Overridable != nil && newAnnotation.Overridable != nil {
			if *oldAnnotation.Overridable != *newAnnotation.Overridable {
				cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.overridable mismatch: `%v` -> `%v`", path, *oldAnnotation.Overridable, *newAnnotation.Overridable))
			}
		}
		if oldAnnotation.Final != nil && newAnnotation.Final != nil {
			if *oldAnnotation.Final != *newAnnotation.Final {
				cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.final mismatch: `%v` -> `%v`", path, *oldAnnotation.Final, *newAnnotation.Final))
			}
		}
		if oldAnnotation.Schema != nil && newAnnotation.Schema != nil {
			if oldAnnotation.Schema != newAnnotation.Schema {
				cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` cti.schema mismatch: `%v` -> `%v`", path, oldAnnotation.Schema, newAnnotation.Schema))
			}
		}
	}
	return nil
}

func (cc *CompatibilityChecker) checkValuesCompatibility(ctx context, oldValues, newValues any, path string) error {
	if oldValues != nil && newValues == nil {
		return errors.New("new values cannot be nil if old values are not nil")
	} else if oldValues == nil || newValues == nil {
		return nil
	}

	switch oldVal := oldValues.(type) {
	case map[string]any:
		newVal, ok := newValues.(map[string]any)
		if !ok {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` is not an object: `%v` -> `%v`", path, oldValues, newValues))
			return nil
		}
		for key, oldValue := range oldVal {
			newValue, ok := newVal[key]
			if !ok {
				cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("key %s not found in new values for %s", key, path))
				continue
			}
			if err := cc.checkValuesCompatibility(ctx, oldValue, newValue, fmt.Sprintf("%s.%s", path, key)); err != nil {
				return fmt.Errorf("check values compatibility for key %s: %w", key, err)
			}
		}
	case []any:
		newVal, ok := newValues.([]any)
		if !ok {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` is not an array: `%v` -> `%v`", path, oldValues, newValues))
			return nil
		}
		if len(oldVal) != len(newVal) {
			cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("mismatching number of elements in %s: %d -> %d", path, len(oldVal), len(newVal)))
			return nil
		}
		for i, oldValue := range oldVal {
			newValue := newVal[i]
			if err := cc.checkValuesCompatibility(ctx, oldValue, newValue, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return fmt.Errorf("check values compatibility for index %d: %w", i, err)
			}
		}
	default:
		// TODO: Better check for primitive types
		if reflect.TypeOf(oldValues) != reflect.TypeOf(newValues) {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("value type mismatch in %s: %T -> %T", path, oldValues, newValues))
			return nil
		}
		if oldValues != newValues {
			cc.addMessage(ctx, SeverityWarning, fmt.Sprintf("`%s` has different value: `%v` -> `%v`", path, oldValues, newValues))
		}
	}
	return nil
}

func (cc *CompatibilityChecker) checkJsonSchemaCompatibility(ctx context, oldSchema, newSchema *jsonschema.JSONSchemaCTI) error {
	if oldSchema != nil && newSchema == nil {
		return errors.New("new schema cannot be nil if old schema is not nil")
	} else if oldSchema == nil || newSchema == nil {
		return nil
	}
	oldSchemaStart, _, err := oldSchema.GetRefSchema()
	if err != nil {
		cc.addMessage(ctx, SeverityError, fmt.Sprintf("failed to extract old schema definition: %v", err))
		return nil
	}
	newSchemaStart, _, err := newSchema.GetRefSchema()
	if err != nil {
		cc.addMessage(ctx, SeverityError, fmt.Sprintf("failed to extract new schema definition: %v", err))
		return nil
	}
	cc.traverseAndCheckSchemas(ctx, oldSchemaStart, newSchemaStart, "$")
	return nil
}

func (cc *CompatibilityChecker) addMessage(ctx context, severity Severity, message string) {
	if severity == SeverityError {
		cc.Pass = false
	}

	cc.Messages[ctx] = append(cc.Messages[ctx], Message{
		Severity: severity,
		Message:  message,
	})
}

// checkJsonSchemaCompatibility checks the compatibility of the changes between two JSON schemas.
// It returns an error if the schemas are not compatible.
func (cc *CompatibilityChecker) traverseAndCheckSchemas(ctx context, oldSchema, newSchema *jsonschema.JSONSchemaCTI, path string) {
	oldTypeName := oldSchema.Type
	if oldTypeName == "" {
		switch {
		case oldSchema.IsRef():
			oldTypeName = "ref"
		case oldSchema.IsAny():
			oldTypeName = "any"
		case oldSchema.IsAnyOf():
			oldTypeName = "anyOf"
		}
	}
	newTypeName := newSchema.Type
	if newTypeName == "" {
		switch {
		case newSchema.IsRef():
			newTypeName = "ref"
		case newSchema.IsAny():
			newTypeName = "any"
		case newSchema.IsAnyOf():
			newTypeName = "anyOf"
		}
	}

	if oldTypeName != newTypeName {
		cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` type mismatch: `%s` -> `%s`", path, oldTypeName, newTypeName))
		return
	}

	if oldSchema.Enum != nil && newSchema.Enum == nil {
		cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'enum' field", path))
	} else if oldSchema.Enum != nil && newSchema.Enum != nil {
		newEnumSet := ToSet(newSchema.Enum)
		invalid := false
		for _, v := range oldSchema.Enum {
			if _, ok := newEnumSet[v]; !ok {
				invalid = true
				break
			}
		}
		if invalid {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different enum values: `%v` -> `%v`", path+".enum", oldSchema.Enum, newSchema.Enum))
		}
	}

	// TODO: Validate $ref?

	switch {
	case oldTypeName == "object":
		requiredSet := make(map[string]struct{})
		if oldSchema.Required != nil && newSchema.Required == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'required' field", path))
		} else if oldSchema.Required != nil && newSchema.Required != nil {
			requiredSet = ToSet(newSchema.Required)
			invalid := false
			for _, v := range oldSchema.Required {
				if _, ok := requiredSet[v]; !ok {
					invalid = true
					break
				}
			}
			if invalid {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different required properties: `%v` -> `%v`", path+".required", oldSchema.Required, newSchema.Required))
			}
		}

		if oldSchema.MaxProperties != nil && newSchema.MaxProperties == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'maxProperties' field", path))
		} else if oldSchema.MaxProperties != nil && newSchema.MaxProperties != nil {
			if *oldSchema.MaxProperties != *newSchema.MaxProperties {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different maxProperties: `%v` -> `%v`", path, *oldSchema.MaxProperties, *newSchema.MaxProperties))
			}
		}

		if oldSchema.MinProperties != nil && newSchema.MinProperties == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'minProperties' field", path))
		} else if oldSchema.MinProperties != nil && newSchema.MinProperties != nil {
			if *oldSchema.MinProperties != *newSchema.MinProperties {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different minProperties: `%v` -> `%v`", path, *oldSchema.MinProperties, *newSchema.MinProperties))
			}
		}

		if oldSchema.Properties != nil && newSchema.Properties == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'properties' field", path))
		} else if oldSchema.Properties != nil && newSchema.Properties != nil {
			for p := oldSchema.Properties.Oldest(); p != nil; p = p.Next() {
				newP, ok := newSchema.Properties.Get(p.Key)
				if _, found := requiredSet[p.Key]; !ok && !found {
					cc.addMessage(ctx, SeverityError, fmt.Sprintf("required property `%s` was removed", fmt.Sprintf("%s.%s", path, p.Key)))
					continue
				}
				cc.traverseAndCheckSchemas(ctx, p.Value, newP, fmt.Sprintf("%s.%s", path, p.Key))
			}
		}

		if oldSchema.PatternProperties != nil && newSchema.PatternProperties == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'patternProperties' field", path))
		} else if oldSchema.PatternProperties != nil && newSchema.PatternProperties != nil {
			for p := oldSchema.PatternProperties.Oldest(); p != nil; p = p.Next() {
				newP, ok := newSchema.PatternProperties.Get(p.Key)
				if !ok {
					cc.addMessage(ctx, SeverityError, fmt.Sprintf("pattern property %s was removed", fmt.Sprintf("%s.%s", path, p.Key)))
					continue
				}
				cc.traverseAndCheckSchemas(ctx, p.Value, newP, fmt.Sprintf("%s.%s", path, p.Key))
			}
		}
	case oldTypeName == "array":
		if oldSchema.Items != nil && newSchema.Items == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'items' field", path))
			return
		} else if oldSchema.Items != nil && newSchema.Items != nil {
			cc.traverseAndCheckSchemas(ctx, oldSchema.Items, newSchema.Items, fmt.Sprintf("%s.%s", path, "items"))
		}

		if oldSchema.MinItems != nil && newSchema.MinItems == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'minItems' field", path))
		} else if oldSchema.MinItems != nil && newSchema.MinItems != nil {
			if *oldSchema.MinItems != *newSchema.MinItems {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different minItems: `%v` -> `%v`", path, *oldSchema.MinItems, *newSchema.MinItems))
			}
		}

		if oldSchema.MaxItems != nil && newSchema.MaxItems == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'maxItems' field", path))
		} else if oldSchema.MaxItems != nil && newSchema.MaxItems != nil {
			if *oldSchema.MaxItems != *newSchema.MaxItems {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different maxItems: `%v` -> `%v`", path, *oldSchema.MaxItems, *newSchema.MaxItems))
			}
		}

		if oldSchema.UniqueItems != nil && newSchema.UniqueItems == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'uniqueItems' field", path))
		} else if oldSchema.UniqueItems != nil && newSchema.UniqueItems != nil {
			if *oldSchema.UniqueItems != *newSchema.UniqueItems {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different uniqueItems: `%v` -> `%v`", path, *oldSchema.UniqueItems, *newSchema.UniqueItems))
			}
		}
	case oldTypeName == "string":
		if oldSchema.Format != "" && newSchema.Format == "" {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'format' field", path))
		} else if oldSchema.Format != "" && newSchema.Format != "" {
			if oldSchema.Format != newSchema.Format {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different format: `%v` -> `%v`", path, oldSchema.Format, newSchema.Format))
			}
		}

		if oldSchema.Pattern != "" && newSchema.Pattern == "" {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'pattern' field", path))
		} else if oldSchema.Pattern != "" && newSchema.Pattern != "" {
			if oldSchema.Pattern != newSchema.Pattern {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different pattern: `%v` -> `%v`", path, oldSchema.Pattern, newSchema.Pattern))
			}
		}

		if oldSchema.MinLength != nil && newSchema.MinLength == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'minLength' field", path))
		} else if oldSchema.MinLength != nil && newSchema.MinLength != nil {
			if *oldSchema.MinLength != *newSchema.MinLength {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different minLength: `%v` -> `%v`", path, *oldSchema.MinLength, *newSchema.MinLength))
			}
		}

		if oldSchema.MaxLength != nil && newSchema.MaxLength == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'maxLength' field", path))
		} else if oldSchema.MaxLength != nil && newSchema.MaxLength != nil {
			if *oldSchema.MaxLength != *newSchema.MaxLength {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different maxLength: `%v` -> `%v`", path, *oldSchema.MaxLength, *newSchema.MaxLength))
			}
		}

	case oldTypeName == "boolean":
		// No additional checks for boolean type
	case oldTypeName == "integer" || oldTypeName == "number":
		if oldSchema.Minimum != "" && newSchema.Minimum == "" {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'minimum' field", path))
		} else if oldSchema.Minimum != "" && newSchema.Minimum != "" {
			if oldSchema.Minimum != newSchema.Minimum {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different minimum: `%v` -> `%v`", path, oldSchema.Minimum, newSchema.Minimum))
			}
		}

		if oldSchema.Maximum != "" && newSchema.Maximum == "" {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'maximum' field", path))
		} else if oldSchema.Maximum != "" && newSchema.Maximum != "" {
			if oldSchema.Maximum != newSchema.Maximum {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different maximum: `%v` -> `%v`", path, oldSchema.Maximum, newSchema.Maximum))
			}
		}

		if oldSchema.MultipleOf != "" && newSchema.MultipleOf == "" {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'multipleOf' field", path))
		} else if oldSchema.MultipleOf != "" && newSchema.MultipleOf != "" {
			if oldSchema.MultipleOf != newSchema.MultipleOf {
				cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` has different multipleOf: `%v` -> `%v`", path, oldSchema.MultipleOf, newSchema.MultipleOf))
			}
		}
	case oldTypeName == "anyOf":
		if oldSchema.AnyOf != nil && oldSchema.AnyOf == nil {
			cc.addMessage(ctx, SeverityError, fmt.Sprintf("`%s` removed 'anyOf' field", path))
		} else if oldSchema.AnyOf != nil && newSchema.AnyOf != nil {
			for i, oldMember := range oldSchema.AnyOf {
				cc.traverseAndCheckSchemas(ctx, oldMember, newSchema.AnyOf[i], fmt.Sprintf("%s.%s[%d]", path, "anyOf", i))
			}
		}
	}
}

// TODO: Maybe pass logger instead?
func (cc *CompatibilityChecker) Report() string {
	if len(cc.Messages) == 0 {
		return "No compatibility issues found."
	}
	var sb strings.Builder
	for ctx, messages := range cc.Messages {
		oldCTI := ctx.OldEntity.GetCTI()
		newCTI := ctx.NewEntity.GetCTI()
		if oldCTI == newCTI {
			sb.WriteString(fmt.Sprintf("%s\n", newCTI))
		} else {
			sb.WriteString(fmt.Sprintf("%s -> %s\n", oldCTI, newCTI))
		}
		for _, msg := range messages {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", msg.Severity, msg.Message))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

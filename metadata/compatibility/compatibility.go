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

func ToSet[K comparable](src []K) map[K]struct{} {
	var result = make(map[K]struct{})
	for _, v := range src {
		result[v] = struct{}{}
	}
	return result
}

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

type Message struct {
	Severity Severity
	Message  string
}

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

	Messages []Message
	Pass     bool
}

// ValidationSummaryTemplate returns a template for when validation fails.
// It provides a concise summary of why validation failed.
func (cc *CompatibilityChecker) ValidationSummaryTemplate() string {
	var sb strings.Builder
	if cc.Pass {
		sb.WriteString("# Compatibility Check Passed ✔️\n\n")
	} else {
		sb.WriteString("# Compatibility Check Failed ❌\n\n")
	}
	sb.WriteString("## Validation summary\n\n")

	// Count errors, warnings, and info messages
	var errorCount, warningCount, infoCount int
	for _, msg := range cc.Messages {
		switch msg.Severity {
		case SeverityError:
			errorCount++
		case SeverityWarning:
			warningCount++
		case SeverityInfo:
			infoCount++
		}
	}

	// Write summary
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Errors:** %d\n", errorCount))
	sb.WriteString(fmt.Sprintf("- **Warnings:** %d\n", warningCount))
	sb.WriteString(fmt.Sprintf("- **Info:** %d\n", infoCount))
	sb.WriteString("\n")

	// Write first few errors if any
	if errorCount > 0 {
		sb.WriteString("### Critical Issues\n\n")
		count := 0
		for _, msg := range cc.Messages {
			if msg.Severity == SeverityError {
				sb.WriteString(fmt.Sprintf("- %s\n", msg.Message))
				count++
				if count >= 5 && errorCount > 5 {
					sb.WriteString(fmt.Sprintf("- ... and %d more errors\n", errorCount-5))
					break
				}
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Please see the detailed diff report for more information.\n")

	return sb.String()
}

// DiffReportTemplate returns a detailed diff report of the differences between the two packages.
func (cc *CompatibilityChecker) DiffReportTemplate() string {
	var sb strings.Builder
	sb.WriteString("# Compatibility Diff Report\n\n")

	// Add summary section
	sb.WriteString("## Summary\n\n")

	// Count errors, warnings, and info messages
	var errorCount, warningCount, infoCount int
	for _, msg := range cc.Messages {
		switch msg.Severity {
		case SeverityError:
			errorCount++
		case SeverityWarning:
			warningCount++
		case SeverityInfo:
			infoCount++
		}
	}

	// Write summary counts
	sb.WriteString(fmt.Sprintf("- **Errors:** %d\n", errorCount))
	sb.WriteString(fmt.Sprintf("- **Warnings:** %d\n", warningCount))
	sb.WriteString(fmt.Sprintf("- **Info:** %d\n", infoCount))
	sb.WriteString("\n")

	// Add new entities section if any
	if len(cc.NewEntities) > 0 {
		sb.WriteString("## New Entities\n\n")
		for _, entity := range cc.NewEntities {
			sb.WriteString(fmt.Sprintf("- `%s`\n", entity.GetCTI()))
		}
		sb.WriteString("\n")
	}

	// Add removed entities section if any
	if len(cc.RemovedEntities) > 0 {
		sb.WriteString("## Removed Entities\n\n")
		for _, entity := range cc.RemovedEntities {
			sb.WriteString(fmt.Sprintf("- `%s`\n", entity.GetCTI()))
		}
		sb.WriteString("\n")
	}

	// Add modified entities section if any
	if len(cc.ModifiedEntities) > 0 {
		sb.WriteString("## Modified Entities\n\n")
		for _, diff := range cc.ModifiedEntities {
			sb.WriteString(fmt.Sprintf("### `%s`\n\n", diff.Entity.GetCTI()))
			for _, msg := range diff.Messages {
				sb.WriteString(fmt.Sprintf("- %s\n", msg))
			}
			sb.WriteString("\n")
		}
	}

	// Add detailed messages section
	sb.WriteString("## Detailed Messages\n\n")

	// Group messages by severity
	if errorCount > 0 {
		sb.WriteString("### Errors\n\n")
		for _, msg := range cc.Messages {
			if msg.Severity == SeverityError {
				sb.WriteString(fmt.Sprintf("- %s\n", msg.Message))
			}
		}
		sb.WriteString("\n")
	}

	if warningCount > 0 {
		sb.WriteString("### Warnings\n\n")
		for _, msg := range cc.Messages {
			if msg.Severity == SeverityWarning {
				sb.WriteString(fmt.Sprintf("- %s\n", msg.Message))
			}
		}
		sb.WriteString("\n")
	}

	if infoCount > 0 {
		sb.WriteString("### Info\n\n")
		for _, msg := range cc.Messages {
			if msg.Severity == SeverityInfo {
				sb.WriteString(fmt.Sprintf("- %s\n", msg.Message))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
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
		return fmt.Errorf("package IDs do not match: %s vs %s", oldPkg.Index.PackageID, newPkg.Index.PackageID)
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
			return fmt.Errorf("check entities compatibility: %w", err)
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
			return fmt.Errorf("check entities compatibility: %w", err)
		}
	}
	return nil
}

func (cc *CompatibilityChecker) checkEntitiesCompatibility(oldObject, newObject metadata.Entity) error {
	if oldObject == nil || newObject == nil {
		return errors.New("entities cannot be nil")
	}
	switch oldEntity := oldObject.(type) {
	case *metadata.EntityType:
		newEntity, ok := newObject.(*metadata.EntityType)
		if !ok {
			return fmt.Errorf("entity %s is not a valid EntityType", oldEntity.CTI)
		}
		if err := cc.checkJsonSchemaCompatibility(oldEntity.Schema, newEntity.Schema); err != nil {
			return fmt.Errorf("failed to check schema compatibility: %w", err)
		}
		if err := cc.checkJsonSchemaCompatibility(oldEntity.TraitsSchema, newEntity.TraitsSchema); err != nil {
			return fmt.Errorf("failed to check traits schema compatibility: %w", err)
		}
		if err := cc.checkValuesCompatibility(oldEntity.Traits, newEntity.Traits); err != nil {
			return fmt.Errorf("failed to check traits compatibility: %w", err)
		}
		if err := cc.checkAnnotationsCompatibility(oldEntity.TraitsAnnotations, newEntity.TraitsAnnotations); err != nil {
			return fmt.Errorf("failed to check traits annotations compatibility: %w", err)
		}
	case *metadata.EntityInstance:
		newEntity, ok := newObject.(*metadata.EntityInstance)
		if !ok {
			return fmt.Errorf("entity %s is not a valid EntityInstance", oldEntity.CTI)
		}
		if err := cc.checkValuesCompatibility(oldEntity.Values, newEntity.Values); err != nil {
			return fmt.Errorf("failed to check values compatibility: %w", err)
		}
	default:
		return fmt.Errorf("invalid entity type: %T", oldEntity)
	}
	if err := cc.checkAnnotationsCompatibility(oldObject.GetAnnotations(), newObject.GetAnnotations()); err != nil {
		return fmt.Errorf("failed to check annotations compatibility: %w", err)
	}
	return nil
}

func (cc *CompatibilityChecker) checkAnnotationsCompatibility(oldAnnotations, newAnnotations map[metadata.GJsonPath]*metadata.Annotations) error {
	if oldAnnotations != nil && newAnnotations == nil {
		return errors.New("new values cannot be nil if old values are not nil")
	} else if oldAnnotations == nil || newAnnotations == nil {
		return nil
	}
	for path, oldAnnotation := range oldAnnotations {
		newAnnotation, ok := newAnnotations[path]
		if !ok {
			cc.addMessage(SeverityError, fmt.Sprintf("annotation %s not found in new annotations", path))
			continue
		}
		if oldAnnotation.Reference != newAnnotation.Reference {
			cc.addMessage(SeverityError, fmt.Sprintf("annotation reference mismatch for %s: %v vs %v", path, oldAnnotation.Reference, newAnnotation.Reference))
		}
		if oldAnnotation.Meta != newAnnotation.Meta {
			cc.addMessage(SeverityError, fmt.Sprintf("annotation meta mismatch for %s: %s vs %s", path, oldAnnotation.Meta, newAnnotation.Meta))
		}
		if oldAnnotation.PropertyNames != nil && newAnnotation.PropertyNames != nil {
			for key, oldValue := range oldAnnotation.PropertyNames {
				newValue, ok := newAnnotation.PropertyNames[key]
				if !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property name %s not found in new annotation", key))
				} else if oldValue != newValue {
					cc.addMessage(SeverityError, fmt.Sprintf("property name mismatch for %s: %v vs %v", key, oldValue, newValue))
				}
			}
		}
		if oldAnnotation.ID != nil && newAnnotation.ID != nil {
			if *oldAnnotation.ID != *newAnnotation.ID {
				cc.addMessage(SeverityError, fmt.Sprintf("ID mismatch for %s: %v vs %v", path, *oldAnnotation.ID, *newAnnotation.ID))
			}
		}
		if oldAnnotation.Asset != nil && newAnnotation.Asset != nil {
			if *oldAnnotation.Asset != *newAnnotation.Asset {
				cc.addMessage(SeverityError, fmt.Sprintf("asset mismatch for %s: %v vs %v", path, *oldAnnotation.Asset, *newAnnotation.Asset))
			}
		}
		if oldAnnotation.L10N != nil && newAnnotation.L10N != nil {
			if *oldAnnotation.L10N != *newAnnotation.L10N {
				cc.addMessage(SeverityError, fmt.Sprintf("L10N mismatch for %s: %v vs %v", path, *oldAnnotation.L10N, *newAnnotation.L10N))
			}
		}
		if oldAnnotation.Overridable != nil && newAnnotation.Overridable != nil {
			if *oldAnnotation.Overridable != *newAnnotation.Overridable {
				cc.addMessage(SeverityError, fmt.Sprintf("overridable mismatch for %s: %v vs %v", path, *oldAnnotation.Overridable, *newAnnotation.Overridable))
			}
		}
		if oldAnnotation.Final != nil && newAnnotation.Final != nil {
			if *oldAnnotation.Final != *newAnnotation.Final {
				cc.addMessage(SeverityError, fmt.Sprintf("final mismatch for %s: %v vs %v", path, *oldAnnotation.Final, *newAnnotation.Final))
			}
		}
		if oldAnnotation.Schema != nil && newAnnotation.Schema != nil {
			if oldAnnotation.Schema != newAnnotation.Schema {
				cc.addMessage(SeverityError, fmt.Sprintf("schema mismatch for %s: %v vs %v", path, oldAnnotation.Schema, newAnnotation.Schema))
			}
		}
	}
	return nil
}

func (cc *CompatibilityChecker) checkValuesCompatibility(oldValues, newValues interface{}) error {
	if oldValues != nil && newValues == nil {
		return errors.New("new values cannot be nil if old values are not nil")
	} else if oldValues == nil || newValues == nil {
		return nil
	}

	switch oldVal := oldValues.(type) {
	case map[string]any:
		newVal, ok := newValues.(map[string]any)
		if !ok {
			cc.addMessage(SeverityError, fmt.Sprintf("value type mismatch for %s: %v vs %v", oldVal, oldValues, newValues))
			return nil
		}
		for key, oldValue := range oldVal {
			newValue, ok := newVal[key]
			if !ok {
				cc.addMessage(SeverityError, fmt.Sprintf("key %s not found in new values", key))
				continue
			}
			if err := cc.checkValuesCompatibility(oldValue, newValue); err != nil {
				return fmt.Errorf("check values compatibility for key %s: %w", key, err)
			}
		}
	case []any:
		newVal, ok := newValues.([]any)
		if !ok {
			cc.addMessage(SeverityError, fmt.Sprintf("value type mismatch for %s: %v vs %v", oldVal, oldValues, newValues))
			return nil
		}
		if len(oldVal) != len(newVal) {
			cc.addMessage(SeverityError, fmt.Sprintf("length mismatch for array: %d vs %d", len(oldVal), len(newVal)))
			return nil
		}
		for i, oldValue := range oldVal {
			newValue := newVal[i]
			if err := cc.checkValuesCompatibility(oldValue, newValue); err != nil {
				return fmt.Errorf("check values compatibility for index %d: %w", i, err)
			}
		}
	default:
		// TODO: Better check for primitive types
		if reflect.TypeOf(oldValues) != reflect.TypeOf(newValues) {
			cc.addMessage(SeverityError, fmt.Sprintf("value type mismatch for %v: %T vs %T", oldValues, oldValues, newValues))
			return nil
		}
		if oldValues != newValues {
			cc.addMessage(SeverityError, fmt.Sprintf("value mismatch for %v: %v vs %v", oldValues, oldValues, newValues))
		}
	}
	return nil
}

func (cc *CompatibilityChecker) checkJsonSchemaCompatibility(oldSchema, newSchema *jsonschema.JSONSchemaCTI) error {
	if oldSchema != nil && newSchema == nil {
		return errors.New("new schema cannot be nil if old schema is not nil")
	} else if oldSchema == nil || newSchema == nil {
		return nil
	}
	oldSchemaStart, _, err := oldSchema.GetRefSchema()
	if err != nil {
		cc.addMessage(SeverityError, fmt.Sprintf("failed to extract old schema definition: %v", err))
		return nil
	}
	newSchemaStart, _, err := newSchema.GetRefSchema()
	if err != nil {
		cc.addMessage(SeverityError, fmt.Sprintf("failed to extract new schema definition: %v", err))
		return nil
	}
	cc.traverseAndCheckSchemas(oldSchemaStart, newSchemaStart, "$")
	return nil
}

func (cc *CompatibilityChecker) addMessage(severity Severity, message string) {
	if severity == SeverityError {
		cc.Pass = false
	}

	cc.Messages = append(cc.Messages, Message{
		Severity: severity,
		Message:  message,
	})
}

// checkJsonSchemaCompatibility checks the compatibility of the changes between two JSON schemas.
// It returns an error if the schemas are not compatible.
func (cc *CompatibilityChecker) traverseAndCheckSchemas(oldSchema, newSchema *jsonschema.JSONSchemaCTI, name string) {
	if oldSchema.Type != "" && oldSchema.Type != newSchema.Type {
		cc.addMessage(SeverityError, fmt.Sprintf("property %s, type mismatch: %s vs %s", name, oldSchema.Type, newSchema.Type))
		return
	}

	if oldSchema.Enum != nil && newSchema.Enum == nil {
		cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type enum, is missing 'enum' field", name))
	} else if oldSchema.Enum != nil && newSchema.Enum != nil {
		if len(oldSchema.Enum) != len(newSchema.Enum) {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type enum, has different number of values: %d vs %d", name, len(oldSchema.Enum), len(newSchema.Enum)))
			return
		}
		newEnumSet := ToSet(newSchema.Enum)
		invalid := false
		for _, v := range oldSchema.Enum {
			if _, ok := newEnumSet[v]; !ok {
				invalid = true
				break
			}
		}
		if invalid {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type enum, has different values: %v vs %v", name, oldSchema.Enum, newSchema.Enum))
		}
	}

	// TODO: Validate $ref

	switch {
	case oldSchema.Type == "object":
		if oldSchema.Required != nil && newSchema.Required == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'required' field", name))
		} else if oldSchema.Required != nil && newSchema.Required != nil {
			// If there are more new required properties than old required properties, it is not compatible
			if len(oldSchema.Required) != len(newSchema.Required) {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different number of required properties: %d vs %d", name, len(oldSchema.Required), len(newSchema.Required)))
				return
			}
			newRequiredSet := ToSet(newSchema.Required)
			invalid := false
			for _, v := range oldSchema.Required {
				if _, ok := newRequiredSet[v]; !ok {
					invalid = true
					break
				}
			}
			if invalid {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different required properties: %v vs %v", name, oldSchema.Required, newSchema.Required))
			}
		}

		if oldSchema.MaxProperties != nil && newSchema.MaxProperties == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'maxProperties' field", name))
		} else if oldSchema.MaxProperties != nil && newSchema.MaxProperties != nil {
			if *oldSchema.MaxProperties != *newSchema.MaxProperties {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different maxProperties: %v vs %v", name, *oldSchema.MaxProperties, *newSchema.MaxProperties))
			}
		}

		if oldSchema.MinProperties != nil && newSchema.MinProperties == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'minProperties' field", name))
		} else if oldSchema.MinProperties != nil && newSchema.MinProperties != nil {
			if *oldSchema.MinProperties != *newSchema.MinProperties {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different minProperties: %v vs %v", name, *oldSchema.MinProperties, *newSchema.MinProperties))
			}
		}

		if oldSchema.Properties != nil && newSchema.Properties == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'properties' field", name))
		} else if oldSchema.Properties != nil && newSchema.Properties != nil {
			for p := oldSchema.Properties.Oldest(); p != nil; p = p.Next() {
				newP, ok := newSchema.Properties.Get(p.Key)
				if !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'properties' field", name))
					continue
				}
				cc.traverseAndCheckSchemas(p.Value, newP, fmt.Sprintf("%s.%s", name, p.Key))
			}
		}

		if oldSchema.PatternProperties != nil && newSchema.PatternProperties == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'patternProperties' field", name))
		} else if oldSchema.PatternProperties != nil && newSchema.PatternProperties != nil {
			for p := oldSchema.PatternProperties.Oldest(); p != nil; p = p.Next() {
				newP, ok := newSchema.PatternProperties.Get(p.Key)
				if !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'patternProperties' field", name))
					continue
				}
				cc.traverseAndCheckSchemas(p.Value, newP, fmt.Sprintf("%s.%s", name, p.Key))
			}
		}
	case oldSchema.Type == "array":
		if oldSchema.Items != nil && newSchema.Items == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'items' field", name))
			return
		} else if oldSchema.Items != nil && newSchema.Items != nil {
			cc.traverseAndCheckSchemas(oldSchema.Items, newSchema.Items, fmt.Sprintf("%s.%s", name, "items"))
		}

		if oldSchema.MinItems != nil && newSchema.MinItems == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'minItems' field", name))
		} else if oldSchema.MinItems != nil && newSchema.MinItems != nil {
			if *oldSchema.MinItems != *newSchema.MinItems {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different minItems: %v vs %v", name, *oldSchema.MinItems, *newSchema.MinItems))
			}
		}

		if oldSchema.MaxItems != nil && newSchema.MaxItems == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'maxItems' field", name))
		} else if oldSchema.MaxItems != nil && newSchema.MaxItems != nil {
			if *oldSchema.MaxItems != *newSchema.MaxItems {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different maxItems: %v vs %v", name, *oldSchema.MaxItems, *newSchema.MaxItems))
			}
		}

		if oldSchema.UniqueItems != nil && newSchema.UniqueItems == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'uniqueItems' field", name))
		} else if oldSchema.UniqueItems != nil && newSchema.UniqueItems != nil {
			if oldSchema.UniqueItems != newSchema.UniqueItems {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different uniqueItems: %v vs %v", name, oldSchema.UniqueItems, newSchema.UniqueItems))
			}
		}
	case oldSchema.Type == "string":
		if oldSchema.Format != "" && newSchema.Format == "" {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'format' field", name))
		} else if oldSchema.Format != "" && newSchema.Format != "" {
			if oldSchema.Format != newSchema.Format {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different formats: %v vs %v", name, oldSchema.Format, newSchema.Format))
			}
		}

		if oldSchema.Pattern != "" && newSchema.Pattern == "" {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'pattern' field", name))
		} else if oldSchema.Pattern != "" && newSchema.Pattern != "" {
			if oldSchema.Pattern != newSchema.Pattern {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different patterns: %v vs %v", name, oldSchema.Pattern, newSchema.Pattern))
			}
		}

		if oldSchema.MinLength != nil && newSchema.MinLength == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'minLength' field", name))
		} else if oldSchema.MinLength != nil && newSchema.MinLength != nil {
			if *oldSchema.MinLength != *newSchema.MinLength {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different minLength: %v vs %v", name, *oldSchema.MinLength, *newSchema.MinLength))
			}
		}

		if oldSchema.MaxLength != nil && newSchema.MaxLength == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'maxLength' field", name))
		} else if oldSchema.MaxLength != nil && newSchema.MaxLength != nil {
			if *oldSchema.MaxLength != *newSchema.MaxLength {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different maxLength: %v vs %v", name, *oldSchema.MaxLength, *newSchema.MaxLength))
			}
		}

	case oldSchema.Type == "boolean":
		// No additional checks for boolean type
	case oldSchema.Type == "integer" || oldSchema.Type == "number":
		if oldSchema.Minimum != "" && newSchema.Minimum == "" {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'minimum' field", name))
		} else if oldSchema.Minimum != "" && newSchema.Minimum != "" {
			if oldSchema.Minimum != newSchema.Minimum {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different minimum: %v vs %v", name, oldSchema.Minimum, newSchema.Minimum))
			}
		}

		if oldSchema.Maximum != "" && newSchema.Maximum == "" {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'maximum' field", name))
		} else if oldSchema.Maximum != "" && newSchema.Maximum != "" {
			if oldSchema.Maximum != newSchema.Maximum {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different maximum: %v vs %v", name, oldSchema.Maximum, newSchema.Maximum))
			}
		}

		if oldSchema.MultipleOf != "" && newSchema.MultipleOf == "" {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'multipleOf' field", name))
		} else if oldSchema.MultipleOf != "" && newSchema.MultipleOf != "" {
			if oldSchema.MultipleOf != newSchema.MultipleOf {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different multipleOf: %v vs %v", name, oldSchema.MultipleOf, newSchema.MultipleOf))
			}
		}
	case oldSchema.IsAnyOf():
		if oldSchema.AnyOf != nil && oldSchema.AnyOf == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type anyOf, is missing 'anyOf' field", name))
		} else if oldSchema.AnyOf != nil && newSchema.AnyOf != nil {
			if len(oldSchema.AnyOf) != len(newSchema.AnyOf) {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type anyOf, has different number of values: %d vs %d", name, len(oldSchema.AnyOf), len(newSchema.AnyOf)))
				return
			}
			// TODO: Make order of anyOf values not important
			for i, oldMember := range oldSchema.AnyOf {
				cc.traverseAndCheckSchemas(oldMember, newSchema.AnyOf[i], fmt.Sprintf("%s.%s", name, "anyOf"))
			}
		}
	}
}

func (cc *CompatibilityChecker) Report() (string, error) {
	if len(cc.Messages) == 0 {
		return "No compatibility issues found.", nil
	}
	var hasError bool
	var sb strings.Builder
	sb.WriteString("Compatibility Report:\n")
	for _, msg := range cc.Messages {
		line := fmt.Sprintf("[%s] %s\n", msg.Severity.String(), msg.Message)
		sb.WriteString(line)
		if msg.Severity == SeverityError {
			hasError = true
		}
	}
	report := sb.String()
	if hasError {
		return report, fmt.Errorf("compatibility errors found")
	}
	return report, nil
}

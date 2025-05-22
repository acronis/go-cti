package compatibility

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/acronis/go-cti/metadata/merger"
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
			sb.WriteString(fmt.Sprintf("- `%s`\n", entity.GetCti()))
		}
		sb.WriteString("\n")
	}

	// Add removed entities section if any
	if len(cc.RemovedEntities) > 0 {
		sb.WriteString("## Removed Entities\n\n")
		for _, entity := range cc.RemovedEntities {
			sb.WriteString(fmt.Sprintf("- `%s`\n", entity.GetCti()))
		}
		sb.WriteString("\n")
	}

	// Add modified entities section if any
	if len(cc.ModifiedEntities) > 0 {
		sb.WriteString("## Modified Entities\n\n")
		for _, diff := range cc.ModifiedEntities {
			sb.WriteString(fmt.Sprintf("### `%s`\n\n", diff.Entity.GetCti()))
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

// TODO: To think how to attach the report to PR
//   - With dedicated file
//   - As a comment
// TODO: Make separate script for checking compatibility and making and sending report in CI

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
		oldCti := oldObject.GetCti()
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
		newCti := newObject.GetCti()
		if _, ok := oldPkg.LocalRegistry.Index[newCti]; ok {
			continue
		}
		cc.NewEntities = append(cc.NewEntities, newObject)

		currentVersion := newObject.Version()
		targetMinor := currentVersion.Minor - 1
		previousMinorVersionObject := newObject.GetObjectVersion(currentVersion.Major, targetMinor)
		if previousMinorVersionObject == nil {
			continue
		}
		if err := cc.checkEntitiesCompatibility(previousMinorVersionObject, newObject); err != nil {
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
			return fmt.Errorf("entity %s is not a valid EntityType", oldEntity.Cti)
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
			return fmt.Errorf("entity %s is not a valid EntityInstance", oldEntity.Cti)
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

func (cc *CompatibilityChecker) checkAnnotationsCompatibility(oldAnnotations, newAnnotations map[metadata.GJsonPath]metadata.Annotations) error {
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

	for key, oldValue := range oldValues.(map[string]any) {
		newValue, ok := (newValues.(map[string]any))[key]
		if !ok {
			cc.addMessage(SeverityError, fmt.Sprintf("value %s not found in new values", key))
			continue
		}
		if oldValue != newValue {
			cc.addMessage(SeverityError, fmt.Sprintf("value mismatch for %s: %v vs %v", key, oldValue, newValue))
		}
	}
	return nil
}

func (cc *CompatibilityChecker) checkJsonSchemaCompatibility(oldSchema, newSchema map[string]interface{}) error {
	if oldSchema != nil && newSchema == nil {
		return errors.New("new schema cannot be nil if old schema is not nil")
	} else if oldSchema == nil || newSchema == nil {
		return nil
	}
	oldSchemaStart, _, err := merger.ExtractSchemaDefinition(oldSchema)
	if err != nil {
		cc.addMessage(SeverityError, fmt.Sprintf("failed to extract old schema definition: %v", err))
		return nil
	}
	newSchemaStart, _, err := merger.ExtractSchemaDefinition(newSchema)
	if err != nil {
		cc.addMessage(SeverityError, fmt.Sprintf("failed to extract old schema definition: %v", err))
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
func (cc *CompatibilityChecker) traverseAndCheckSchemas(oldSchema, newSchema map[string]any, name string) {
	oldTyp, oldExists := oldSchema["type"].(string)
	newTyp, _ := newSchema["type"].(string)
	if oldTyp != "" && oldTyp != newTyp {
		cc.addMessage(SeverityError, fmt.Sprintf("property %s, type mismatch: %s vs %s", name, oldTyp, newTyp))
		return
	}

	if oldSchema["enum"] != nil && newSchema["enum"] == nil {
		cc.addMessage(SeverityError, fmt.Sprintf("property %s, type mismatch: %s vs %s", name, oldTyp, newTyp))
	} else if oldSchema["enum"] != nil && newSchema["enum"] != nil {
		oldEnum, ok := oldSchema["enum"].([]any)
		if !ok {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type enum, is missing 'enum' field", name))
			return
		}
		newEnum, ok := newSchema["enum"].([]any)
		if !ok {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type enum, is missing 'enum' field", name))
			return
		}
		if len(oldEnum) != len(newEnum) {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type enum, has different number of values: %d vs %d", name, len(oldEnum), len(newEnum)))
			return
		}
		newEnumSet := ToSet(newEnum)
		invalid := false
		for _, v := range oldEnum {
			if _, ok = newEnumSet[v]; !ok {
				invalid = true
				break
			}
		}
		if invalid {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type enum, has different values: %v vs %v", name, oldEnum, newEnum))
		}
	}

	// TODO: Validate $ref

	switch {
	case oldTyp == "object":
		if oldSchema["required"] != nil && newSchema["required"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'required' field", name))
		} else if oldSchema["required"] != nil && newSchema["required"] != nil {
			oldRequired, ok := oldSchema["required"].([]any)
			if !ok {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'required' field", name))
				return
			}
			newRequired, ok := newSchema["required"].([]any)
			if !ok {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'required' field", name))
				return
			}
			// If there are more new required properties than old required properties, it is not compatible
			if len(oldRequired) != len(newRequired) {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different number of required properties: %d vs %d", name, len(oldRequired), len(newRequired)))
				return
			}
			newRequiredSet := ToSet(newRequired)
			invalid := false
			for _, v := range oldRequired {
				if _, ok := newRequiredSet[v]; !ok {
					invalid = true
					break
				}
			}
			if invalid {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different required properties: %v vs %v", name, oldRequired, newRequired))
			}
		}

		if oldSchema["maxProperties"] != nil && newSchema["maxProperties"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'maxProperties' field", name))
		} else if oldSchema["maxProperties"] != nil && newSchema["maxProperties"] != nil {
			oldMaxProperties, ok := oldSchema["maxProperties"].(uint64)
			if !ok {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'maxProperties' field", name))
				return
			}
			newMaxProperties, ok := newSchema["maxProperties"].(uint64)
			if !ok {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'maxProperties' field", name))
				return
			}
			if oldMaxProperties != newMaxProperties {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different maxProperties: %v vs %v", name, oldMaxProperties, newMaxProperties))
			}
		}

		if oldSchema["minProperties"] != nil && newSchema["minProperties"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'minProperties' field", name))
		} else if oldSchema["minProperties"] != nil && newSchema["minProperties"] != nil {
			oldMinProperties, ok := oldSchema["minProperties"].(uint64)
			if !ok {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'minProperties' field", name))
				return
			}
			newMinProperties, ok := newSchema["minProperties"].(uint64)
			if !ok {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'minProperties' field", name))
				return
			}
			if oldMinProperties != newMinProperties {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different minProperties: %v vs %v", name, oldMinProperties, newMinProperties))
			}
		}

		// Properties of type "object" must have a "p" attribute
		oldProperties, oldPropertiesOk := oldSchema["properties"].(map[string]any)
		newProperties, newPropertiesOk := newSchema["properties"].(map[string]any)
		if oldPropertiesOk && !newPropertiesOk {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'patternProperties' field", name))
		} else if oldPropertiesOk && newPropertiesOk {
			for key, p := range oldProperties {
				newP, ok := newProperties[key]
				if !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'properties' field", name))
					continue
				}
				cc.traverseAndCheckSchemas(p.(map[string]any), newP.(map[string]any), fmt.Sprintf("%s.%s", name, key))
			}
		}

		oldPatternProperties, oldPatternPropertiesOk := oldSchema["patternProperties"].(map[string]any)
		newPatternProperties, newPatternPropertiesOk := newSchema["patternProperties"].(map[string]any)
		if oldPatternPropertiesOk && !newPatternPropertiesOk {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'patternProperties' field", name))
		} else if oldPatternPropertiesOk && newPatternPropertiesOk {
			for key, p := range oldPatternProperties {
				newP, ok := newPatternProperties[key]
				if !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'patternProperties' field", name))
					continue
				}
				cc.traverseAndCheckSchemas(p.(map[string]any), newP.(map[string]any), fmt.Sprintf("%s.%s", name, key))
			}
		}
	case oldTyp == "array":
		oldItems, oldItemsOk := oldSchema["items"].(map[string]any)
		newItems, newItemsOk := newSchema["items"].(map[string]any)
		if oldItemsOk && !newItemsOk {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'items' field", name))
			return
		} else if oldItemsOk && newItemsOk {
			cc.traverseAndCheckSchemas(oldItems, newItems, fmt.Sprintf("%s.%s", name, "items"))
		}

		if oldSchema["minItems"] != nil && newSchema["minItems"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'minItems' field", name))
		} else if oldSchema["minItems"] != nil && newSchema["minItems"] != nil {
			if oldSchema["minItems"] != newSchema["minItems"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different minItems: %v vs %v", name, oldSchema["minItems"], newSchema["minItems"]))
			}
		}

		if oldSchema["maxItems"] != nil && newSchema["maxItems"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'maxItems' field", name))
		} else if oldSchema["maxItems"] != nil && newSchema["maxItems"] != nil {
			if oldSchema["maxItems"] != newSchema["maxItems"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different maxItems: %v vs %v", name, oldSchema["maxItems"], newSchema["maxItems"]))
			}
		}

		if oldSchema["uniqueItems"] != nil && newSchema["uniqueItems"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'uniqueItems' field", name))
		} else if oldSchema["uniqueItems"] != nil && newSchema["uniqueItems"] != nil {
			if oldSchema["uniqueItems"] != newSchema["uniqueItems"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different uniqueItems: %v vs %v", name, oldSchema["uniqueItems"], newSchema["uniqueItems"]))
			}
		}
	case oldTyp == "string":
		if oldSchema["format"] != nil && newSchema["format"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'format' field", name))
		} else if oldSchema["format"] != nil && newSchema["format"] != nil {
			if oldSchema["format"] != newSchema["format"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different formats: %v vs %v", name, oldSchema["format"], newSchema["format"]))
			}
		}

		if oldSchema["pattern"] != nil && newSchema["pattern"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'pattern' field", name))
		} else if oldSchema["pattern"] != nil && newSchema["pattern"] != nil {
			if oldSchema["pattern"] != newSchema["pattern"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different patterns: %v vs %v", name, oldSchema["pattern"], newSchema["pattern"]))
			}
		}

		if oldSchema["minLength"] != nil && newSchema["minLength"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'minLength' field", name))
		} else if oldSchema["minLength"] != nil && newSchema["minLength"] != nil {
			if oldSchema["minLength"] != newSchema["minLength"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different minLength: %v vs %v", name, oldSchema["minLength"], newSchema["minLength"]))
			}
		}

		if oldSchema["maxLength"] != nil && newSchema["maxLength"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'maxLength' field", name))
		} else if oldSchema["maxLength"] != nil && newSchema["maxLength"] != nil {
			if oldSchema["maxLength"] != newSchema["maxLength"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different maxLength: %v vs %v", name, oldSchema["maxLength"], newSchema["maxLength"]))
			}
		}

	case oldTyp == "boolean":
		// No additional checks for boolean type
	case oldTyp == "integer" || oldTyp == "number":
		if oldSchema["minimum"] != nil && newSchema["minimum"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'minimum' field", name))
		} else if oldSchema["minimum"] != nil && newSchema["minimum"] != nil {
			if oldSchema["minimum"] != newSchema["minimum"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different minimum: %v vs %v", name, oldSchema["minimum"], newSchema["minimum"]))
			}
		}

		if oldSchema["maximum"] != nil && newSchema["maximum"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'maximum' field", name))
		} else if oldSchema["maximum"] != nil && newSchema["maximum"] != nil {
			if oldSchema["maximum"] != newSchema["maximum"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different maximum: %v vs %v", name, oldSchema["maximum"], newSchema["maximum"]))
			}
		}

		if oldSchema["multipleOf"] != nil && newSchema["multipleOf"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'multipleOf' field", name))
		} else if oldSchema["multipleOf"] != nil && newSchema["multipleOf"] != nil {
			if oldSchema["multipleOf"] != newSchema["multipleOf"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different multipleOf: %v vs %v", name, oldSchema["multipleOf"], newSchema["multipleOf"]))
			}
		}

		if oldSchema["exclusiveMinimum"] != nil && newSchema["exclusiveMinimum"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'exclusiveMinimum' field", name))
		} else if oldSchema["exclusiveMinimum"] != nil && newSchema["exclusiveMinimum"] != nil {
			if oldSchema["exclusiveMinimum"] != newSchema["exclusiveMinimum"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different exclusiveMinimum: %v vs %v", name, oldSchema["exclusiveMinimum"], newSchema["exclusiveMinimum"]))
			}
		}

		if oldSchema["exclusiveMaximum"] != nil && newSchema["exclusiveMaximum"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'exclusiveMaximum' field", name))
		} else if oldSchema["exclusiveMaximum"] != nil && newSchema["exclusiveMaximum"] != nil {
			if oldSchema["exclusiveMaximum"] != newSchema["exclusiveMaximum"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different exclusiveMaximum: %v vs %v", name, oldSchema["exclusiveMaximum"], newSchema["exclusiveMaximum"]))
			}
		}
	case !oldExists:
		// No type may mean that this is an "anyOf"
		oldMembers, oldMembersOk := oldSchema["anyOf"].([]any)
		newMembers, newMembersOk := newSchema["anyOf"].([]any)
		if oldMembersOk && !newMembersOk {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type anyOf, is missing 'anyOf' field", name))
		} else if oldMembersOk && newMembersOk {
			if len(oldMembers) != len(newMembers) {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type anyOf, has different number of values: %d vs %d", name, len(oldMembers), len(newMembers)))
				return
			}
			// TODO: Make order of anyOf values not important
			for i, oldMember := range oldMembers {
				newMember, ok := newMembers[i].(map[string]any)
				if !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type anyOf, is missing 'anyOf' field", name))
					return
				}
				cc.traverseAndCheckSchemas(oldMember.(map[string]any), newMember, fmt.Sprintf("%s.%s", name, "anyOf"))
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

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
}

// TODO: Report formatter
// TODO: Template for "Validation failed" comment
// TODO: Template for "Diff report"
// TODO: To think how to attach the report to PR
//   - With dedicated file
//   - As a comment
// TODO: Make separate script for checking compatibility and making and sending report in CI

func (cc *CompatibilityChecker) CheckPackagesCompatibility(oldPkg, newPkg *ctipackage.Package) bool {
	if oldPkg == nil || newPkg == nil {
		cc.addMessage(SeverityError, "packages cannot be nil")
		return false
	}
	if !oldPkg.Parsed || !newPkg.Parsed {
		cc.addMessage(SeverityError, "packages must be parsed")
		return false
	}

	if oldPkg.Index.PackageID != newPkg.Index.PackageID {
		cc.addMessage(SeverityError, fmt.Sprintf("package IDs do not match: %s vs %s", oldPkg.Index.PackageID, newPkg.Index.PackageID))
		return false
	}

	// Check compatibility of entities that are present in both packages.
	for _, oldObject := range oldPkg.LocalRegistry.Index {
		oldCti := oldObject.GetCti()
		newObject, ok := newPkg.LocalRegistry.Index[oldCti]
		if !ok {
			cc.RemovedEntities = append(cc.RemovedEntities, oldObject)
			continue
		}
		cc.CheckEntitiesCompatibility(oldObject, newObject)
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
		cc.CheckEntitiesCompatibility(previousMinorVersionObject, newObject)
	}
	return len(cc.Messages) == 0
}

func (cc *CompatibilityChecker) CheckEntitiesCompatibility(oldObject, newObject metadata.Entity) bool {
	if oldObject == nil || newObject == nil {
		cc.addMessage(SeverityError, "objects cannot be nil")
		return false
	}
	switch oldEntity := oldObject.(type) {
	case *metadata.EntityType:
		newEntity, ok := newObject.(*metadata.EntityType)
		if !ok {
			cc.addMessage(SeverityError, fmt.Sprintf("entity %s is not a valid EntityType", oldEntity.Cti))
			return false
		}
		if err := cc.CheckJsonSchemaCompatibility(oldEntity.Schema, newEntity.Schema); err != nil {
			cc.addMessage(SeverityError, fmt.Sprintf("failed to check schema compatibility: %v", err))
			return false
		}
		if err := cc.CheckJsonSchemaCompatibility(oldEntity.TraitsSchema, newEntity.TraitsSchema); err != nil {
			cc.addMessage(SeverityError, fmt.Sprintf("failed to check traits schema compatibility: %v", err))
			return false
		}
		if err := cc.CheckValuesCompatibility(oldEntity.Traits, newEntity.Traits); err != nil {
			cc.addMessage(SeverityError, fmt.Sprintf("failed to check traits compatibility: %v", err))
			return false
		}
		if err := cc.CheckAnnotationsCompatibility(oldEntity.TraitsAnnotations, newEntity.TraitsAnnotations); err != nil {
			cc.addMessage(SeverityError, fmt.Sprintf("failed to check traits annotations compatibility: %v", err))
			return false
		}
	case *metadata.EntityInstance:
		newEntity, ok := newObject.(*metadata.EntityInstance)
		if !ok {
			cc.addMessage(SeverityError, fmt.Sprintf("entity %s is not a valid EntityInstance", oldEntity.Cti))
			return false
		}
		if err := cc.CheckValuesCompatibility(oldEntity.Values, newEntity.Values); err != nil {
			cc.addMessage(SeverityError, fmt.Sprintf("failed to check values compatibility: %v", err))
			return false
		}
	default:
		cc.addMessage(SeverityError, fmt.Sprintf("invalid entity type: %T", oldEntity))
		return false
	}
	if err := cc.CheckAnnotationsCompatibility(oldObject.GetAnnotations(), newObject.GetAnnotations()); err != nil {
		cc.addMessage(SeverityError, fmt.Sprintf("failed to check annotations compatibility: %v", err))
		return false
	}
	return true
}

func (cc *CompatibilityChecker) CheckAnnotationsCompatibility(oldAnnotations, newAnnotations map[metadata.GJsonPath]metadata.Annotations) error {
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
					continue
				}
				if oldValue != newValue {
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

func (cc *CompatibilityChecker) CheckValuesCompatibility(oldValues, newValues interface{}) error {
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

func (cc *CompatibilityChecker) CheckJsonSchemaCompatibility(oldSchema, newSchema map[string]interface{}) error {
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
	cc.TraverseAndCheckSchemas(oldSchemaStart, newSchemaStart, "$")
	return nil
}

func (cc *CompatibilityChecker) addMessage(severity Severity, message string) {
	cc.Messages = append(cc.Messages, Message{
		Severity: severity,
		Message:  message,
	})
}

// CheckJsonSchemaCompatibility checks the compatibility of the changes between two JSON schemas.
// It returns an error if the schemas are not compatible.
func (cc *CompatibilityChecker) TraverseAndCheckSchemas(oldSchema, newSchema map[string]any, name string) {
	oldTyp, oldExists := oldSchema["type"].(string)
	newTyp, _ := newSchema["type"].(string)
	if oldTyp != "" && oldTyp != newTyp {
		cc.addMessage(SeverityError, fmt.Sprintf("property %s, type mismatch: %s vs %s", name, oldTyp, newTyp))
		return
	}

	if oldSchema["enum"] != nil && newSchema["enum"] == nil {
		cc.addMessage(SeverityError, fmt.Sprintf("property %s, type mismatch: %s vs %s", name, oldTyp, newTyp))
		return
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
		for _, v := range oldEnum {
			if _, ok = newEnumSet[v]; !ok {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type enum, has different values: %v vs %v", name, v, newEnumSet[v]))
				return
			}
		}
	}

	// TODO: Validate $ref

	switch {
	case oldTyp == "object":
		if oldSchema["required"] != nil && newSchema["required"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'required' field", name))
			return
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
			for _, v := range oldRequired {
				if _, ok := newRequiredSet[v]; !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, has different required properties: %v vs %v", name, v, newRequiredSet[v]))
					return
				}
			}
		}

		if oldSchema["maxProperties"] != nil && newSchema["maxProperties"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'maxProperties' field", name))
			return
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
				return
			}
		}

		if oldSchema["minProperties"] != nil && newSchema["minProperties"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'minProperties' field", name))
			return
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
				return
			}
		}

		// Properties of type "object" must have a "p" attribute
		oldProperties, oldPropertiesOk := oldSchema["properties"].(map[string]any)
		newProperties, newPropertiesOk := newSchema["properties"].(map[string]any)
		if oldPropertiesOk && !newPropertiesOk {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'patternProperties' field", name))
			return
		} else if oldPropertiesOk && newPropertiesOk {
			for key, p := range oldProperties {
				newP, ok := newProperties[key]
				if !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'properties' field", name))
					return
				}
				cc.TraverseAndCheckSchemas(p.(map[string]any), newP.(map[string]any), fmt.Sprintf("%s.%s", name, key))
			}
		}

		oldPatternProperties, oldPatternPropertiesOk := oldSchema["patternProperties"].(map[string]any)
		newPatternProperties, newPatternPropertiesOk := newSchema["patternProperties"].(map[string]any)
		if oldPatternPropertiesOk && !newPatternPropertiesOk {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'patternProperties' field", name))
			return
		} else if oldPatternPropertiesOk && newPatternPropertiesOk {
			for key, p := range oldPatternProperties {
				newP, ok := newPatternProperties[key]
				if !ok {
					cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type object, is missing 'patternProperties' field", name))
					return
				}
				cc.TraverseAndCheckSchemas(p.(map[string]any), newP.(map[string]any), fmt.Sprintf("%s.%s", name, key))
			}
		}
	case oldTyp == "array":
		oldItems, oldItemsOk := oldSchema["items"].(map[string]any)
		newItems, newItemsOk := newSchema["items"].(map[string]any)
		if oldItemsOk && !newItemsOk {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'items' field", name))
			return
		} else if oldItemsOk && newItemsOk {
			cc.TraverseAndCheckSchemas(oldItems, newItems, fmt.Sprintf("%s.%s", name, "items"))
		}

		if oldSchema["minItems"] != nil && newSchema["minItems"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'minItems' field", name))
			return
		} else if oldSchema["minItems"] != nil && newSchema["minItems"] != nil {
			if oldSchema["minItems"] != newSchema["minItems"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different minItems: %v vs %v", name, oldSchema["minItems"], newSchema["minItems"]))
				return
			}
		}

		if oldSchema["maxItems"] != nil && newSchema["maxItems"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'maxItems' field", name))
			return
		} else if oldSchema["maxItems"] != nil && newSchema["maxItems"] != nil {
			if oldSchema["maxItems"] != newSchema["maxItems"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different maxItems: %v vs %v", name, oldSchema["maxItems"], newSchema["maxItems"]))
				return
			}
		}

		if oldSchema["uniqueItems"] != nil && newSchema["uniqueItems"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, is missing 'uniqueItems' field", name))
			return
		} else if oldSchema["uniqueItems"] != nil && newSchema["uniqueItems"] != nil {
			if oldSchema["uniqueItems"] != newSchema["uniqueItems"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type array, has different uniqueItems: %v vs %v", name, oldSchema["uniqueItems"], newSchema["uniqueItems"]))
				return
			}
		}
	case oldTyp == "string":
		if oldSchema["format"] != nil && newSchema["format"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'format' field", name))
			return
		} else if oldSchema["format"] != nil && newSchema["format"] != nil {
			if oldSchema["format"] != newSchema["format"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different formats: %v vs %v", name, oldSchema["format"], newSchema["format"]))
				return
			}
		}

		if oldSchema["pattern"] != nil && newSchema["pattern"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'pattern' field", name))
			return
		} else if oldSchema["pattern"] != nil && newSchema["pattern"] != nil {
			if oldSchema["pattern"] != newSchema["pattern"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different patterns: %v vs %v", name, oldSchema["pattern"], newSchema["pattern"]))
				return
			}
		}

		if oldSchema["minLength"] != nil && newSchema["minLength"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'minLength' field", name))
			return
		} else if oldSchema["minLength"] != nil && newSchema["minLength"] != nil {
			if oldSchema["minLength"] != newSchema["minLength"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different minLength: %v vs %v", name, oldSchema["minLength"], newSchema["minLength"]))
				return
			}
		}

		if oldSchema["maxLength"] != nil && newSchema["maxLength"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, is missing 'maxLength' field", name))
			return
		} else if oldSchema["maxLength"] != nil && newSchema["maxLength"] != nil {
			if oldSchema["maxLength"] != newSchema["maxLength"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type string, has different maxLength: %v vs %v", name, oldSchema["maxLength"], newSchema["maxLength"]))
				return
			}
		}

	case oldTyp == "boolean":
		// No additional checks for boolean type
	case oldTyp == "integer" || oldTyp == "number":
		if oldSchema["minimum"] != nil && newSchema["minimum"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'minimum' field", name))
			return
		} else if oldSchema["minimum"] != nil && newSchema["minimum"] != nil {
			if oldSchema["minimum"] != newSchema["minimum"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different minimum: %v vs %v", name, oldSchema["minimum"], newSchema["minimum"]))
				return
			}
		}

		if oldSchema["maximum"] != nil && newSchema["maximum"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'maximum' field", name))
			return
		} else if oldSchema["maximum"] != nil && newSchema["maximum"] != nil {
			if oldSchema["maximum"] != newSchema["maximum"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different maximum: %v vs %v", name, oldSchema["maximum"], newSchema["maximum"]))
				return
			}
		}

		if oldSchema["multipleOf"] != nil && newSchema["multipleOf"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'multipleOf' field", name))
			return
		} else if oldSchema["multipleOf"] != nil && newSchema["multipleOf"] != nil {
			if oldSchema["multipleOf"] != newSchema["multipleOf"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different multipleOf: %v vs %v", name, oldSchema["multipleOf"], newSchema["multipleOf"]))
				return
			}
		}

		if oldSchema["exclusiveMinimum"] != nil && newSchema["exclusiveMinimum"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'exclusiveMinimum' field", name))
			return
		} else if oldSchema["exclusiveMinimum"] != nil && newSchema["exclusiveMinimum"] != nil {
			if oldSchema["exclusiveMinimum"] != newSchema["exclusiveMinimum"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different exclusiveMinimum: %v vs %v", name, oldSchema["exclusiveMinimum"], newSchema["exclusiveMinimum"]))
				return
			}
		}

		if oldSchema["exclusiveMaximum"] != nil && newSchema["exclusiveMaximum"] == nil {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, is missing 'exclusiveMaximum' field", name))
			return
		} else if oldSchema["exclusiveMaximum"] != nil && newSchema["exclusiveMaximum"] != nil {
			if oldSchema["exclusiveMaximum"] != newSchema["exclusiveMaximum"] {
				cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type number, has different exclusiveMaximum: %v vs %v", name, oldSchema["exclusiveMaximum"], newSchema["exclusiveMaximum"]))
				return
			}
		}
	case !oldExists:
		// No type may mean that this is an "anyOf"
		oldMembers, oldMembersOk := oldSchema["anyOf"].([]any)
		newMembers, newMembersOk := newSchema["anyOf"].([]any)
		if oldMembersOk && !newMembersOk {
			cc.addMessage(SeverityError, fmt.Sprintf("property %s, of type anyOf, is missing 'anyOf' field", name))
			return
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
				cc.TraverseAndCheckSchemas(oldMember.(map[string]any), newMember, fmt.Sprintf("%s.%s", name, "anyOf"))
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

package merger

import (
	"errors"
	"fmt"
	"strings"
)

const (
	anyOfKey             = "anyOf"
	definitionsKey       = "definitions"
	itemsKey             = "items"
	propertiesKey        = "properties"
	patternPropertiesKey = "patternProperties"
	refKey               = "$ref"
	requiredKey          = "required"
	typeKey              = "type"
)

type merger func(source, target map[string]any) (map[string]any, error)

var errInvalidSchemaError = errors.New("invalid schema")

var (
	commonProperties = [...]string{
		"title", "description", "enum",
	}
	stringProperties = [...]string{
		"format", "pattern", "contentMediaType", "contentEncoding",
		"minLength", "maxLength",
	}
	numberProperties = [...]string{
		"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum",
		"multipleOf",
	}
	arrayProperties = [...]string{
		"minItems", "maxItems", "uniqueItems",
	}
	objectProperties = [...]string{
		"minProperties", "maxProperties", "additionalProperties",
	}
)

// MergeSchemas merges a source schema onto a target one in-place based on the rules of inheritance.
// Source is parent, target is child.
// Make a copy of the target schema if necessary.
func MergeSchemas(source, target map[string]any) (map[string]any, error) {
	mergedSchema, err := mergeSchemas(source, target)
	if err != nil {
		return nil, fmt.Errorf("failed to merge schemas: %w", err)
	}

	return mergedSchema, nil
}

func isCompatibleType(source, target map[string]any) bool {
	// If source is an "any" type, is "ref", or either of types is "anyOf", assume compatibility.
	return isAny(source) || isAnyOf(source) || isAnyOf(target) || isRef(target) || source[typeKey] == target[typeKey]
}

func mergeSchemas(source, target map[string]any) (map[string]any, error) {
	if !isCompatibleType(source, target) {
		return nil, errors.New("attempting to merge incompatible types")
	}

	// TODO: Handle $ref properly.
	// If target is a reference, return it as is without setting common properties.
	if isRef(target) {
		return target, nil
	}

	for _, key := range commonProperties {
		if target[key] == nil && source[key] != nil {
			target[key] = source[key]
		}
	}

	var mergerFn merger
	// Check for special cases first.
	// TODO: Need to consider "oneOf" and "allOf".
	switch {
	case isAny(source):
		// If source is an "any" type, return target as is since it always fully overrides "any" type.
		return target, nil
	case isAnyOf(source):
		mergerFn = mergeSourceAnyOf
	case !isAnyOf(source) && isAnyOf(target):
		mergerFn = mergeTargetAnyOf
	default:
		// TODO: Support for the list of types
		typ, ok := source[typeKey].(string)
		if !ok {
			return nil, fmt.Errorf("source schema does not have a valid type: %v", source[typeKey])
		}
		switch typ {
		case "array":
			mergerFn = mergeArrays
		case "object":
			mergerFn = mergeObjects
		case "string":
			mergerFn = mergeString
		case "number", "integer":
			mergerFn = mergeNumeric
		case "boolean", "null":
			// Return target as is since these types do not have any properties to merge.
			return target, nil
		default:
			return nil, fmt.Errorf("unsupported type for merging: %s", typ)
		}
	}

	return mergerFn(source, target)
}

// mergeRequired merges two "required" arrays
func mergeRequired(source, target map[string]any) ([]any, error) {
	// Use maps to simulate sets
	requiredSet := make(map[any]struct{})

	// Extract source required fields
	if someRequired, ok := source[requiredKey]; ok {
		required, ok := someRequired.([]any)
		if !ok {
			return nil, fmt.Errorf("source required field is not a list: %v", someRequired)
		}
		for _, item := range required {
			requiredSet[item] = struct{}{}
		}
	}

	// Extract target required fields
	if someRequired, ok := target[requiredKey]; ok {
		required, ok := someRequired.([]any)
		if !ok {
			return nil, fmt.Errorf("target required field is not a list: %v", someRequired)
		}
		for _, item := range required {
			requiredSet[item] = struct{}{}
		}
	}

	targetRequired := make([]any, 0, len(requiredSet))
	for key := range requiredSet {
		targetRequired = append(targetRequired, key)
	}

	return targetRequired, nil
}

func mergeString(source, target map[string]any) (map[string]any, error) {
	// TODO: Proper inheritance handling
	for _, key := range stringProperties {
		if target[key] == nil && source[key] != nil {
			target[key] = source[key]
		}
	}
	return target, nil
}

func mergeNumeric(source, target map[string]any) (map[string]any, error) {
	// TODO: Proper inheritance handling
	for _, key := range numberProperties {
		if target[key] == nil && source[key] != nil {
			target[key] = source[key]
		}
	}
	return target, nil
}

func mergeArrays(source, target map[string]any) (map[string]any, error) {
	// TODO: Proper inheritance handling
	for _, key := range arrayProperties {
		if target[key] == nil && source[key] != nil {
			target[key] = source[key]
		}
	}

	if target[itemsKey] == nil && source[itemsKey] != nil {
		target[itemsKey] = source[itemsKey]
	} else if source[itemsKey] != nil {
		sourceItems, ok := source[itemsKey].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source items is not a map: %v", source[itemsKey])
		}
		targetItems, ok := target[itemsKey].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("target items is not a map: %v", target[itemsKey])
		}

		mergedItems, err := mergeSchemas(sourceItems, targetItems)
		if err != nil {
			return nil, fmt.Errorf("failed to merge items: %w", err)
		}
		target[itemsKey] = mergedItems
	}
	return target, nil
}

func mergeObjects(source, target map[string]any) (map[string]any, error) {
	// TODO: Proper inheritance handling
	for _, key := range objectProperties {
		if target[key] == nil && source[key] != nil {
			target[key] = source[key]
		}
	}

	if required, err := mergeRequired(source, target); err != nil {
		return nil, fmt.Errorf("failed to merge required fields: %w", err)
	} else if len(required) > 0 {
		target[requiredKey] = required
	}

	if target[patternPropertiesKey] == nil && source[patternPropertiesKey] != nil {
		target[patternPropertiesKey] = source[patternPropertiesKey]
	} else if source[patternPropertiesKey] != nil {
		sourcePatternProperties, ok := source[patternPropertiesKey].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source patternProperties is not a map: %v", source[patternPropertiesKey])
		}
		targetPatternProperties, ok := target[patternPropertiesKey].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("target patternProperties is not a map: %v", target[patternPropertiesKey])
		}
		for key, someSourceProperty := range sourcePatternProperties {
			sourceProperty, ok := someSourceProperty.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("source pattern property is not a map: %v", someSourceProperty)
			}
			if someTargetProperty, ok := targetPatternProperties[key]; !ok {
				targetPatternProperties[key] = sourceProperty
			} else {
				targetProperty, ok := someTargetProperty.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("target pattern property is not a map: %v", someTargetProperty)
				}
				mergedProperty, err := mergeSchemas(sourceProperty, targetProperty)
				if err != nil {
					return nil, fmt.Errorf("failed to merge pattern properties: %w", err)
				}
				targetPatternProperties[key] = mergedProperty
			}
		}
	}

	if target[propertiesKey] == nil && source[propertiesKey] != nil {
		target[propertiesKey] = source[propertiesKey]
	} else if source[propertiesKey] != nil {
		sourceProperties, ok := source[propertiesKey].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source properties is not a map: %v", source[propertiesKey])
		}
		targetProperties, ok := target[propertiesKey].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("target properties is not a map: %v", target[propertiesKey])
		}

		for key, someSourceProperty := range sourceProperties {
			sourceProperty, ok := someSourceProperty.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("source property is not a map: %v", someSourceProperty)
			}

			if someTargetProperty, ok := targetProperties[key]; !ok {
				targetProperties[key] = sourceProperty
			} else {
				targetProperty, ok := someTargetProperty.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("target property is not a map: %v", someTargetProperty)
				}
				mergedProperty, err := mergeSchemas(sourceProperty, targetProperty)
				if err != nil {
					return nil, fmt.Errorf("failed to merge properties: %w", err)
				}
				targetProperties[key] = mergedProperty
			}
		}
	}
	return target, nil
}

func mergeTargetAnyOf(source, target map[string]any) (map[string]any, error) {
	// Special case where parent is not a union but child is a union.
	targetAnyOf, ok := target[anyOfKey].([]any)
	if !ok {
		return nil, fmt.Errorf("target anyOf is not a list: %v", target[anyOfKey])
	}

	anyOfs := make([]any, 0)
	for _, someTargetMember := range targetAnyOf {
		targetMember, ok := someTargetMember.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source anyOf member is not a map: %v", someTargetMember)
		}
		// All child members must comply with the source type.
		merged, err := mergeSchemas(source, targetMember)
		if err != nil {
			return nil, fmt.Errorf("failed to merge anyOf: %w", err)
		}
		anyOfs = append(anyOfs, merged)
	}
	target = map[string]any{anyOfKey: anyOfs}

	return target, nil
}

func mergeSourceAnyOf(source, target map[string]any) (map[string]any, error) {
	sourceAnyOf, ok := source[anyOfKey].([]any)
	if !ok {
		return nil, fmt.Errorf("source anyOf is not a list: %v", source[anyOfKey])
	}

	anyOfs := make([]any, 0)
	if target[anyOfKey] == nil {
		// Child schema is not a union and specifies concrete parent type(s).
		for _, someSourceMember := range sourceAnyOf {
			sourceMember, ok := someSourceMember.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("source anyOf member is not a map: %v", someSourceMember)
			}

			if isAny(sourceMember) {
				// If source member is an "any" type, we can just return the target as is.
				return target, nil
			}

			// Copy is required to avoid modifying the child member schema since multiple parent members are merged into it.
			merged, err := mergeSchemas(sourceMember, DeepCopyMap(target))
			if err != nil {
				// TODO: Accumulate errors
				continue
			}
			anyOfs = append(anyOfs, merged)
		}
		if len(anyOfs) == 0 {
			return nil, errors.New("failed to find compatible type in union")
		}
		if len(anyOfs) == 1 {
			// If only one union member remains - simplify to target type
			return anyOfs[0].(map[string]any), nil // Type assertion is safe here since we know anyOfs are all maps.
		}
		target = map[string]any{anyOfKey: anyOfs}
	} else {
		targetAnyOf, ok := target[anyOfKey].([]any)
		if !ok {
			return nil, fmt.Errorf("target anyOf is not a list: %v", target[anyOfKey])
		}
		for _, someSourceMember := range sourceAnyOf {
			sourceMember, ok := someSourceMember.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("source anyOf member is not a map: %v", someSourceMember)
			}

			for _, someTargetMember := range targetAnyOf {
				targetMember, ok := someTargetMember.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("target anyOf member is not a map: %v", someTargetMember)
				}

				if !isCompatibleType(sourceMember, targetMember) {
					continue
				}

				// Copy is required to avoid modifying the original target member schema since parent members are merged into each child member.
				merged, err := mergeSchemas(sourceMember, DeepCopyMap(targetMember))
				if err != nil {
					return nil, fmt.Errorf("failed to merge anyOf members: %w", err)
				}
				anyOfs = append(anyOfs, merged)
			}
		}
		// In case source and target union do not intersect - return an error.
		if len(anyOfs) == 0 {
			return nil, errors.New("failed to find compatible type in union")
		}
		target[anyOfKey] = anyOfs
	}

	return target, nil
}

// getRefType extracts the type from a ref value.
// E.g.: "MarketingInfo" from "#/definitions/MarketingInfo"
func getRefType(ref string) (string, error) {
	prefix := "#/definitions/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):], nil
	}
	return "", errors.New("non-definition references are not implemented")
}

// ExtractSchemaDefinition extracts the actual schema definition from the wider structure,
// which includes $ref, $schema, etc.
func ExtractSchemaDefinition(object map[string]any) (map[string]any, string, error) {
	ref, ok := object[refKey].(string)
	if !ok {
		return nil, "", errInvalidSchemaError
	}

	refType, err := getRefType(ref)
	if err != nil {
		return nil, "", err
	}

	definitions, ok := object[definitionsKey].(map[string]any)
	if !ok {
		return nil, "", errInvalidSchemaError
	}

	schema, ok := definitions[refType].(map[string]any)
	if !ok || schema == nil {
		return nil, "", fmt.Errorf("schema does not have $ref: %s", refType)
	}

	return schema, refType, nil
}

func isRef(schema map[string]any) bool {
	// A schema is a reference if it has a $ref attribute.
	_, ok := schema[refKey]
	return ok
}

// isAnyOf tells whether the object is an anyOf property.
func isAnyOf(schema map[string]any) bool {
	_, ok := schema[anyOfKey]
	return ok && schema[typeKey] == nil
}

func isAny(obj map[string]any) bool {
	// An "any" type is one that has no type defined and is not an anyOf.
	return obj[typeKey] == nil && !isAnyOf(obj)
}

// ValidateSchemaProperty recursively runs a minimal validation on schema property, which is not
// performed by the gojsonlibrary we're currently using.
// It checks that:
// a property of type "object" has a map of properties, named "properties"
// a property of type "array" has the definition of each item as "items"
// the attribute "anyOf", in a property with no type, is a list of properties
func ValidateSchemaProperty(property map[string]any, name string) error {
	typ, exists := property["type"].(string)
	switch {
	case typ == "object":
		// Properties of type "object" must have a "p" attribute
		properties, ok := property["properties"].(map[string]any)
		if !ok {
			return fmt.Errorf("property %s, of type object, is missing 'properties' field", name)
		}
		for key, p := range properties {
			if err := ValidateSchemaProperty(p.(map[string]any), fmt.Sprintf("%s.%s", name, key)); err != nil {
				return fmt.Errorf("property %s, of type object, is invalid: %w", name, err)
			}
		}
	case typ == "array":
		// Properties of type "array" must have an "items" attribute
		p, ok := property["items"].(map[string]any)
		if !ok {
			return fmt.Errorf("property %s, of type array, is missing 'items' field", name)
		}
		if err := ValidateSchemaProperty(p, name); err != nil {
			return fmt.Errorf("property %s, of type array, is invalid: %w", name, err)
		}
	case !exists:
		// No type presumably means the p is an "anyOf"
		options, ok := property["anyOf"].([]any)
		if !ok {
			// Currently unsupported: no type, but not anyOf
			return nil
		}
		for i, p := range options {
			if err := ValidateSchemaProperty(p.(map[string]any), fmt.Sprintf("%s[%d]", name, i)); err != nil {
				return fmt.Errorf("property %s, of type anyOf, is invalid: %w", name, err)
			}
		}
	}

	return nil
}

func FixSelfReferences(schema map[string]any, sourceRefType string, refsToReplace map[string]struct{}) error {
	if ref, ok := schema[refKey].(string); ok {
		if _, ok = refsToReplace[ref]; ok {
			schema[refKey] = sourceRefType
		}
	}
	switch {
	case schema[itemsKey] != nil:
		items, ok := schema[itemsKey].(map[string]any)
		if !ok {
			return fmt.Errorf("items is not a map: %v", schema[itemsKey])
		}
		if err := FixSelfReferences(items, sourceRefType, refsToReplace); err != nil {
			return fmt.Errorf("failed to fix self references in items: %w", err)
		}
	case schema[propertiesKey] != nil:
		properties, ok := schema[propertiesKey].(map[string]any)
		if !ok {
			return fmt.Errorf("properties is not a map: %v", schema[propertiesKey])
		}
		for _, property := range properties {
			v, ok := property.(map[string]any)
			if !ok {
				return fmt.Errorf("property is not a map: %v", property)
			}
			if err := FixSelfReferences(v, sourceRefType, refsToReplace); err != nil {
				return fmt.Errorf("failed to fix self references in properties: %w", err)
			}
		}
	case schema[anyOfKey] != nil:
		anyOfs, ok := schema[anyOfKey].([]any)
		if !ok {
			return fmt.Errorf("anyOf is not a list: %v", schema[anyOfKey])
		}
		for _, anyOf := range anyOfs {
			v, ok := anyOf.(map[string]any)
			if !ok {
				return fmt.Errorf("anyOf is not a map: %v", anyOf)
			}
			if err := FixSelfReferences(v, sourceRefType, refsToReplace); err != nil {
				return fmt.Errorf("failed to fix self references in anyOf: %w", err)
			}
		}
	}
	return nil
}

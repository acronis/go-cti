package merger

import (
	"errors"
	"fmt"

	"github.com/acronis/go-cti/metadata/jsonschema"
)

// TODO: To move to jsonschema package

type merger func(source, target map[string]any) (map[string]any, error)

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
	return jsonschema.IsAny(source) || jsonschema.IsAnyOf(source) || jsonschema.IsAnyOf(target) || jsonschema.IsRef(target) || source["type"] == target["type"]
}

func mergeSchemas(source, target map[string]any) (map[string]any, error) {
	if !isCompatibleType(source, target) {
		return nil, errors.New("attempting to merge incompatible types")
	}

	// TODO: Handle $ref properly.
	// If target is a reference, return it as is without setting common properties.
	if jsonschema.IsRef(target) {
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
	case jsonschema.IsAny(source):
		// If source is an "any" type, return target as is since it always fully overrides "any" type.
		return target, nil
	case jsonschema.IsAnyOf(source):
		mergerFn = mergeSourceAnyOf
	case !jsonschema.IsAnyOf(source) && jsonschema.IsAnyOf(target):
		mergerFn = mergeTargetAnyOf
	default:
		// TODO: Support for the list of types
		typ, ok := source["type"].(string)
		if !ok {
			return nil, fmt.Errorf("source schema does not have a valid type: %v", source["type"])
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
	if someRequired, ok := source["required"]; ok {
		required, ok := someRequired.([]any)
		if !ok {
			return nil, fmt.Errorf("source required field is not a list: %v", someRequired)
		}
		for _, item := range required {
			requiredSet[item] = struct{}{}
		}
	}

	// Extract target required fields
	if someRequired, ok := target["required"]; ok {
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

	if target["items"] == nil && source["items"] != nil {
		target["items"] = source["items"]
	} else if source["items"] != nil {
		sourceItems, ok := source["items"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source items is not a map: %v", source["items"])
		}
		targetItems, ok := target["items"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("target items is not a map: %v", target["items"])
		}

		mergedItems, err := mergeSchemas(sourceItems, targetItems)
		if err != nil {
			return nil, fmt.Errorf("failed to merge items: %w", err)
		}
		target["items"] = mergedItems
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
		target["required"] = required
	}

	if target["patternProperties"] == nil && source["patternProperties"] != nil {
		target["patternProperties"] = source["patternProperties"]
	} else if source["patternProperties"] != nil {
		sourcePatternProperties, ok := source["patternProperties"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source patternProperties is not a map: %v", source["patternProperties"])
		}
		targetPatternProperties, ok := target["patternProperties"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("target patternProperties is not a map: %v", target["patternProperties"])
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

	if target["properties"] == nil && source["properties"] != nil {
		target["properties"] = source["properties"]
	} else if source["properties"] != nil {
		sourceProperties, ok := source["properties"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source properties is not a map: %v", source["properties"])
		}
		targetProperties, ok := target["properties"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("target properties is not a map: %v", target["properties"])
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
	targetAnyOf, ok := target["anyOf"].([]any)
	if !ok {
		return nil, fmt.Errorf("target anyOf is not a list: %v", target["anyOf"])
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
	target = map[string]any{"anyOf": anyOfs}

	return target, nil
}

func mergeSourceAnyOf(source, target map[string]any) (map[string]any, error) {
	sourceAnyOf, ok := source["anyOf"].([]any)
	if !ok {
		return nil, fmt.Errorf("source anyOf is not a list: %v", source["anyOf"])
	}

	anyOfs := make([]any, 0)
	if target["anyOf"] == nil {
		// Child schema is not a union and specifies concrete parent type(s).
		for _, someSourceMember := range sourceAnyOf {
			sourceMember, ok := someSourceMember.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("source anyOf member is not a map: %v", someSourceMember)
			}

			if jsonschema.IsAny(sourceMember) {
				// If source member is an "any" type, we can just return the target as is.
				return target, nil
			}

			// Copy is required to avoid modifying the child member schema since multiple parent members are merged into it.
			merged, err := mergeSchemas(sourceMember, jsonschema.DeepCopyMap(target))
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
		target = map[string]any{"anyOf": anyOfs}
	} else {
		targetAnyOf, ok := target["anyOf"].([]any)
		if !ok {
			return nil, fmt.Errorf("target anyOf is not a list: %v", target["anyOf"])
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
				merged, err := mergeSchemas(sourceMember, jsonschema.DeepCopyMap(targetMember))
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
		target["anyOf"] = anyOfs
	}

	return target, nil
}

func FixSelfReferences(schema map[string]any, sourceRefType string, refsToReplace map[string]struct{}) error {
	if ref, ok := schema["$ref"].(string); ok {
		if _, ok = refsToReplace[ref]; ok {
			schema["$ref"] = sourceRefType
		}
	}
	switch {
	case schema["items"] != nil:
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return fmt.Errorf("items is not a map: %v", schema["items"])
		}
		if err := FixSelfReferences(items, sourceRefType, refsToReplace); err != nil {
			return fmt.Errorf("failed to fix self references in items: %w", err)
		}
	case schema["properties"] != nil:
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			return fmt.Errorf("properties is not a map: %v", schema["properties"])
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
	case schema["anyOf"] != nil:
		anyOfs, ok := schema["anyOf"].([]any)
		if !ok {
			return fmt.Errorf("anyOf is not a list: %v", schema["anyOf"])
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

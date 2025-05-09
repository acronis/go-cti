package merger

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
)

const (
	anyOfKey       = "anyOf"
	definitionsKey = "definitions"
	itemsKey       = "items"
	propertiesKey  = "properties"
	refKey         = "$ref"
	requiredKey    = "required"
	typeKey        = "type"
)

type merger func(source, target map[string]any) (map[string]any, error)

var errInvalidSchemaError = errors.New("invalid schema")

var propertiesToMerge = [...]string{
	"title", "description", "default", "pattern", "format", "enum", "additionalProperties",
	"minimum", "maximum", "multipleOf", "maxLength", "minLength", "minItems", "maxItems",
	"uniqueItems", "minProperties", "maxProperties",
}

// MergeSchemas merges a source schema onto a target one, applying various validations
func MergeSchemas(source, target map[string]any) (map[string]any, error) {
	mergedSchema, err := mergeObjects(source, target)
	if err != nil {
		return nil, fmt.Errorf("failed to merge schemas: %w", err)
	}

	return mergedSchema, nil
}

func mergeObjects(source, target map[string]any) (map[string]any, error) {
	isSourceAnyOf := isAnyOf(source)
	isTargetAnyOf := isAnyOf(target)
	if isSourceAnyOf && !isTargetAnyOf {
		return nil, errors.New("cannot merge union into non-union type")
	}
	if !isSourceAnyOf && isTargetAnyOf {
		// Override the same or any type.
		var err error
		target, err = overrideUnionType(source, target)
		if err != nil {
			return nil, err
		}
		isTargetAnyOf = isAnyOf(target)
		if isTargetAnyOf {
			return nil, errors.New("cannot specialize union of union")
		}
	}

	for _, key := range propertiesToMerge {
		if source[key] != nil {
			target[key] = source[key]
		}
	}

	// Insert source type only if target is any type.
	isTargetAny := target[typeKey] == nil && !isTargetAnyOf
	if source[typeKey] != nil && isTargetAny {
		target[typeKey] = source[typeKey]
	}
	if source[typeKey] != target[typeKey] {
		return nil, errors.New("attempting to merge incompatible types")
	}

	if required, err := mergeRequired(source, target); err != nil {
		return nil, fmt.Errorf("failed to merge required fields: %w", err)
	} else if len(required) > 0 {
		target[requiredKey] = required
	}

	var mergerFn merger
	switch {
	case source[itemsKey] != nil:
		mergerFn = mergeItems
	case source[propertiesKey] != nil:
		mergerFn = mergeProperties
	case source[anyOfKey] != nil:
		mergerFn = mergeAnyOf
	default:
		// Nothing to merge
		return target, nil
	}

	return mergerFn(source, target)
}

// overrideUnionType does what?
func overrideUnionType(source, target map[string]any) (map[string]any, error) {
	for _, val := range target[anyOfKey].([]any) {
		object, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("anyOf value is not a map: %v", val)
		}
		if object[typeKey] == source[typeKey] || (object[typeKey] == nil && object[anyOfKey] == nil) {
			return object, nil
		}
	}

	return nil, errors.New("failed to find compatible type in union")
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

func mergeItems(source, target map[string]any) (map[string]any, error) {
	if target[itemsKey] == nil {
		target[itemsKey] = source[itemsKey]
	} else {
		sourceItems, ok := source[itemsKey].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source items is not a map: %v", source[itemsKey])
		}
		targetItems, ok := target[itemsKey].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("target items is not a map: %v", target[itemsKey])
		}

		mergedItems, err := mergeObjects(sourceItems, targetItems)
		if err != nil {
			return nil, fmt.Errorf("failed to merge items: %w", err)
		}
		target[itemsKey] = mergedItems
	}
	return target, nil
}

func mergeProperties(source, target map[string]any) (map[string]any, error) {
	if target[propertiesKey] == nil {
		target[propertiesKey] = source[propertiesKey]
	} else {
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
				// If property not in target, make a copy through json.Marshal/Unmarshal
				// to avoid modifying the original object.
				propertyBytes, _ := json.Marshal(sourceProperty)
				var newProperty map[string]any
				err := json.Unmarshal(propertyBytes, &newProperty)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal property: %w", err)
				}
				targetProperties[key] = newProperty
			} else {
				targetProperty, ok := someTargetProperty.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("target property is not a map: %v", someTargetProperty)
				}
				mergedProperty, err := mergeObjects(sourceProperty, targetProperty)
				if err != nil {
					return nil, fmt.Errorf("failed to merge properties: %w", err)
				}
				targetProperties[key] = mergedProperty
			}
		}
	}
	return target, nil
}

func mergeAnyOf(source, target map[string]any) (map[string]any, error) {
	if target[anyOfKey] == nil {
		target[anyOfKey] = source[anyOfKey]
	} else {
		sourceAnyOf, ok := source[anyOfKey].([]any)
		if !ok {
			return nil, fmt.Errorf("source anyOf is not a list: %v", source[anyOfKey])
		}
		targetAnyOf, ok := target[anyOfKey].([]any)
		if !ok {
			return nil, fmt.Errorf("target anyOf is not a list: %v", target[anyOfKey])
		}

		anyOfs := make([]any, 0)
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
				if sourceMember[typeKey] != targetMember[typeKey] {
					continue
				}
				merged, err := mergeObjects(sourceMember, targetMember)
				if err != nil {
					return nil, fmt.Errorf("failed to merge anyOf: %w", err)
				}
				anyOfs = append(anyOfs, merged)
			}
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

// isAnyOf tells whether the object is an anyOf property.
func isAnyOf(obj map[string]any) bool {
	_, ok := obj[anyOfKey]
	return ok
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

func fixSelfReferences(schema map[string]any, sourceRefType string, refsToReplace map[string]struct{}) error {
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
		if err := fixSelfReferences(items, sourceRefType, refsToReplace); err != nil {
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
			if err := fixSelfReferences(v, sourceRefType, refsToReplace); err != nil {
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
			if err := fixSelfReferences(v, sourceRefType, refsToReplace); err != nil {
				return fmt.Errorf("failed to fix self references in anyOf: %w", err)
			}
		}
	}
	return nil
}

func GetMergedCtiSchema(cti string, r *collector.MetadataRegistry) (map[string]any, error) {
	root := cti

	entity, ok := r.Index[root]
	if !ok {
		return nil, fmt.Errorf("failed to find cti %s", root)
	}
	var childRootSchema map[string]any
	if err := json.Unmarshal([]byte(entity.Schema), &childRootSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	childSchema, refType, err := ExtractSchemaDefinition(childRootSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema definition: %w", err)
	}

	definitions := map[string]any{}
	for k, v := range childRootSchema[definitionsKey].(map[string]any) {
		if k == refType {
			continue
		}
		definitions[k] = v
	}

	origSelfRefType := "#/definitions/" + refType
	refsToReplace := map[string]struct{}{}

	for {
		parentCti := metadata.GetParentCti(root)
		if parentCti == root {
			break
		}
		root = parentCti

		entity, ok := r.Index[parentCti]
		if !ok {
			return nil, fmt.Errorf("failed to find cti parent %s", parentCti)
		}
		var parentRootSchema map[string]any
		if err = json.Unmarshal([]byte(entity.Schema), &parentRootSchema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parent schema: %w", err)
		}
		parentSchema, parentRefType, err := ExtractSchemaDefinition(parentRootSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to extract parent schema definition: %w", err)
		}
		refsToReplace["#/definitions/"+parentRefType] = struct{}{}

		childSchema, err = MergeSchemas(childSchema, parentSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to merge schemas: %w", err)
		}

		for k, v := range parentRootSchema[definitionsKey].(map[string]any) {
			if k == parentRefType {
				continue
			}
			if definition, ok := definitions[k]; ok {
				definition, err = MergeSchemas(v.(map[string]any), definition.(map[string]any))
				if err != nil {
					return nil, fmt.Errorf("failed to merge definitions: %w", err)
				}
				definitions[k] = definition
			} else {
				definitions[k] = v
			}
		}
	}
	definitions[refType] = childSchema
	for _, someDefinition := range definitions {
		definition, ok := someDefinition.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("definition is not a map: %v", someDefinition)
		}
		if err = fixSelfReferences(definition, origSelfRefType, refsToReplace); err != nil {
			return nil, fmt.Errorf("failed to fix self references: %w", err)
		}
	}

	outSchema := map[string]any{
		"$schema":     "http://json-schema.org/draft-07/schema",
		"$ref":        origSelfRefType,
		"definitions": definitions,
	}

	return outSchema, nil
}

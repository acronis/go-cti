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

// MergeSchemas merges a source schema onto a target one, applying various validations,,
func MergeSchemas(source, target map[string]any) (map[string]any, error) {
	mergedSchema, err := mergeObjects(source, target)
	if err != nil {
		return nil, err
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
		return nil, err
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
			return nil, errInvalidSchemaError
		}
		if object[typeKey] == source[typeKey] || (object[typeKey] == nil && object[anyOfKey] == nil) {
			return object, nil
		}
	}

	return nil, errors.New("failed to find compatible type in union")
}

// mergeRequired merges two "required" arrays
func mergeRequired(source, target map[string]any) ([]string, error) {
	// Use maps to simulate sets
	requiredSet := make(map[any]struct{})

	requiredSrc, _ := source[requiredKey].([]any)
	for _, item := range requiredSrc {
		requiredSet[item] = struct{}{}
	}

	requiredTrg, _ := target[requiredKey].([]any)
	for _, item := range requiredTrg {
		requiredSet[item] = struct{}{}
	}

	targetRequired := make([]string, 0, len(requiredSet))
	for key := range requiredSet {
		targetRequired = append(targetRequired, key.(string))
	}

	return targetRequired, nil
}

func mergeItems(source, target map[string]any) (map[string]any, error) {
	if target[itemsKey] == nil {
		target[itemsKey] = source[itemsKey]
	} else {
		mergedItems, err := mergeObjects(source[itemsKey].(map[string]any), target[itemsKey].(map[string]any))
		if err != nil {
			return nil, err
		}
		target[itemsKey] = mergedItems
	}
	return target, nil
}

func mergeProperties(source, target map[string]any) (map[string]any, error) {
	if target[propertiesKey] == nil {
		target[propertiesKey] = source[propertiesKey]
	} else {
		for key, property := range source[propertiesKey].(map[string]any) {
			if targetProperty, ok := target[propertiesKey].(map[string]any)[key]; !ok {
				propertyBytes, _ := json.Marshal(property)
				var newProperty map[string]any
				err := json.Unmarshal(propertyBytes, &newProperty)
				if err != nil {
					return nil, err
				}
				target[propertiesKey].(map[string]any)[key] = newProperty
			} else {
				var err error
				mergedProperty, err := mergeObjects(property.(map[string]any), targetProperty.(map[string]any))
				if err != nil {
					return nil, err
				}
				target[propertiesKey].(map[string]any)[key] = mergedProperty
			}
		}
	}
	return target, nil
}

func mergeAnyOf(source, target map[string]any) (map[string]any, error) {
	if target[anyOfKey] == nil {
		target[anyOfKey] = source[anyOfKey]
	} else {
		anyOfs := make([]map[string]any, 0)
		for _, schema := range source[anyOfKey].([]interface{}) {
			for _, item := range target[anyOfKey].([]interface{}) {
				if item.(map[string]any)[typeKey] == schema.(map[string]any)[typeKey] {
					merged, err := mergeObjects(schema.(map[string]any), item.(map[string]any))
					if err != nil {
						return nil, err
					}
					anyOfs = append(anyOfs, merged)
				}
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
func ExtractSchemaDefinition(object map[string]any) (map[string]any, error) {
	ref, ok := object[refKey].(string)
	if !ok {
		return nil, errInvalidSchemaError
	}

	refType, err := getRefType(ref)
	if err != nil {
		return nil, err
	}

	definitions, ok := object[definitionsKey].(map[string]any)
	if !ok {
		return nil, errInvalidSchemaError
	}

	schema, ok := definitions[refType].(map[string]any)
	if !ok || schema == nil {
		return nil, fmt.Errorf("schema does not have $ref:%s", refType)
	}

	return schema, nil
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
				return err
			}
		}
	case typ == "array":
		// Properties of type "array" must have an "items" attribute
		p, ok := property["items"].(map[string]any)
		if !ok {
			return fmt.Errorf("property %s, of type array, is missing 'items' field", name)
		}
		if err := ValidateSchemaProperty(p, name); err != nil {
			return err
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
				return err
			}
		}
	}

	return nil
}

func GetMergedCtiSchema(cti string, r *collector.MetadataRegistry) (map[string]interface{}, error) {
	root := cti

	entity, ok := r.Index[root]
	if !ok {
		return nil, fmt.Errorf("failed to find cti %s", root)
	}
	var err error
	var schema map[string]any
	if err = json.Unmarshal([]byte(entity.Schema), &schema); err != nil {
		return nil, err
	}
	schema, err = ExtractSchemaDefinition(schema)
	if err != nil {
		return nil, err
	}

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
		var parentSchema map[string]any
		if err := json.Unmarshal([]byte(entity.Schema), &parentSchema); err != nil {
			return nil, err
		}
		parentSchema, err = ExtractSchemaDefinition(parentSchema)
		if err != nil {
			return nil, err
		}

		// NOTE: Resulting schema does not have ref.
		schema, err = MergeSchemas(schema, parentSchema)
		if err != nil {
			return nil, err
		}
	}
	return schema, nil
}

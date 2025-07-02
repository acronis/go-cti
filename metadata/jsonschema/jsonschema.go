package jsonschema

import (
	"errors"
	"fmt"
	"strings"
)

var errInvalidSchemaError = errors.New("invalid schema")

// GetRefType extracts the type from a ref value.
// E.g.: "MarketingInfo" from "#/definitions/MarketingInfo"
func GetRefType(ref string) (string, error) {
	prefix := "#/definitions/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):], nil
	}
	return "", errors.New("non-definition references are not implemented")
}

// ExtractSchemaDefinition extracts the actual schema definition from the wider structure,
// which includes $ref, $schema, etc.
func ExtractSchemaDefinition(object map[string]any) (map[string]any, string, error) {
	ref, ok := object["$ref"].(string)
	if !ok {
		return nil, "", errInvalidSchemaError
	}

	refType, err := GetRefType(ref)
	if err != nil {
		return nil, "", err
	}

	definitions, ok := object["definitions"].(map[string]any)
	if !ok {
		return nil, "", errInvalidSchemaError
	}

	schema, ok := definitions[refType].(map[string]any)
	if !ok || schema == nil {
		return nil, "", fmt.Errorf("schema does not have $ref: %s", refType)
	}

	return schema, refType, nil
}

func IsRef(schema map[string]any) bool {
	// A schema is a reference if it has a $ref attribute.
	_, ok := schema["$ref"]
	return ok
}

func IsAnyOf(schema map[string]any) bool {
	_, ok := schema["anyOf"]
	return ok && schema["type"] == nil
}

func IsAny(obj map[string]any) bool {
	// An "any" type is one that has no type defined and is not an anyOf.
	return obj["type"] == nil && !IsAnyOf(obj)
}

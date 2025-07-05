package jsonschema

import (
	"errors"
	"fmt"
)

type merger func(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error)

// MergeSchemas merges a source schema onto a target one in-place based on the rules of inheritance.
// Source is parent, target is child.
// Make a copy of the target schema if necessary.
func MergeSchemas(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error) {
	mergedSchema, err := mergeSchemas(source, target)
	if err != nil {
		return nil, fmt.Errorf("failed to merge schemas: %w", err)
	}

	return mergedSchema, nil
}

func mergeSchemas(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error) {
	if !target.IsCompatibleWith(source) {
		return nil, errors.New("attempting to merge incompatible types")
	}

	// TODO: Handle $ref properly.
	// If target is a reference, return it as is without setting common properties.
	if target.IsRef() {
		return target, nil
	}

	if target.JSONSchemaGeneric.Title == "" && source.JSONSchemaGeneric.Title != "" {
		target.JSONSchemaGeneric.Title = source.JSONSchemaGeneric.Title
	}
	if target.JSONSchemaGeneric.Description == "" && source.JSONSchemaGeneric.Description != "" {
		target.JSONSchemaGeneric.Description = source.JSONSchemaGeneric.Description
	}
	if target.JSONSchemaGeneric.Enum == nil && source.JSONSchemaGeneric.Enum != nil {
		target.JSONSchemaGeneric.Enum = source.JSONSchemaGeneric.Enum
	}

	var mergerFn merger
	// Check for special cases first.
	// TODO: Need to consider "oneOf" and "allOf".
	switch {
	case source.IsAny():
		// If source is an "any" type, return target as is since it always fully overrides "any" type.
		return target, nil
	case source.IsAnyOf():
		mergerFn = mergeSourceAnyOf
	case !source.IsAnyOf() && target.IsAnyOf():
		mergerFn = mergeTargetAnyOf
	default:
		switch source.Type {
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
			return nil, fmt.Errorf("unsupported type for merging: %s", source.Type)
		}
	}

	return mergerFn(source, target)
}

// mergeRequired merges two "required" arrays
func mergeRequired(source, target *JSONSchemaCTI) ([]string, error) {
	// Use maps to simulate sets
	requiredSet := make(map[string]struct{})

	// Extract source required fields
	for _, item := range source.Required {
		requiredSet[item] = struct{}{}
	}

	// Extract target required fields
	for _, item := range target.Required {
		requiredSet[item] = struct{}{}
	}

	targetRequired := make([]string, 0, len(requiredSet))
	for key := range requiredSet {
		targetRequired = append(targetRequired, key)
	}

	return targetRequired, nil
}

func mergeString(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error) {
	// TODO: Proper inheritance handling
	if target.Format == "" && source.Format != "" {
		target.Format = source.Format
	}
	if target.Pattern == "" && source.Pattern != "" {
		target.Pattern = source.Pattern
	}
	if target.ContentMediaType == "" && source.ContentMediaType != "" {
		target.ContentMediaType = source.ContentMediaType
	}
	if target.ContentEncoding == "" && source.ContentEncoding != "" {
		target.ContentEncoding = source.ContentEncoding
	}
	if target.MinLength == nil && source.MinLength != nil {
		target.MinLength = source.MinLength
	}
	if target.MaxLength == nil && source.MaxLength != nil {
		target.MaxLength = source.MaxLength
	}
	return target, nil
}

func mergeNumeric(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error) {
	if target.Minimum == "" && source.Minimum != "" {
		target.Minimum = source.Minimum
	}
	if target.Maximum == "" && source.Maximum != "" {
		target.Maximum = source.Maximum
	}
	if target.MultipleOf == "" && source.MultipleOf != "" {
		target.MultipleOf = source.MultipleOf
	}
	return target, nil
}

func mergeArrays(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error) {
	// TODO: Proper inheritance handling
	if target.MinItems == nil && source.MinItems != nil {
		target.MinItems = source.MinItems
	}
	if target.MaxItems == nil && source.MaxItems != nil {
		target.MaxItems = source.MaxItems
	}
	if target.UniqueItems == nil && source.UniqueItems != nil {
		target.UniqueItems = source.UniqueItems
	}

	if target.Items == nil && source.Items != nil {
		target.Items = source.Items
	} else if source.Items != nil {
		mergedItems, err := mergeSchemas(source.Items, target.Items)
		if err != nil {
			return nil, fmt.Errorf("failed to merge items: %w", err)
		}
		target.Items = mergedItems
	}
	return target, nil
}

func mergeObjects(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error) {
	if target.AdditionalProperties == nil && source.AdditionalProperties != nil {
		target.AdditionalProperties = source.AdditionalProperties
	}
	if target.MinProperties == nil && source.MinProperties != nil {
		target.MinProperties = source.MinProperties
	}
	if target.MaxProperties == nil && source.MaxProperties != nil {
		target.MaxProperties = source.MaxProperties
	}

	if required, err := mergeRequired(source, target); err != nil {
		return nil, fmt.Errorf("failed to merge required fields: %w", err)
	} else if len(required) > 0 {
		target.Required = required
	}

	if target.PatternProperties == nil && source.PatternProperties != nil {
		target.PatternProperties = source.PatternProperties
	} else if source.PatternProperties != nil {
		for p := source.PatternProperties.Oldest(); p != nil; p = p.Next() {
			sourceProperty := p.Value
			if targetProperty, ok := target.PatternProperties.Get(p.Key); !ok {
				target.PatternProperties.Set(p.Key, sourceProperty)
			} else {
				mergedProperty, err := mergeSchemas(sourceProperty, targetProperty)
				if err != nil {
					return nil, fmt.Errorf("failed to merge pattern properties: %w", err)
				}
				target.PatternProperties.Set(p.Key, mergedProperty)
			}
		}
	}

	if target.Properties == nil && source.Properties != nil {
		target.Properties = source.Properties
	} else if source.Properties != nil {
		for p := source.Properties.Oldest(); p != nil; p = p.Next() {
			sourceProperty := p.Value
			if targetProperty, ok := target.Properties.Get(p.Key); !ok {
				target.Properties.Set(p.Key, sourceProperty)
			} else {
				mergedProperty, err := mergeSchemas(sourceProperty, targetProperty)
				if err != nil {
					return nil, fmt.Errorf("failed to merge pattern properties: %w", err)
				}
				target.Properties.Set(p.Key, mergedProperty)
			}
		}
	}

	return target, nil
}

func mergeTargetAnyOf(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error) {
	anyOfs := make([]*JSONSchemaCTI, 0)
	for _, targetMember := range target.AnyOf {
		// All child members must comply with the source type.
		merged, err := mergeSchemas(source, targetMember)
		if err != nil {
			return nil, fmt.Errorf("failed to merge anyOf: %w", err)
		}
		anyOfs = append(anyOfs, merged)
	}
	target = &JSONSchemaCTI{JSONSchemaGeneric: &JSONSchemaGeneric{AnyOf: anyOfs}}
	return target, nil
}

func mergeSourceAnyOf(source, target *JSONSchemaCTI) (*JSONSchemaCTI, error) {
	anyOfs := make([]*JSONSchemaCTI, 0)
	if target.AnyOf == nil {
		// Child schema is not a union and specifies concrete parent type(s).
		for _, sourceMember := range source.AnyOf {
			if sourceMember.IsAny() {
				// If source member is an "any" type, we can just return the target as is.
				return target, nil
			}
			// Copy is required to avoid modifying the child member schema since multiple parent members are merged into it.
			merged, err := mergeSchemas(sourceMember, target.DeepCopy())
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
			return anyOfs[0], nil
		}
		target = &JSONSchemaCTI{JSONSchemaGeneric: &JSONSchemaGeneric{AnyOf: anyOfs}}
	} else {
		for _, sourceMember := range source.AnyOf {
			for _, targetMember := range target.AnyOf {
				if !targetMember.IsCompatibleWith(sourceMember) {
					continue
				}
				// Copy is required to avoid modifying the original target member schema since parent members are merged into each child member.
				merged, err := mergeSchemas(sourceMember, targetMember.DeepCopy())
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
		target.AnyOf = anyOfs
	}

	return target, nil
}

func FixSelfReferences(schema *JSONSchemaCTI, sourceRefType string, refsToReplace map[string]struct{}) error {
	if _, ok := refsToReplace[schema.Ref]; ok {
		schema.Ref = sourceRefType
	}
	switch {
	case schema.Items != nil:
		if err := FixSelfReferences(schema.Items, sourceRefType, refsToReplace); err != nil {
			return fmt.Errorf("failed to fix self references in items: %w", err)
		}
	case schema.Properties != nil:
		for p := schema.Properties.Oldest(); p != nil; p = p.Next() {
			if err := FixSelfReferences(p.Value, sourceRefType, refsToReplace); err != nil {
				return fmt.Errorf("failed to fix self references in properties: %w", err)
			}
		}
	case schema.AnyOf != nil:
		for _, member := range schema.AnyOf {
			if err := FixSelfReferences(member, sourceRefType, refsToReplace); err != nil {
				return fmt.Errorf("failed to fix self references in anyOf: %w", err)
			}
		}
	}
	return nil
}

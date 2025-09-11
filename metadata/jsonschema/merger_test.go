package jsonschema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestMergeRequired(t *testing.T) {
	tests := map[string]struct {
		source        *JSONSchemaCTI
		target        *JSONSchemaCTI
		expected      []string
		expectedError bool
	}{
		"simple required merge": {
			source:   &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Required: []string{"foo", "bar"}}},
			target:   &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Required: []string{"baz", "bar"}}},
			expected: []string{"foo", "bar", "baz"},
		},
		"empty source required": {
			source:   &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			target:   &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Required: []string{"baz", "bar"}}},
			expected: []string{"baz", "bar"},
		},
		"empty target required": {
			source:   &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Required: []string{"foo", "bar"}}},
			target:   &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			expected: []string{"foo", "bar"},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			required := mergeRequired(tc.source, tc.target)
			require.Equal(t, tc.expected, required)
		})
	}
}

func TestFixSelfReferences(t *testing.T) {
	tests := map[string]struct {
		schema         *JSONSchemaCTI
		sourceRefType  string
		refsToReplace  map[string]struct{}
		expectedSchema *JSONSchemaCTI
		wantErr        bool
	}{
		"simple ref replacement": {
			schema:         &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/OldRef"}},
			sourceRefType:  "#/definitions/NewRef",
			refsToReplace:  map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/NewRef"}},
			wantErr:        false,
		},
		"nested items ref replacement": {
			schema: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/OldRef"}},
			}},
			sourceRefType: "#/definitions/NewRef",
			refsToReplace: map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/NewRef"}},
			}},
			wantErr: false,
		},
		"nested properties ref replacement": {
			schema: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "field1",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/OldRef"}},
					},
				)),
			}},
			sourceRefType: "#/definitions/NewRef",
			refsToReplace: map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "field1",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/NewRef"}},
					},
				)),
			}},
			wantErr: false,
		},
		"nested anyOf ref replacement": {
			schema: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/OldRef"}},
				},
			}},
			sourceRefType: "#/definitions/NewRef",
			refsToReplace: map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/NewRef"}},
				},
			}},
			wantErr: false,
		},
		"no replacement when ref not in refsToReplace": {
			schema:         &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/AnotherRef"}},
			sourceRefType:  "#/definitions/NewRef",
			refsToReplace:  map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/AnotherRef"}},
			wantErr:        false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := FixSelfReferences(tc.schema, tc.sourceRefType, tc.refsToReplace)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, tc.schema)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedSchema, tc.schema)
			}
		})
	}
}

func TestMergeString(t *testing.T) {
	tests := map[string]struct {
		source   *JSONSchemaCTI
		target   *JSONSchemaCTI
		expected *JSONSchemaCTI
	}{
		"target missing all string properties": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Format:           "email",
				Pattern:          "^[a-z]+$",
				ContentMediaType: "text/plain",
				ContentEncoding:  "base64",
				MinLength:        &[]uint64{3}[0],
				MaxLength:        &[]uint64{10}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Format:           "email",
				Pattern:          "^[a-z]+$",
				ContentMediaType: "text/plain",
				ContentEncoding:  "base64",
				MinLength:        &[]uint64{3}[0],
				MaxLength:        &[]uint64{10}[0],
			}},
		},
		"target has some string properties, source has others": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Format:    "date-time",
				Pattern:   "[0-9]+",
				MinLength: &[]uint64{5}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Format:    "date-time",
				Pattern:   "[A-Z]+",
				MaxLength: &[]uint64{20}[0],
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Format:    "date-time",
				Pattern:   "[A-Z]+", // target value preserved
				MinLength: &[]uint64{5}[0],
				MaxLength: &[]uint64{20}[0],
			}},
		},
		"target has all string properties set": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Format:    "uri",
				Pattern:   "abc",
				MinLength: &[]uint64{1}[0],
				MaxLength: &[]uint64{100}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Format:           "hostname",
				Pattern:          "xyz",
				MinLength:        &[]uint64{10}[0],
				MaxLength:        &[]uint64{50}[0],
				ContentMediaType: "application/json",
				ContentEncoding:  "utf-8",
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Format:           "hostname",       // target value preserved
				Pattern:          "xyz",            // target value preserved
				MinLength:        &[]uint64{10}[0], // target value preserved
				MaxLength:        &[]uint64{50}[0], // target value preserved
				ContentMediaType: "application/json",
				ContentEncoding:  "utf-8",
			}},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := mergeString(tc.source, tc.target)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMergeNumeric(t *testing.T) {
	tests := map[string]struct {
		source   *JSONSchemaCTI
		target   *JSONSchemaCTI
		expected *JSONSchemaCTI
	}{
		"target missing all numeric properties": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Minimum:    json.Number("1"),
				Maximum:    json.Number("10"),
				MultipleOf: json.Number("3"),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Minimum:    json.Number("1"),
				Maximum:    json.Number("10"),
				MultipleOf: json.Number("3"),
			}},
		},
		"target has some numeric properties, source has others": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Minimum:    json.Number("5"),
				Maximum:    json.Number("20"),
				MultipleOf: json.Number("2"),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Maximum: json.Number("100"),
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Minimum:    json.Number("5"),   // source value preserved
				Maximum:    json.Number("100"), // target value preserved
				MultipleOf: json.Number("2"),
			}},
		},
		"target has all numeric properties set": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Minimum:    json.Number("0"),
				Maximum:    json.Number("50"),
				MultipleOf: json.Number("5"),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Minimum:    json.Number("-10"),
				Maximum:    json.Number("100"),
				MultipleOf: json.Number("10"),
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Minimum:    json.Number("-10"), // target value preserved
				Maximum:    json.Number("100"), // target value preserved
				MultipleOf: json.Number("10"),  // target value preserved
			}},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := mergeNumeric(tc.source, tc.target)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMergeArrays(t *testing.T) {
	tests := map[string]struct {
		source   *JSONSchemaCTI
		target   *JSONSchemaCTI
		expected *JSONSchemaCTI
		wantErr  bool
	}{
		"target missing all array properties": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinItems:    &[]uint64{1}[0],
				MaxItems:    &[]uint64{10}[0],
				UniqueItems: &[]bool{true}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinItems:    &[]uint64{1}[0],
				MaxItems:    &[]uint64{10}[0],
				UniqueItems: &[]bool{true}[0],
			}},
			wantErr: false,
		},
		"target has some array properties, source has others": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinItems: &[]uint64{2}[0],
				MaxItems: &[]uint64{5}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MaxItems:    &[]uint64{20}[0],
				UniqueItems: &[]bool{false}[0],
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinItems:    &[]uint64{2}[0],   // source value
				MaxItems:    &[]uint64{20}[0],  // target value preserved
				UniqueItems: &[]bool{false}[0], // target value preserved
			}},
			wantErr: false,
		},
		"target has all array properties set": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinItems:    &[]uint64{0}[0],
				MaxItems:    &[]uint64{50}[0],
				UniqueItems: &[]bool{true}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinItems:    &[]uint64{10}[0],
				MaxItems:    &[]uint64{100}[0],
				UniqueItems: &[]bool{false}[0],
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinItems:    &[]uint64{10}[0],  // target value preserved
				MaxItems:    &[]uint64{100}[0], // target value preserved
				UniqueItems: &[]bool{false}[0], // target value preserved
			}},
			wantErr: false,
		},
		"target missing items, source has items": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
			}},
			wantErr: false,
		},
		"both source and target have items, merge items": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{3}[0]}},
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$", MinLength: &[]uint64{3}[0]}},
			}},
			wantErr: false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := mergeArrays(tc.source, tc.target)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestMergeObjects(t *testing.T) {
	tests := map[string]struct {
		source   *JSONSchemaCTI
		target   *JSONSchemaCTI
		expected *JSONSchemaCTI
		wantErr  bool
	}{
		"target missing all object properties": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinProperties:        &[]uint64{1}[0],
				MaxProperties:        &[]uint64{10}[0],
				AdditionalProperties: &[]bool{false}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinProperties:        &[]uint64{1}[0],
				MaxProperties:        &[]uint64{10}[0],
				AdditionalProperties: &[]bool{false}[0],
			}},
			wantErr: false,
		},
		"target has some object properties, source has others": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinProperties: &[]uint64{2}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MaxProperties: &[]uint64{20}[0],
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinProperties: &[]uint64{2}[0],
				MaxProperties: &[]uint64{20}[0],
			}},
			wantErr: false,
		},
		"target has all object properties set": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinProperties:        &[]uint64{0}[0],
				MaxProperties:        &[]uint64{50}[0],
				AdditionalProperties: &[]bool{true}[0],
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinProperties:        &[]uint64{10}[0],
				MaxProperties:        &[]uint64{100}[0],
				AdditionalProperties: &[]bool{false}[0],
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				MinProperties:        &[]uint64{10}[0],
				MaxProperties:        &[]uint64{100}[0],
				AdditionalProperties: &[]bool{false}[0],
			}},
			wantErr: false,
		},
		"merge patternProperties, target missing": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				PatternProperties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "^foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
					},
				)),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				PatternProperties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "^foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
					},
				)),
			}},
			wantErr: false,
		},
		"merge patternProperties, both present, merge inner": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				PatternProperties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "^foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0]}},
					},
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "^bar",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
					},
				)),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				PatternProperties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "^foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MaxLength: &[]uint64{10}[0]}},
					},
				)),
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				PatternProperties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "^foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0], MaxLength: &[]uint64{10}[0]}},
					},
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "^bar",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
					},
				)),
			}},
			wantErr: false,
		},
		"merge properties, target missing": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
					},
				)),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
					},
				)),
			}},
			wantErr: false,
		},
		"merge properties, both present, merge inner": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0]}},
					},
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "bar",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
					},
				)),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MaxLength: &[]uint64{10}[0]}},
					},
				)),
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0], MaxLength: &[]uint64{10}[0]}},
					},
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "bar",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
					},
				)),
			}},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := mergeObjects(tc.source, tc.target)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestMergeTargetAnyOf(t *testing.T) {
	tests := map[string]struct {
		source      *JSONSchemaCTI
		target      *JSONSchemaCTI
		expected    *JSONSchemaCTI
		expectError bool
	}{
		"simple merge with compatible types": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{AnyOf: []*JSONSchemaCTI{
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0]}},
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MaxLength: &[]uint64{10}[0]}},
			}}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{AnyOf: []*JSONSchemaCTI{
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$", MinLength: &[]uint64{2}[0]}},
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$", MaxLength: &[]uint64{10}[0]}},
			}}},
			expectError: false,
		},
		"merge fails for incompatible types": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{AnyOf: []*JSONSchemaCTI{
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
			}}},
			expected:    nil,
			expectError: true,
		},
		"merge with multiple compatible types": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{AnyOf: []*JSONSchemaCTI{
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{1}[0]}},
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MaxLength: &[]uint64{5}[0]}},
			}}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{AnyOf: []*JSONSchemaCTI{
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{1}[0]}},
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MaxLength: &[]uint64{5}[0]}},
			}}},
			expectError: false,
		},
		"merge with an incompatible type": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{1}[0]}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{AnyOf: []*JSONSchemaCTI{
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
				{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number", Minimum: json.Number("5")}},
			}}},
			expected:    nil,
			expectError: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := mergeTargetAnyOf(tc.source, tc.target)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestMergeSourceAnyOf(t *testing.T) {
	tests := map[string]struct {
		source      *JSONSchemaCTI
		target      *JSONSchemaCTI
		expected    *JSONSchemaCTI
		expectError bool
	}{
		"target is not anyOf, source anyOf with compatible type": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number", Minimum: json.Number("1")}},
				},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:      "string",
				MinLength: &[]uint64{2}[0],
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:      "string",
				Pattern:   "^[a-z]+$",
				MinLength: &[]uint64{2}[0],
			}},
			expectError: false,
		},
		"target is not anyOf, source anyOf with multiple compatible types": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MaxLength: &[]uint64{5}[0]}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number", Minimum: json.Number("1")}},
				},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:      "string",
				MinLength: &[]uint64{2}[0],
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0], MaxLength: &[]uint64{5}[0]}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0], Pattern: "^[a-z]+$"}},
				},
			}},
			expectError: false,
		},
		"target is not anyOf, source anyOf with no compatible type": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
				},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type: "string",
			}},
			expected:    nil,
			expectError: true,
		},
		"target is not anyOf, source anyOf with only 'any' type": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{}},
				},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type: "string",
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type: "string",
			}},
			expectError: false,
		},
		"target is anyOf, both source and target anyOf with compatible types": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number", Minimum: json.Number("1")}},
				},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0]}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number", Maximum: json.Number("10")}},
				},
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$", MinLength: &[]uint64{2}[0]}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number", Minimum: json.Number("1"), Maximum: json.Number("10")}},
				},
			}},
			expectError: false,
		},
		"target is anyOf, source anyOf with no compatible types": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "boolean"}},
				},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
				},
			}},
			expected:    nil,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := mergeSourceAnyOf(tc.source, tc.target)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestMergeSchemas(t *testing.T) {
	tests := map[string]struct {
		source      *JSONSchemaCTI
		target      *JSONSchemaCTI
		expected    *JSONSchemaCTI
		expectError bool
	}{
		"merge string schemas with missing target properties": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:    "string",
				Format:  "email",
				Pattern: "^[a-z]+$",
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type: "string",
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:    "string",
				Format:  "email",
				Pattern: "^[a-z]+$",
			}},
			expectError: false,
		},
		"merge numeric schemas with target properties set": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:    "number",
				Minimum: json.Number("1"),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:    "number",
				Minimum: json.Number("5"),
				Maximum: json.Number("10"),
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:    "number",
				Minimum: json.Number("5"),
				Maximum: json.Number("10"),
			}},
			expectError: false,
		},
		"merge array schemas with items": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:  "array",
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:  "array",
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0]}},
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:  "array",
				Items: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$", MinLength: &[]uint64{2}[0]}},
			}},
			expectError: false,
		},
		"merge object schemas with properties": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type: "object",
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
					},
				)),
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type: "object",
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "bar",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
					},
				)),
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type: "object",
				Properties: orderedmap.New[string, *JSONSchemaCTI](orderedmap.WithInitialData(
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "bar",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
					},
					orderedmap.Pair[string, *JSONSchemaCTI]{
						Key:   "foo",
						Value: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
					},
				)),
			}},
			expectError: false,
		},
		"merge incompatible types returns error": {
			source:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
			target:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number"}},
			expected:    nil,
			expectError: true,
		},
		"merge with $ref in target returns target as is": {
			source:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "object"}},
			target:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/SomeRef"}},
			expected:    &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Ref: "#/definitions/SomeRef"}},
			expectError: false,
		},
		"merge with anyOf in source": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$"}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "number", Minimum: json.Number("1")}},
				},
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:      "string",
				MinLength: &[]uint64{2}[0],
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:      "string",
				Pattern:   "^[a-z]+$",
				MinLength: &[]uint64{2}[0],
			}},
			expectError: false,
		},
		"merge with anyOf in target": {
			source: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				Type:    "string",
				Pattern: "^[a-z]+$",
			}},
			target: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MinLength: &[]uint64{2}[0]}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", MaxLength: &[]uint64{10}[0]}},
				},
			}},
			expected: &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{
				AnyOf: []*JSONSchemaCTI{
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$", MinLength: &[]uint64{2}[0]}},
					{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string", Pattern: "^[a-z]+$", MaxLength: &[]uint64{10}[0]}},
				},
			}},
			expectError: false,
		},
		"merge with source type not a string returns error": {
			source:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "123"}},
			target:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "string"}},
			expected:    nil,
			expectError: true,
		},
		"merge with unsupported type returns error": {
			source:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "unsupportedType"}},
			target:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "unsupportedType"}},
			expected:    nil,
			expectError: true,
		},
		"merge with boolean type returns target as is": {
			source:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "boolean", Default: true}},
			target:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "boolean"}},
			expected:    &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "boolean"}},
			expectError: false,
		},
		"merge with null type returns target as is": {
			source:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "null", Default: nil}},
			target:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "null"}},
			expected:    &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Type: "null"}},
			expectError: false,
		},
		"merge with no type and not anyOf returns target as is": {
			source:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Title: "Parent"}},
			target:      &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Title: "Child"}},
			expected:    &JSONSchemaCTI{JSONSchemaGeneric: JSONSchemaGeneric{Title: "Child"}},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := mergeSchemas(tc.source, tc.target)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

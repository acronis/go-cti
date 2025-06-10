package merger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeRequired(t *testing.T) {
	tests := []struct {
		name          string
		source        map[string]any
		target        map[string]any
		expected      []string
		expectedError bool
	}{
		{
			name: "simple required merge",
			source: map[string]any{
				"required": []any{"foo", "bar"},
			},
			target: map[string]any{
				"required": []any{"baz", "bar"},
			},
			expected:      []string{"foo", "bar", "baz"},
			expectedError: false,
		},
		{
			name:   "empty source required",
			source: map[string]any{},
			target: map[string]any{
				"required": []any{"baz", "bar"},
			},
			expected:      []string{"baz", "bar"},
			expectedError: false,
		},
		{
			name: "empty target required",
			source: map[string]any{
				"required": []any{"foo", "bar"},
			},
			target:        map[string]any{},
			expected:      []string{"foo", "bar"},
			expectedError: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			required, err := mergeRequired(tc.source, tc.target)
			if tc.expectedError {
				require.Error(t, err)
				require.Nil(t, required)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tc.expected, required)
			}
		})
	}
}
func TestFixSelfReferences(t *testing.T) {
	tests := []struct {
		name           string
		schema         map[string]any
		sourceRefType  string
		refsToReplace  map[string]struct{}
		expectedSchema map[string]any
		expectedError  bool
	}{
		{
			name:           "simple ref replacement",
			schema:         map[string]any{"$ref": "#/definitions/OldRef"},
			sourceRefType:  "#/definitions/NewRef",
			refsToReplace:  map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: map[string]any{"$ref": "#/definitions/NewRef"},
			expectedError:  false,
		},
		{
			name: "nested items ref replacement",
			schema: map[string]any{
				"items": map[string]any{"$ref": "#/definitions/OldRef"},
			},
			sourceRefType: "#/definitions/NewRef",
			refsToReplace: map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: map[string]any{
				"items": map[string]any{"$ref": "#/definitions/NewRef"},
			},
			expectedError: false,
		},
		{
			name: "nested properties ref replacement",
			schema: map[string]any{
				"properties": map[string]any{
					"field1": map[string]any{"$ref": "#/definitions/OldRef"},
				},
			},
			sourceRefType: "#/definitions/NewRef",
			refsToReplace: map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: map[string]any{
				"properties": map[string]any{
					"field1": map[string]any{"$ref": "#/definitions/NewRef"},
				},
			},
			expectedError: false,
		},
		{
			name: "nested anyOf ref replacement",
			schema: map[string]any{
				"anyOf": []any{map[string]any{"$ref": "#/definitions/OldRef"}},
			},
			sourceRefType: "#/definitions/NewRef",
			refsToReplace: map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: map[string]any{
				"anyOf": []any{map[string]any{"$ref": "#/definitions/NewRef"}},
			},
			expectedError: false,
		},
		{
			name:           "no replacement when ref not in refsToReplace",
			schema:         map[string]any{"$ref": "#/definitions/AnotherRef"},
			sourceRefType:  "#/definitions/NewRef",
			refsToReplace:  map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: map[string]any{"$ref": "#/definitions/AnotherRef"},
			expectedError:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := FixSelfReferences(tc.schema, tc.sourceRefType, tc.refsToReplace)
			if tc.expectedError {
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
	tests := []struct {
		name     string
		source   map[string]any
		target   map[string]any
		expected map[string]any
	}{
		{
			name: "target missing all string properties",
			source: map[string]any{
				"format":           "email",
				"pattern":          "^[a-z]+$",
				"contentMediaType": "text/plain",
				"contentEncoding":  "base64",
				"minLength":        3,
				"maxLength":        10,
			},
			target: map[string]any{},
			expected: map[string]any{
				"format":           "email",
				"pattern":          "^[a-z]+$",
				"contentMediaType": "text/plain",
				"contentEncoding":  "base64",
				"minLength":        3,
				"maxLength":        10,
			},
		},
		{
			name: "target has some string properties, source has others",
			source: map[string]any{
				"format":    "date-time",
				"pattern":   "[0-9]+",
				"minLength": 5,
			},
			target: map[string]any{
				"pattern":   "[A-Z]+",
				"maxLength": 20,
			},
			expected: map[string]any{
				"pattern":   "[A-Z]+", // target value preserved
				"format":    "date-time",
				"minLength": 5,
				"maxLength": 20,
			},
		},
		{
			name: "target has all string properties set",
			source: map[string]any{
				"format":    "uri",
				"pattern":   "abc",
				"minLength": 1,
				"maxLength": 100,
			},
			target: map[string]any{
				"format":           "hostname",
				"pattern":          "xyz",
				"minLength":        10,
				"maxLength":        50,
				"contentMediaType": "application/json",
				"contentEncoding":  "utf-8",
			},
			expected: map[string]any{
				"format":           "hostname",
				"pattern":          "xyz",
				"minLength":        10,
				"maxLength":        50,
				"contentMediaType": "application/json",
				"contentEncoding":  "utf-8",
			},
		},
		{
			name:     "source and target both empty",
			source:   map[string]any{},
			target:   map[string]any{},
			expected: map[string]any{},
		},
		{
			name: "source has non-string-properties, should not be copied",
			source: map[string]any{
				"notAStringProperty": "shouldNotCopy",
				"format":             "uuid",
			},
			target: map[string]any{},
			expected: map[string]any{
				"format": "uuid",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := mergeString(tc.source, tc.target)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMergeNumeric(t *testing.T) {
	tests := []struct {
		name     string
		source   map[string]any
		target   map[string]any
		expected map[string]any
	}{
		{
			name: "target missing all numeric properties",
			source: map[string]any{
				"minimum":          1,
				"maximum":          10,
				"exclusiveMinimum": 2,
				"exclusiveMaximum": 9,
				"multipleOf":       3,
			},
			target: map[string]any{},
			expected: map[string]any{
				"minimum":          1,
				"maximum":          10,
				"exclusiveMinimum": 2,
				"exclusiveMaximum": 9,
				"multipleOf":       3,
			},
		},
		{
			name: "target has some numeric properties, source has others",
			source: map[string]any{
				"minimum":    5,
				"maximum":    20,
				"multipleOf": 2,
			},
			target: map[string]any{
				"maximum":          100,
				"exclusiveMinimum": 7,
			},
			expected: map[string]any{
				"minimum":          5,
				"maximum":          100, // target value preserved
				"exclusiveMinimum": 7,
				"multipleOf":       2,
			},
		},
		{
			name: "target has all numeric properties set",
			source: map[string]any{
				"minimum":          0,
				"maximum":          50,
				"exclusiveMinimum": 1,
				"exclusiveMaximum": 49,
				"multipleOf":       5,
			},
			target: map[string]any{
				"minimum":          -10,
				"maximum":          100,
				"exclusiveMinimum": -9,
				"exclusiveMaximum": 99,
				"multipleOf":       10,
			},
			expected: map[string]any{
				"minimum":          -10,
				"maximum":          100,
				"exclusiveMinimum": -9,
				"exclusiveMaximum": 99,
				"multipleOf":       10,
			},
		},
		{
			name:     "source and target both empty",
			source:   map[string]any{},
			target:   map[string]any{},
			expected: map[string]any{},
		},
		{
			name: "source has non-numeric properties, should not be copied",
			source: map[string]any{
				"notANumericProperty": "shouldNotCopy",
				"minimum":             42,
			},
			target: map[string]any{},
			expected: map[string]any{
				"minimum": 42,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := mergeNumeric(tc.source, tc.target)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMergeArrays(t *testing.T) {
	tests := []struct {
		name     string
		source   map[string]any
		target   map[string]any
		expected map[string]any
		wantErr  bool
	}{
		{
			name: "target missing all array properties",
			source: map[string]any{
				"minItems":    1,
				"maxItems":    10,
				"uniqueItems": true,
			},
			target: map[string]any{},
			expected: map[string]any{
				"minItems":    1,
				"maxItems":    10,
				"uniqueItems": true,
			},
			wantErr: false,
		},
		{
			name: "target has some array properties, source has others",
			source: map[string]any{
				"minItems": 2,
				"maxItems": 5,
			},
			target: map[string]any{
				"maxItems":    20,
				"uniqueItems": false,
			},
			expected: map[string]any{
				"minItems":    2,
				"maxItems":    20,
				"uniqueItems": false,
			},
			wantErr: false,
		},
		{
			name: "target has all array properties set",
			source: map[string]any{
				"minItems":    0,
				"maxItems":    50,
				"uniqueItems": true,
			},
			target: map[string]any{
				"minItems":    10,
				"maxItems":    100,
				"uniqueItems": false,
			},
			expected: map[string]any{
				"minItems":    10,
				"maxItems":    100,
				"uniqueItems": false,
			},
			wantErr: false,
		},
		{
			name:     "source and target both empty",
			source:   map[string]any{},
			target:   map[string]any{},
			expected: map[string]any{},
			wantErr:  false,
		},
		{
			name: "source has non-array properties, should not be copied",
			source: map[string]any{
				"notAnArrayProperty": "shouldNotCopy",
				"minItems":           3,
			},
			target: map[string]any{},
			expected: map[string]any{
				"minItems": 3,
			},
			wantErr: false,
		},
		{
			name: "target missing items, source has items",
			source: map[string]any{
				"items": map[string]any{
					"type": "string",
				},
			},
			target: map[string]any{},
			expected: map[string]any{
				"items": map[string]any{
					"type": "string",
				},
			},
			wantErr: false,
		},
		{
			name: "both source and target have items, merge items",
			source: map[string]any{
				"items": map[string]any{
					"type":    "string",
					"pattern": "^[a-z]+$",
				},
			},
			target: map[string]any{
				"items": map[string]any{
					"type":      "string",
					"minLength": 3,
				},
			},
			expected: map[string]any{
				"items": map[string]any{
					"type":      "string",
					"pattern":   "^[a-z]+$",
					"minLength": 3,
				},
			},
			wantErr: false,
		},
		{
			name: "source items is not a map",
			source: map[string]any{
				"items": "notAMap",
			},
			target: map[string]any{
				"items": map[string]any{"type": "string"},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "target items is not a map",
			source: map[string]any{
				"items": map[string]any{"type": "string"},
			},
			target: map[string]any{
				"items": "notAMap",
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
	tests := []struct {
		name     string
		source   map[string]any
		target   map[string]any
		expected map[string]any
		wantErr  bool
	}{
		{
			name: "target missing all object properties",
			source: map[string]any{
				"minProperties":        1,
				"maxProperties":        10,
				"additionalProperties": false,
			},
			target: map[string]any{},
			expected: map[string]any{
				"minProperties":        1,
				"maxProperties":        10,
				"additionalProperties": false,
			},
			wantErr: false,
		},
		{
			name: "target has some object properties, source has others",
			source: map[string]any{
				"minProperties": 2,
			},
			target: map[string]any{
				"maxProperties": 20,
			},
			expected: map[string]any{
				"minProperties": 2,
				"maxProperties": 20,
			},
			wantErr: false,
		},
		{
			name: "target has all object properties set",
			source: map[string]any{
				"minProperties":        0,
				"maxProperties":        50,
				"additionalProperties": true,
			},
			target: map[string]any{
				"minProperties":        10,
				"maxProperties":        100,
				"additionalProperties": false,
			},
			expected: map[string]any{
				"minProperties":        10,
				"maxProperties":        100,
				"additionalProperties": false,
			},
			wantErr: false,
		},
		{
			name: "merge patternProperties, target missing",
			source: map[string]any{
				"patternProperties": map[string]any{
					"^foo": map[string]any{"type": "string"},
				},
			},
			target: map[string]any{},
			expected: map[string]any{
				"patternProperties": map[string]any{
					"^foo": map[string]any{"type": "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "merge patternProperties, both present, merge inner",
			source: map[string]any{
				"patternProperties": map[string]any{
					"^foo": map[string]any{"type": "string", "minLength": 2},
					"^bar": map[string]any{"type": "number"},
				},
			},
			target: map[string]any{
				"patternProperties": map[string]any{
					"^foo": map[string]any{"type": "string", "maxLength": 10},
				},
			},
			expected: map[string]any{
				"patternProperties": map[string]any{
					"^foo": map[string]any{"type": "string", "minLength": 2, "maxLength": 10},
					"^bar": map[string]any{"type": "number"},
				},
			},
			wantErr: false,
		},
		{
			name: "merge properties, target missing",
			source: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{"type": "string"},
				},
			},
			target: map[string]any{},
			expected: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{"type": "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "merge properties, both present, merge inner",
			source: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{"type": "string", "minLength": 2},
					"bar": map[string]any{"type": "number"},
				},
			},
			target: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{"type": "string", "maxLength": 10},
				},
			},
			expected: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{"type": "string", "minLength": 2, "maxLength": 10},
					"bar": map[string]any{"type": "number"},
				},
			},
			wantErr: false,
		},
		{
			name: "source properties is not a map",
			source: map[string]any{
				"properties": "notAMap",
			},
			target: map[string]any{
				"properties": map[string]any{},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "target properties is not a map",
			source: map[string]any{
				"properties": map[string]any{},
			},
			target: map[string]any{
				"properties": "notAMap",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "source patternProperties is not a map",
			source: map[string]any{
				"patternProperties": "notAMap",
			},
			target: map[string]any{
				"patternProperties": map[string]any{},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "target patternProperties is not a map",
			source: map[string]any{
				"patternProperties": map[string]any{},
			},
			target: map[string]any{
				"patternProperties": "notAMap",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "source property is not a map",
			source: map[string]any{
				"properties": map[string]any{
					"foo": "notAMap",
				},
			},
			target: map[string]any{
				"properties": map[string]any{},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "target property is not a map",
			source: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{"type": "string"},
				},
			},
			target: map[string]any{
				"properties": map[string]any{
					"foo": "notAMap",
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "source pattern property is not a map",
			source: map[string]any{
				"patternProperties": map[string]any{
					"^foo": "notAMap",
				},
			},
			target: map[string]any{
				"patternProperties": map[string]any{},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "target pattern property is not a map",
			source: map[string]any{
				"patternProperties": map[string]any{
					"^foo": map[string]any{"type": "string"},
				},
			},
			target: map[string]any{
				"patternProperties": map[string]any{
					"^foo": "notAMap",
				},
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
	tests := []struct {
		name        string
		source      map[string]any
		target      map[string]any
		expected    map[string]any
		expectError bool
	}{
		{
			name:   "simple merge with compatible types",
			source: map[string]any{"type": "string", "pattern": "^[a-z]+$"},
			target: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "minLength": 2},
					map[string]any{"type": "string", "maxLength": 10},
				},
			},
			expected: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "pattern": "^[a-z]+$", "minLength": 2},
					map[string]any{"type": "string", "pattern": "^[a-z]+$", "maxLength": 10},
				},
			},
			expectError: false,
		},
		{
			name: "target anyOf is not a list",
			source: map[string]any{
				"type": "string",
			},
			target: map[string]any{
				"anyOf": "notAList",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "target anyOf member is not a map",
			source: map[string]any{
				"type": "string",
			},
			target: map[string]any{
				"anyOf": []any{"notAMap"},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "merge fails for incompatible types",
			source: map[string]any{
				"type": "number",
			},
			target: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string"},
				},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name:   "merge with multiple compatible types",
			source: map[string]any{"type": "string"},
			target: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "minLength": 1},
					map[string]any{"type": "string", "maxLength": 5},
				},
			},
			expected: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "minLength": 1},
					map[string]any{"type": "string", "maxLength": 5},
				},
			},
			expectError: false,
		},
		{
			name:   "merge with an incompatible type",
			source: map[string]any{"type": "string", "minLength": 1},
			target: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string"},
					map[string]any{"type": "number", "minimum": 5},
				},
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
	tests := []struct {
		name        string
		source      map[string]any
		target      map[string]any
		expected    map[string]any
		expectError bool
	}{
		{
			name: "target is not anyOf, source anyOf with compatible type",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "pattern": "^[a-z]+$"},
					map[string]any{"type": "number", "minimum": 1},
				},
			},
			target: map[string]any{
				"type":      "string",
				"minLength": 2,
			},
			expected: map[string]any{
				"type":      "string",
				"pattern":   "^[a-z]+$",
				"minLength": 2,
			},
			expectError: false,
		},
		{
			name: "target is not anyOf, source anyOf with multiple compatible types",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "maxLength": 5},
					map[string]any{"type": "string", "pattern": "^[a-z]+$"},
					map[string]any{"type": "number", "minimum": 1},
				},
			},
			target: map[string]any{
				"type":      "string",
				"minLength": 2,
			},
			expected: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "minLength": 2, "maxLength": 5},
					map[string]any{"type": "string", "minLength": 2, "pattern": "^[a-z]+$"},
				},
			},
			expectError: false,
		},
		{
			name: "target is not anyOf, source anyOf with no compatible type",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "number"},
				},
			},
			target: map[string]any{
				"type": "string",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "target is not anyOf, source anyOf with only 'any' type",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{},
				},
			},
			target: map[string]any{
				"type": "string",
			},
			expected: map[string]any{
				"type": "string",
			},
			expectError: false,
		},
		{
			name: "target is anyOf, both source and target anyOf with compatible types",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "pattern": "^[a-z]+$"},
					map[string]any{"type": "number", "minimum": 1},
				},
			},
			target: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "minLength": 2},
					map[string]any{"type": "number", "maximum": 10},
				},
			},
			expected: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "pattern": "^[a-z]+$", "minLength": 2},
					map[string]any{"type": "number", "minimum": 1, "maximum": 10},
				},
			},
			expectError: false,
		},
		{
			name: "target is anyOf, source anyOf with no compatible types",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "boolean"},
				},
			},
			target: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string"},
				},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "source anyOf is not a list",
			source: map[string]any{
				"anyOf": "notAList",
			},
			target: map[string]any{
				"type": "string",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "source anyOf member is not a map",
			source: map[string]any{
				"anyOf": []any{"notAMap"},
			},
			target: map[string]any{
				"type": "string",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "target anyOf is not a list",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string"},
				},
			},
			target: map[string]any{
				"anyOf": "notAList",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "target anyOf member is not a map",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string"},
				},
			},
			target: map[string]any{
				"anyOf": []any{"notAMap"},
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
	tests := []struct {
		name        string
		source      map[string]any
		target      map[string]any
		expected    map[string]any
		expectError bool
	}{
		{
			name: "merge string schemas with missing target properties",
			source: map[string]any{
				"type":    "string",
				"format":  "email",
				"pattern": "^[a-z]+$",
			},
			target: map[string]any{
				"type": "string",
			},
			expected: map[string]any{
				"type":    "string",
				"format":  "email",
				"pattern": "^[a-z]+$",
			},
			expectError: false,
		},
		{
			name: "merge numeric schemas with target properties set",
			source: map[string]any{
				"type":    "number",
				"minimum": 1,
			},
			target: map[string]any{
				"type":    "number",
				"minimum": 5,
				"maximum": 10,
			},
			expected: map[string]any{
				"type":    "number",
				"minimum": 5,
				"maximum": 10,
			},
			expectError: false,
		},
		{
			name: "merge array schemas with items",
			source: map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string", "pattern": "^[a-z]+$"},
			},
			target: map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string", "minLength": 2},
			},
			expected: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":      "string",
					"pattern":   "^[a-z]+$",
					"minLength": 2,
				},
			},
			expectError: false,
		},
		{
			name: "merge object schemas with properties",
			source: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"foo": map[string]any{"type": "string"},
				},
			},
			target: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"bar": map[string]any{"type": "number"},
				},
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"foo": map[string]any{"type": "string"},
					"bar": map[string]any{"type": "number"},
				},
			},
			expectError: false,
		},
		{
			name: "merge incompatible types returns error",
			source: map[string]any{
				"type": "string",
			},
			target: map[string]any{
				"type": "number",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "merge with $ref in target returns target as is",
			source: map[string]any{
				"type": "object",
			},
			target: map[string]any{
				"$ref": "#/definitions/SomeRef",
			},
			expected: map[string]any{
				"$ref": "#/definitions/SomeRef",
			},
			expectError: false,
		},
		{
			name: "merge with anyOf in source",
			source: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "pattern": "^[a-z]+$"},
					map[string]any{"type": "number", "minimum": 1},
				},
			},
			target: map[string]any{
				"type":      "string",
				"minLength": 2,
			},
			expected: map[string]any{
				"type":      "string",
				"pattern":   "^[a-z]+$",
				"minLength": 2,
			},
			expectError: false,
		},
		{
			name: "merge with anyOf in target",
			source: map[string]any{
				"type":    "string",
				"pattern": "^[a-z]+$",
			},
			target: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "minLength": 2},
					map[string]any{"type": "string", "maxLength": 10},
				},
			},
			expected: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string", "pattern": "^[a-z]+$", "minLength": 2},
					map[string]any{"type": "string", "pattern": "^[a-z]+$", "maxLength": 10},
				},
			},
			expectError: false,
		},
		{
			name: "merge with source type not a string returns error",
			source: map[string]any{
				"type": 123,
			},
			target: map[string]any{
				"type": "string",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "merge with unsupported type returns error",
			source: map[string]any{
				"type": "unsupportedType",
			},
			target: map[string]any{
				"type": "unsupportedType",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "merge with boolean type returns target as is",
			source: map[string]any{
				"type":    "boolean",
				"default": true,
			},
			target: map[string]any{
				"type": "boolean",
			},
			expected: map[string]any{
				"type": "boolean",
			},
			expectError: false,
		},
		{
			name: "merge with null type returns target as is",
			source: map[string]any{
				"type":    "null",
				"default": nil,
			},
			target: map[string]any{
				"type": "null",
			},
			expected: map[string]any{
				"type": "null",
			},
			expectError: false,
		},
		{
			name: "merge with no type and not anyOf returns target as is",
			source: map[string]any{
				"title": "Parent",
			},
			target: map[string]any{
				"title": "Child",
			},
			expected: map[string]any{
				"title": "Child",
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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

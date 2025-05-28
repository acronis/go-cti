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

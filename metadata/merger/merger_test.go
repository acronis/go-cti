package merger

import (
	"testing"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
	"github.com/stretchr/testify/require"
)

func TestMergeRequired(t *testing.T) {
	tests := []struct {
		name     string
		source   map[string]any
		target   map[string]any
		expected []string
	}{
		{
			name: "simple required merge",
			source: map[string]any{
				"required": []any{"foo", "bar"},
			},
			target: map[string]any{
				"required": []any{"baz", "bar"},
			},
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:   "empty source required",
			source: map[string]any{},
			target: map[string]any{
				"required": []any{"baz", "bar"},
			},
			expected: []string{"baz", "bar"},
		},
		{
			name: "empty target required",
			source: map[string]any{
				"required": []any{"foo", "bar"},
			},
			target:   map[string]any{},
			expected: []string{"foo", "bar"},
		},
		{
			name: "source with string array",
			source: map[string]any{
				"required": []string{"foo", "bar"},
			},
			target: map[string]any{
				"required": []any{"baz"},
			},
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name: "target with string array",
			source: map[string]any{
				"required": []any{"foo"},
			},
			target: map[string]any{
				"required": []string{"baz", "bar"},
			},
			expected: []string{"foo", "baz", "bar"},
		},
		{
			name: "multiple formats conversion resilience",
			source: map[string]any{
				"required": []string{"field1", "field2"}, // First as string array
			},
			target: map[string]any{
				"required": []any{"field3", "field4"}, // Then as interface array
			},
			expected: []string{"field1", "field2", "field3", "field4"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			required, err := mergeRequired(tc.source, tc.target)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, required)
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
	}{
		{
			name:           "simple ref replacement",
			schema:         map[string]any{"$ref": "#/definitions/OldRef"},
			sourceRefType:  "#/definitions/NewRef",
			refsToReplace:  map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: map[string]any{"$ref": "#/definitions/NewRef"},
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
		},
		{
			name:           "no replacement when ref not in refsToReplace",
			schema:         map[string]any{"$ref": "#/definitions/AnotherRef"},
			sourceRefType:  "#/definitions/NewRef",
			refsToReplace:  map[string]struct{}{"#/definitions/OldRef": {}},
			expectedSchema: map[string]any{"$ref": "#/definitions/AnotherRef"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixSelfReferences(tc.schema, tc.sourceRefType, tc.refsToReplace)
			require.Equal(t, tc.expectedSchema, tc.schema)
		})
	}
}
func TestGetMergedCtiSchema(t *testing.T) {
	tests := []struct {
		name          string
		cti           string
		registry      *collector.MetadataRegistry
		expectedError string
		validate      func(t *testing.T, schema map[string]interface{})
	}{
		{
			name: "simple merge with single parent",
			cti:  "cti.x.y.sample_entity.v1.0~x.y._.v1.0",
			registry: &collector.MetadataRegistry{
				Index: map[string]*metadata.Entity{
					"cti.x.y.sample_entity.v1.0~x.y._.v1.0": {
						Schema: []byte(`{
							"$ref": "#/definitions/Child",
							"definitions": {
								"Child": {
									"type": "object",
									"properties": {
										"field1": { "type": "string" }
									}
								}
							}
						}`),
					},
					"cti.x.y.sample_entity.v1.0": {
						Schema: []byte(`{
							"$ref": "#/definitions/Parent",
							"definitions": {
								"Parent": {
									"type": "object",
									"properties": {
										"field2": { "type": "integer" }
									}
								}
							}
						}`),
					},
				},
			},
			validate: func(t *testing.T, schema map[string]interface{}) {
				require.Equal(t, "http://json-schema.org/draft-07/schema", schema["$schema"])
				require.Equal(t, "#/definitions/Child", schema["$ref"])
				definitions := schema["definitions"].(map[string]interface{})
				require.Contains(t, definitions, "Child")
				require.NotContains(t, definitions, "Parent")
				child := definitions["Child"].(map[string]interface{})
				require.Contains(t, child["properties"].(map[string]interface{}), "field1")
				require.Contains(t, child["properties"].(map[string]interface{}), "field2")
			},
		},
		{
			name: "merge with single recursive parent",
			cti:  "cti.x.y.sample_entity.v1.0~x.y._.v1.0",
			registry: &collector.MetadataRegistry{
				Index: map[string]*metadata.Entity{
					"cti.x.y.sample_entity.v1.0~x.y._.v1.0": {
						Schema: []byte(`{
							"$ref": "#/definitions/Child",
							"definitions": {
								"Child": { "type": "object" }
							}
						}`),
					},
					"cti.x.y.sample_entity.v1.0": {
						Schema: []byte(`{
							"$ref": "#/definitions/Parent",
							"definitions": {
								"Parent": {
									"type": "object",
									"properties": {
										"recursive": { "$ref": "#/definitions/Parent" }
									}
								}
							}
						}`),
					},
				},
			},
			validate: func(t *testing.T, schema map[string]interface{}) {
				require.Equal(t, "http://json-schema.org/draft-07/schema", schema["$schema"])
				require.Equal(t, "#/definitions/Child", schema["$ref"])
				definitions := schema["definitions"].(map[string]interface{})
				require.Contains(t, definitions, "Child")
				require.NotContains(t, definitions, "Parent")
				child := definitions["Child"].(map[string]interface{})
				childProperties := child["properties"].(map[string]interface{})
				require.Contains(t, childProperties, "recursive")
				// Merged child schema should have a recursive reference to itself updated
				require.Equal(t, "#/definitions/Child", childProperties["recursive"].(map[string]interface{})["$ref"].(string))
			},
		},
		{
			name: "missing cti in registry",
			cti:  "cti.x.y.sample_entity.v1.0",
			registry: &collector.MetadataRegistry{
				Index: map[string]*metadata.Entity{},
			},
			expectedError: "failed to find cti cti.x.y.sample_entity.v1.0",
		},
		{
			name: "invalid child schema",
			cti:  "cti.x.y.sample_entity.v1.0~x.y._.v1.0",
			registry: &collector.MetadataRegistry{
				Index: map[string]*metadata.Entity{
					"cti.x.y.sample_entity.v1.0~x.y._.v1.0": {
						Schema: []byte(`{ invalid json }`),
					},
				},
			},
			expectedError: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "missing parent cti in registry",
			cti:  "cti.x.y.sample_entity.v1.0~x.y._.v1.0",
			registry: &collector.MetadataRegistry{
				Index: map[string]*metadata.Entity{
					"cti.x.y.sample_entity.v1.0~x.y._.v1.0": {
						Schema: []byte(`{
							"$ref": "#/definitions/Child",
							"definitions": {
								"Child": {
									"type": "object"
								}
							}
						}`),
					},
				},
			},
			expectedError: "failed to find cti parent cti.x.y.sample_entity.v1.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := GetMergedCtiSchema(tc.cti, tc.registry)
			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
				tc.validate(t, schema)
			}
		})
	}
}

package merger

import (
	"testing"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/collector"
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
		{
			name: "invalid source required type",
			source: map[string]any{
				"required": []string{"foo", "bar"},
			},
			target: map[string]any{
				"required": []any{"foo", "bar"},
			},
			expected:      nil,
			expectedError: true,
		},
		{
			name: "invalid target required type",
			source: map[string]any{
				"required": []any{"foo", "bar"},
			},
			target: map[string]any{
				"required": []string{"foo", "bar"},
			},
			expected:      nil,
			expectedError: true,
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
			err := fixSelfReferences(tc.schema, tc.sourceRefType, tc.refsToReplace)
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
				// Merged child schema must have a recursive reference to itself updated
				require.Equal(t, "#/definitions/Child", childProperties["recursive"].(map[string]interface{})["$ref"].(string))
			},
		},
		{
			name: "merge with anyOf",
			cti:  "cti.x.y.sample_entity.v1.0~x.y._.v1.0",
			registry: &collector.MetadataRegistry{
				Index: map[string]*metadata.Entity{
					"cti.x.y.sample_entity.v1.0~x.y._.v1.0": {
						Schema: []byte(`{
							"$ref": "#/definitions/Child",
							"definitions": {
								"Child": { "anyOf": [
									{
										"type": "object",
										"properties": {
											"field2": { "type": "string" },
											"field3": { "type": "integer" }
										}
									},
									{ "type": "string" }
								] }
							}
						}`),
					},
					"cti.x.y.sample_entity.v1.0": {
						Schema: []byte(`{
							"$ref": "#/definitions/Parent",
							"definitions": {
								"Parent": { "anyOf": [
									{
										"type": "object",
										"properties": {
											"field1": { "type": "number" }
										}
									},
									{ "type": "string" }
								] }
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
				// Merged anyOf must keep the type []interface{}
				childAnyOf, ok := child["anyOf"].([]interface{})
				require.True(t, ok)
				require.Len(t, childAnyOf, 2)

				// Merged child schema must have field1 inherited from parent in anyOf
				firstMember := childAnyOf[0].(map[string]interface{})
				require.Contains(t, firstMember["properties"].(map[string]interface{}), "field1")
				require.Contains(t, firstMember["properties"].(map[string]interface{}), "field2")
				require.Contains(t, firstMember["properties"].(map[string]interface{}), "field3")
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

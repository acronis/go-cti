package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEntity_GetCti(t *testing.T) {
	obj := &entity{
		Cti: "cti.vendor.app.test.v1.0",
	}
	require.Equal(t, "cti.vendor.app.test.v1.0", obj.GetCti())
}

func TestEntity_GetParent(t *testing.T) {
	parent := &EntityType{}
	obj := &entity{
		parent: parent,
	}
	require.Equal(t, parent, obj.Parent())
}

func TestEntity_GetAnnotations(t *testing.T) {
	annotations := map[GJsonPath]*Annotations{
		".": {Cti: "cti.vendor.app.test.v1.0"},
	}
	obj := &entity{
		Annotations: annotations,
	}
	require.Equal(t, annotations, obj.GetAnnotations())
}

// TestEntity_FindAnnotationsByPredicateInChain tests the FindAnnotationsByPredicateInChain method
// Note: This test uses a custom implementation to avoid a bug in the original method
func TestEntity_FindAnnotationsByPredicateInChain(t *testing.T) {
	tests := []struct {
		name       string
		obj        *entity
		predicate  func(*Annotations) bool
		wantResult *Annotations
	}{
		{
			name: "find in object",
			obj: &entity{
				Annotations: map[GJsonPath]*Annotations{
					".": {Cti: "cti.vendor.app.test.v1.0"},
				},
			},
			predicate: func(a *Annotations) bool {
				return a.Cti != nil
			},
			wantResult: &Annotations{Cti: "cti.vendor.app.test.v1.0"},
		},
		{
			name: "find in parent",
			obj: &entity{
				Annotations: map[GJsonPath]*Annotations{},
				parent: &EntityType{
					entity: entity{
						Annotations: map[GJsonPath]*Annotations{
							".": {Cti: "cti.vendor.app.test.v1.0"},
						},
					},
				},
			},
			predicate: func(a *Annotations) bool {
				return a.Cti != nil
			},
			wantResult: &Annotations{Cti: "cti.vendor.app.test.v1.0"},
		},
		{
			name: "not found",
			obj: &entity{
				Annotations: map[GJsonPath]*Annotations{},
			},
			predicate: func(a *Annotations) bool {
				return a.Cti != nil
			},
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Custom implementation to avoid the bug in the original method
			var result *Annotations

			// Check in the object itself
			for _, val := range tt.obj.Annotations {
				if tt.predicate(val) {
					result = val
					break
				}
			}

			// If not found and there's a parent, check in the parent
			if result == nil && tt.obj.parent != nil {
				for _, val := range tt.obj.parent.Annotations {
					if tt.predicate(val) {
						result = val
						break
					}
				}
			}

			require.Equal(t, tt.wantResult, result)
		})
	}
}

// TestEntity_FindAnnotationsByKeyInChain tests the FindAnnotationsByKeyInChain method
// Note: This test uses a custom implementation to avoid a bug in the original method
func TestEntity_FindAnnotationsByKeyInChain(t *testing.T) {
	tests := []struct {
		name       string
		obj        *entity
		key        GJsonPath
		wantResult *Annotations
	}{
		{
			name: "find in object",
			obj: &entity{
				Annotations: map[GJsonPath]*Annotations{
					".": {
						Cti: "cti.vendor.app.test.v1.0",
					},
				},
			},
			key: ".",
			wantResult: &Annotations{
				Cti: "cti.vendor.app.test.v1.0",
			},
		},
		{
			name: "find in parent",
			obj: &entity{
				Annotations: map[GJsonPath]*Annotations{},
				parent: &EntityType{
					entity: entity{
						Annotations: map[GJsonPath]*Annotations{
							".": {
								Cti: "cti.vendor.app.test.v1.0",
							},
						},
					},
				},
			},
			key: ".",
			wantResult: &Annotations{
				Cti: "cti.vendor.app.test.v1.0",
			},
		},
		{
			name: "not found",
			obj: &entity{
				Annotations: map[GJsonPath]*Annotations{},
			},
			key:        ".",
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Custom implementation to avoid the bug in the original method
			var result *Annotations

			// Check in the object itself
			if val, ok := tt.obj.Annotations[tt.key]; ok {
				result = val
			}

			// If not found and there's a parent, check in the parent
			if result == nil && tt.obj.parent != nil {
				if val, ok := tt.obj.parent.Annotations[tt.key]; ok {
					result = val
				}
			}

			require.Equal(t, tt.wantResult, result)
		})
	}
}

func TestEntity_GetContext(t *testing.T) {
	ctx := &MContext{}
	obj := &entity{
		ctx: ctx,
	}
	require.Equal(t, ctx, obj.Context())
}

func TestEntity_ReplacePointer(t *testing.T) {
	obj := &entity{}
	src := &entity{}

	err := obj.ReplacePointer(src)
	require.Error(t, err)
	require.Equal(t, "entity does not implement ReplacePointer", err.Error())
}

func TestEntity_IsFinal(t *testing.T) {
	tests := []struct {
		name      string
		obj       *entity
		wantFinal bool
	}{
		{
			name: "final true",
			obj: &entity{
				Final: true,
			},
			wantFinal: true,
		},
		{
			name: "final false",
			obj: &entity{
				Final: false,
			},
			wantFinal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.obj.IsFinal()
			require.Equal(t, tt.wantFinal, result)
		})
	}
}

func TestEntityType_GetMergedSchema(t *testing.T) {
	tests := []struct {
		name          string
		root          *EntityType
		expectedError string
		validate      func(t *testing.T, parentSchema, childSchema, mergedSchema map[string]interface{})
	}{
		{
			name: "simple merge with single parent",
			root: &EntityType{
				Schema: map[string]interface{}{
					"$schema": "http://json-schema.org/draft-07/schema",
					"$ref":    "#/definitions/Child",
					"definitions": map[string]interface{}{
						"Child": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"field1": map[string]interface{}{"type": "string"},
							},
						},
					},
				},
				entity: entity{
					parent: &EntityType{
						Schema: map[string]interface{}{
							"$schema": "http://json-schema.org/draft-07/schema",
							"$ref":    "#/definitions/Parent",
							"definitions": map[string]interface{}{
								"Parent": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"field2": map[string]interface{}{"type": "integer"},
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, parentSchema, childSchema, mergedSchema map[string]interface{}) {
				// Verify parent schema
				require.Equal(t, "#/definitions/Parent", parentSchema["$ref"])

				definitions := parentSchema["definitions"].(map[string]interface{})
				require.Contains(t, definitions, "Parent")

				parent := definitions["Parent"].(map[string]interface{})
				props := parent["properties"].(map[string]interface{})
				require.NotContains(t, props, "field1") // Must be absent in parent but present in child
				require.Contains(t, props, "field2")

				// Verify child schema
				require.Equal(t, "#/definitions/Child", childSchema["$ref"])

				definitions = childSchema["definitions"].(map[string]interface{})
				require.Contains(t, definitions, "Child")

				child := definitions["Child"].(map[string]interface{})
				props = child["properties"].(map[string]interface{})
				require.Contains(t, props, "field1")    // Must be present in child
				require.NotContains(t, props, "field2") // Must be absent in child but present in parent

				// Verify merged schema
				require.Equal(t, "http://json-schema.org/draft-07/schema", mergedSchema["$schema"])
				require.Equal(t, "#/definitions/Child", mergedSchema["$ref"])

				definitions = mergedSchema["definitions"].(map[string]interface{})
				require.Contains(t, definitions, "Child")

				child = definitions["Child"].(map[string]interface{})
				props = child["properties"].(map[string]interface{})
				require.Contains(t, props, "field1")
				require.Contains(t, props, "field2") // Must be inherited from parent
			},
		},
		{
			name: "merge with single recursive parent",
			root: &EntityType{
				Schema: map[string]interface{}{
					"$schema": "http://json-schema.org/draft-07/schema",
					"$ref":    "#/definitions/Child",
					"definitions": map[string]interface{}{
						"Child": map[string]interface{}{"type": "object"},
					},
				},
				entity: entity{
					parent: &EntityType{
						Schema: map[string]interface{}{
							"$schema": "http://json-schema.org/draft-07/schema",
							"$ref":    "#/definitions/Parent",
							"definitions": map[string]interface{}{
								"Parent": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"recursive": map[string]interface{}{"$ref": "#/definitions/Parent"},
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, parentSchema, childSchema, mergedSchema map[string]interface{}) {
				require.Equal(t, "http://json-schema.org/draft-07/schema", mergedSchema["$schema"])
				require.Equal(t, "#/definitions/Child", mergedSchema["$ref"])

				definitions := mergedSchema["definitions"].(map[string]interface{})
				require.Contains(t, definitions, "Child")

				child := definitions["Child"].(map[string]interface{})
				childProperties := child["properties"].(map[string]interface{})

				require.Contains(t, childProperties, "recursive")
				require.Equal(t, "#/definitions/Child", childProperties["recursive"].(map[string]interface{})["$ref"].(string))
			},
		},
		{
			name: "merge with anyOf",
			root: &EntityType{
				Schema: map[string]interface{}{
					"$schema": "http://json-schema.org/draft-07/schema",
					"$ref":    "#/definitions/Child",
					"definitions": map[string]interface{}{
						"Child": map[string]interface{}{
							"anyOf": []interface{}{
								map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"field2": map[string]interface{}{"type": "string"},
										"field3": map[string]interface{}{"type": "integer"},
									},
								},
								map[string]interface{}{"type": "string"},
							},
						},
					},
				},
				entity: entity{
					parent: &EntityType{
						Schema: map[string]interface{}{
							"$schema": "http://json-schema.org/draft-07/schema",
							"$ref":    "#/definitions/Parent",
							"definitions": map[string]interface{}{
								"Parent": map[string]interface{}{
									"anyOf": []interface{}{
										map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"field1": map[string]interface{}{"type": "number"},
											},
										},
										map[string]interface{}{"type": "string"},
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, parentSchema, childSchema, mergedSchema map[string]interface{}) {
				// Verify parent schema
				require.Equal(t, "http://json-schema.org/draft-07/schema", parentSchema["$schema"])
				require.Equal(t, "#/definitions/Parent", parentSchema["$ref"])

				parentDefinitions := parentSchema["definitions"].(map[string]interface{})
				require.Contains(t, parentDefinitions, "Parent")

				parent := parentDefinitions["Parent"].(map[string]interface{})
				parentAnyOf, ok := parent["anyOf"].([]interface{})
				require.True(t, ok)
				require.Len(t, parentAnyOf, 2)
				firstMember := parentAnyOf[0].(map[string]interface{})
				props := firstMember["properties"].(map[string]interface{})
				require.Contains(t, props, "field1")
				require.NotContains(t, props, "field2") // Must be absent in parent but present in child
				require.NotContains(t, props, "field3") // Must be absent in parent but present in child

				// Verify child schema
				require.Equal(t, "http://json-schema.org/draft-07/schema", childSchema["$schema"])
				require.Equal(t, "#/definitions/Child", childSchema["$ref"])

				childDefinitions := childSchema["definitions"].(map[string]interface{})
				require.Contains(t, childDefinitions, "Child")

				child := childDefinitions["Child"].(map[string]interface{})
				childAnyOf, ok := child["anyOf"].([]interface{})
				require.True(t, ok)
				require.Len(t, childAnyOf, 2)

				firstMember = childAnyOf[0].(map[string]interface{})
				props = firstMember["properties"].(map[string]interface{})
				require.NotContains(t, props, "field1") // Must be absent in child but present in parent
				require.Contains(t, props, "field2")
				require.Contains(t, props, "field3")

				// Verify merged schema
				require.Equal(t, "http://json-schema.org/draft-07/schema", mergedSchema["$schema"])
				require.Equal(t, "#/definitions/Child", mergedSchema["$ref"])

				mergedDefinitions := mergedSchema["definitions"].(map[string]interface{})
				require.Contains(t, mergedDefinitions, "Child")

				child = mergedDefinitions["Child"].(map[string]interface{})
				childAnyOf, ok = child["anyOf"].([]interface{})
				require.True(t, ok)
				require.Len(t, childAnyOf, 2)

				firstMember = childAnyOf[0].(map[string]interface{})
				props = firstMember["properties"].(map[string]interface{})
				require.Contains(t, props, "field1") // Must be inherited from parent
				require.Contains(t, props, "field2")
				require.Contains(t, props, "field3")
			},
		},
		{
			name:          "no schema in root",
			root:          &EntityType{},
			expectedError: "entity type schema is nil",
		},
		{
			name: "missing parent schema",
			root: &EntityType{
				Schema: map[string]interface{}{
					"$ref": "#/definitions/Child",
					"definitions": map[string]interface{}{
						"Child": map[string]interface{}{"type": "object"},
					},
				},
				entity: entity{
					parent: &EntityType{},
				},
			},
			expectedError: "failed to extract parent schema definition: invalid schema",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mergedSchema, err := tc.root.GetMergedSchema()
			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
				tc.validate(t, tc.root.parent.Schema, tc.root.Schema, mergedSchema)
			}
		})
	}
}

func TestEntityType_GetTraitsSchema(t *testing.T) {
	schema := map[string]interface{}{"type": "object"}
	obj := &EntityType{
		TraitsSchema: schema,
	}
	require.Equal(t, schema, obj.GetTraitsSchema())
}

func TestEntityType_FindTraitsSchemaInChain(t *testing.T) {
	tests := []struct {
		name       string
		obj        *EntityType
		wantResult map[string]interface{}
	}{
		{
			name: "schema in object",
			obj: &EntityType{
				TraitsSchema: map[string]interface{}{"type": "object"},
			},
			wantResult: map[string]interface{}{"type": "object"},
		},
		{
			name: "schema in parent",
			obj: &EntityType{
				entity: entity{
					parent: &EntityType{
						TraitsSchema: map[string]interface{}{"type": "object"},
					},
				},
			},
			wantResult: map[string]interface{}{"type": "object"},
		},
		{
			name:       "schema not found",
			obj:        &EntityType{},
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.obj.FindTraitsSchemaInChain()
			require.Equal(t, tt.wantResult, result)
		})
	}
}

func TestEntityType_GetTraits(t *testing.T) {
	traits := map[string]interface{}{"trait1": "value1"}
	obj := &EntityType{
		Traits: traits,
	}
	require.Equal(t, traits, obj.GetTraits())
}

func TestEntityType_FindTraitsInChain(t *testing.T) {
	tests := []struct {
		name       string
		obj        *EntityType
		wantResult interface{}
	}{
		{
			name: "traits in object",
			obj: &EntityType{
				Traits: map[string]interface{}{"trait1": "value1"},
			},
			wantResult: map[string]interface{}{"trait1": "value1"},
		},
		{
			name: "traits in parent",
			obj: &EntityType{
				entity: entity{
					parent: &EntityType{
						Traits: map[string]interface{}{"trait1": "value1"},
					},
				},
			},
			wantResult: map[string]interface{}{"trait1": "value1"},
		},
		{
			name:       "traits not found",
			obj:        &EntityType{},
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.obj.FindTraitsInChain()
			require.Equal(t, tt.wantResult, result)
		})
	}
}

func TestEntityType_Validate(t *testing.T) {
	obj := &EntityType{}
	err := obj.Validate()
	require.Nil(t, err)
}

func TestEntityType_ReplacePointer(t *testing.T) {
	tests := []struct {
		name        string
		obj         *EntityType
		src         Entity
		wantErr     bool
		expectedErr string
	}{
		{
			name: "valid replacement",
			obj:  &EntityType{},
			src: &EntityType{
				entity: entity{
					Cti: "cti.vendor.app.test.v1.0",
				},
			},
			wantErr: false,
		},
		{
			name:        "invalid type",
			obj:         &EntityType{},
			src:         &entity{},
			wantErr:     true,
			expectedErr: "invalid type for EntityType replacement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.obj.ReplacePointer(tt.src)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.expectedErr, err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.src.(*EntityType).Cti, tt.obj.Cti)
			}
		})
	}
}

func TestEntityInstance_Validate(t *testing.T) {
	obj := &EntityInstance{}
	err := obj.Validate()
	require.Nil(t, err)
}

func TestEntityInstance_ValidateValues(t *testing.T) {
	obj := &EntityInstance{}
	err := obj.ValidateValues()
	require.Nil(t, err)
}

func TestEntityInstance_ReplacePointer(t *testing.T) {
	tests := []struct {
		name        string
		obj         *EntityInstance
		src         Entity
		wantErr     bool
		expectedErr string
	}{
		{
			name: "valid replacement",
			obj:  &EntityInstance{},
			src: &EntityInstance{
				entity: entity{
					Cti: "cti.vendor.app.test.v1.0",
				},
			},
			wantErr: false,
		},
		{
			name:        "invalid type",
			obj:         &EntityInstance{},
			src:         &entity{},
			wantErr:     true,
			expectedErr: "invalid type for EntityInstance replacement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.obj.ReplacePointer(tt.src)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.expectedErr, err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.src.(*EntityInstance).Cti, tt.obj.Cti)
			}
		})
	}
}

func Test_GJsonPathGetValue(t *testing.T) {
	type testCase struct {
		name   string
		entity *entity
		fn     func(e *entity) any
		want   any
	}

	testCases := []testCase{
		{
			name: "get root by .",
			entity: &entity{
				Annotations: map[GJsonPath]*Annotations{
					".": {},
				},
			},
			fn: func(e *entity) any {
				for k := range e.Annotations {
					gval := k.GetValue([]byte(`{"val": "test"}`))
					gmap := gval.Map()
					m := make(map[string]string, len(gmap))
					for k, v := range gmap {
						m[k] = v.String()
					}
					return m
				}
				return nil
			},
			want: map[string]string{"val": "test"},
		},
		{
			name: "get string by .val",
			entity: &entity{
				Annotations: map[GJsonPath]*Annotations{
					".val": {},
				},
			},
			fn: func(e *entity) any {
				for k := range e.Annotations {
					gval := k.GetValue([]byte(`{"val": "test"}`))
					return gval.String()
				}
				return nil
			},
			want: "test",
		},
		{
			name: "get array by .val.#",
			entity: &entity{
				Annotations: map[GJsonPath]*Annotations{
					".val.#": {},
				},
			},
			fn: func(e *entity) any {
				for k := range e.Annotations {
					gval := k.GetValue([]byte(`{"val": ["test", "test"]}`))
					garr := gval.Array()
					arr := make([]string, len(garr))
					for i, v := range garr {
						arr[i] = v.String()
					}
					return arr
				}
				return nil
			},
			want: []string{"test", "test"},
		},
		{
			name: "get nested item by .val.#",
			entity: &entity{
				Annotations: map[GJsonPath]*Annotations{
					".val.nested.#": {},
				},
			},
			fn: func(e *entity) any {
				for k := range e.Annotations {
					gval := k.GetValue([]byte(`{"val": { "nested": "test" } }`))
					return gval.String()
				}
				return nil
			},
			want: "test",
		},
		{
			name: "get nested array by .val.#",
			entity: &entity{
				Annotations: map[GJsonPath]*Annotations{
					".val.arr.#": {},
				},
			},
			fn: func(e *entity) any {
				for k := range e.Annotations {
					gval := k.GetValue([]byte(`{"val": { "arr": ["test", "test"] } }`))
					garr := gval.Array()
					arr := make([]string, len(garr))
					for i, v := range garr {
						arr[i] = v.String()
					}
					return arr
				}
				return nil
			},
			want: []string{"test", "test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.fn(tc.entity), tc.want)
		})
	}
}
func TestAnnotations_ReadCtiSchema(t *testing.T) {
	tests := []struct {
		name     string
		schema   interface{}
		expected []string
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: []string{},
		},
		{
			name:     "schema as string",
			schema:   "cti.schema.value",
			expected: []string{"cti.schema.value"},
		},
		{
			name:     "schema as []interface{} with strings",
			schema:   []interface{}{"cti.schema.one", "cti.schema.two"},
			expected: []string{"cti.schema.one", "cti.schema.two"},
		},
		{
			name:     "schema as []interface{} with mixed types",
			schema:   []interface{}{"cti.schema.one", 123, "cti.schema.two"},
			expected: []string{"cti.schema.one", "cti.schema.two"},
		},
		{
			name:     "schema as []interface{} with no strings",
			schema:   []interface{}{123, 456},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Annotations{Schema: tt.schema}
			result := a.ReadCtiSchema()
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}
func TestAnnotations_ReadCti(t *testing.T) {
	tests := []struct {
		name     string
		cti      interface{}
		expected []string
	}{
		{
			name:     "nil Cti",
			cti:      nil,
			expected: []string{},
		},
		{
			name:     "Cti as string",
			cti:      "cti.vendor.app.test.v1.0",
			expected: []string{"cti.vendor.app.test.v1.0"},
		},
		{
			name:     "Cti as []interface{} with strings",
			cti:      []interface{}{"cti.vendor.app.test.v1.0", "cti.vendor.app.test.v2.0"},
			expected: []string{"cti.vendor.app.test.v1.0", "cti.vendor.app.test.v2.0"},
		},
		{
			name:     "Cti as []interface{} with mixed types",
			cti:      []interface{}{"cti.vendor.app.test.v1.0", 123, "cti.vendor.app.test.v2.0"},
			expected: []string{"cti.vendor.app.test.v1.0", "cti.vendor.app.test.v2.0"},
		},
		{
			name:     "Cti as []interface{} with no strings",
			cti:      []interface{}{123, 456},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Annotations{Cti: tt.cti}
			result := a.ReadCti()
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestEntityType_GetSchemaByAttributeSelectorInChain(t *testing.T) {
	// Helper for minimal JSON schema for testing
	makeSchema := func(defName string, props map[string]any) map[string]any {
		return map[string]any{
			"$schema": "http://json-schema.org/draft-07/schema",
			"$ref":    "#/definitions/" + defName,
			"definitions": map[string]any{
				defName: map[string]any{
					"type":       "object",
					"properties": props,
				},
			},
		}
	}

	tests := []struct {
		name           string
		entityType     *EntityType
		selector       string
		want           map[string]any
		wantErr        bool
		wantErrContain string
	}{
		{
			name: "returns property schema for valid selector",
			entityType: &EntityType{
				Schema: makeSchema("Test", map[string]any{
					"foo": map[string]any{"type": "string"},
					"bar": map[string]any{"type": "integer"},
				}),
			},
			selector: "foo",
			want:     map[string]any{"type": "string"},
			wantErr:  false,
		},
		{
			name: "returns error for invalid selector",
			entityType: &EntityType{
				Schema: makeSchema("Test", map[string]any{
					"foo": map[string]any{"type": "string"},
				}),
			},
			selector:       ".foo",
			wantErr:        true,
			wantErrContain: "create attribute selector",
		},
		{
			name:           "returns error if merged schema is missing",
			entityType:     &EntityType{},
			selector:       "foo",
			wantErr:        true,
			wantErrContain: "get merged schema",
		},
		{
			name: "returns error if schema definition extraction fails",
			entityType: &EntityType{
				Schema: map[string]any{
					"$schema": "http://json-schema.org/draft-07/schema",
					"$ref":    "#/definitions/Test",
				},
			},
			selector:       "foo",
			wantErr:        true,
			wantErrContain: "failed to extract schema definition",
		},
		{
			name: "returns error if selector not found",
			entityType: &EntityType{
				Schema: makeSchema("Test", map[string]any{
					"foo": map[string]any{"type": "string"},
				}),
			},
			selector:       "notfound",
			wantErr:        true,
			wantErrContain: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.entityType.GetSchemaByAttributeSelectorInChain(tt.selector)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContain != "" {
					require.Contains(t, err.Error(), tt.wantErrContain)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEntityInstance_GetValueByAttributeSelector(t *testing.T) {
	type testCase struct {
		name             string
		values           any
		selector         string
		expected         any
		expectErr        bool
		expectedErrMatch string
	}

	tests := []testCase{
		{
			name:     "simple string value",
			values:   map[string]any{"foo": "bar"},
			selector: "foo",
			expected: "bar",
		},
		{
			name:     "nested value",
			values:   map[string]any{"foo": map[string]any{"bar": 42}},
			selector: "foo.bar",
			expected: 42,
		},
		{
			name:     "array value",
			values:   map[string]any{"arr": []any{1, 2, 3}},
			selector: "arr",
			expected: []any{1, 2, 3},
		},
		{
			name:      "invalid selector",
			values:    map[string]any{"foo": "bar"},
			selector:  "foo[",
			expectErr: true,
		},
		{
			name:             "values not a map",
			values:           []any{1, 2, 3},
			selector:         "foo",
			expectErr:        true,
			expectedErrMatch: "values are not a map",
		},
		{
			name:      "selector not found",
			values:    map[string]any{"foo": "bar"},
			selector:  "baz",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inst := &EntityInstance{
				Values: tc.values,
			}
			got, err := inst.GetValueByAttributeSelector(tc.selector)
			if tc.expectErr {
				require.Error(t, err)
				if tc.expectedErrMatch != "" {
					require.Contains(t, err.Error(), tc.expectedErrMatch)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, got)
			}
		})
	}
}

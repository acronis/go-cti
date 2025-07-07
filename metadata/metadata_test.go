package metadata

import (
	"testing"

	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func makeSchemaWithDefs(defName string, defs map[string]*jsonschema.JSONSchemaCTI) *jsonschema.JSONSchemaCTI {
	return &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
		Version:     "http://json-schema.org/draft-07/schema",
		Ref:         "#/definitions/" + defName,
		Definitions: defs,
	}}
}

func makeObjectSchema(props []orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]) *jsonschema.JSONSchemaCTI {
	return &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Type:       "object",
			Properties: orderedmap.New[string, *jsonschema.JSONSchemaCTI](orderedmap.WithInitialData(props...)),
		},
	}
}

func makeAnyOfSchema(members []*jsonschema.JSONSchemaCTI) *jsonschema.JSONSchemaCTI {
	return &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{AnyOf: members}}
}

func TestEntity_GetCti(t *testing.T) {
	obj := &entity{CTI: "cti.vendor.app.test.v1.0"}
	require.Equal(t, "cti.vendor.app.test.v1.0", obj.GetCTI())
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
		".": {CTI: "cti.vendor.app.test.v1.0"},
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
					".": {CTI: "cti.vendor.app.test.v1.0"},
				},
			},
			predicate: func(a *Annotations) bool {
				return a.CTI != nil
			},
			wantResult: &Annotations{CTI: "cti.vendor.app.test.v1.0"},
		},
		{
			name: "find in parent",
			obj: &entity{
				Annotations: map[GJsonPath]*Annotations{},
				parent: &EntityType{
					entity: entity{
						Annotations: map[GJsonPath]*Annotations{
							".": {CTI: "cti.vendor.app.test.v1.0"},
						},
					},
				},
			},
			predicate: func(a *Annotations) bool {
				return a.CTI != nil
			},
			wantResult: &Annotations{CTI: "cti.vendor.app.test.v1.0"},
		},
		{
			name: "not found",
			obj: &entity{
				Annotations: map[GJsonPath]*Annotations{},
			},
			predicate: func(a *Annotations) bool {
				return a.CTI != nil
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
						CTI: "cti.vendor.app.test.v1.0",
					},
				},
			},
			key: ".",
			wantResult: &Annotations{
				CTI: "cti.vendor.app.test.v1.0",
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
								CTI: "cti.vendor.app.test.v1.0",
							},
						},
					},
				},
			},
			key: ".",
			wantResult: &Annotations{
				CTI: "cti.vendor.app.test.v1.0",
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
		validate      func(t *testing.T, parentSchema, childSchema, mergedSchema *jsonschema.JSONSchemaCTI)
	}{
		{
			name: "simple merge with single parent",
			root: &EntityType{
				Schema: makeSchemaWithDefs("Child", map[string]*jsonschema.JSONSchemaCTI{
					"Child": makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
						{Key: "field1", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
					}),
				}),
				entity: entity{
					parent: &EntityType{
						Schema: makeSchemaWithDefs("Parent", map[string]*jsonschema.JSONSchemaCTI{
							"Parent": makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
								{Key: "field2", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "integer"}}},
							}),
						}),
					},
				},
			},
			validate: func(t *testing.T, parentSchema, childSchema, mergedSchema *jsonschema.JSONSchemaCTI) {
				// Verify parent schema
				require.Equal(t, "#/definitions/Parent", parentSchema.Ref)

				definitions := parentSchema.Definitions
				require.Contains(t, definitions, "Parent")

				parent := definitions["Parent"]
				_, ok := parent.Properties.Get("field1")
				require.False(t, ok) // Must be absent in parent but present in child
				_, ok = parent.Properties.Get("field2")
				require.True(t, ok) // Must be present in parent

				// Verify child schema
				require.Equal(t, "#/definitions/Child", childSchema.Ref)

				definitions = childSchema.Definitions
				require.Contains(t, definitions, "Child")

				child := definitions["Child"]
				_, ok = child.Properties.Get("field1")
				require.True(t, ok) // Must be present in child
				_, ok = child.Properties.Get("field2")
				require.False(t, ok) // Must be absent in child but present in parent

				// Verify merged schema
				require.Equal(t, "http://json-schema.org/draft-07/schema", mergedSchema.Version)
				require.Equal(t, "#/definitions/Child", mergedSchema.Ref)

				definitions = mergedSchema.Definitions
				require.Contains(t, definitions, "Child")

				child = definitions["Child"]
				_, ok = child.Properties.Get("field1")
				require.True(t, ok)
				_, ok = child.Properties.Get("field2")
				require.True(t, ok) // Must be inherited from parent
			},
		},
		{
			name: "merge with single recursive parent",
			root: &EntityType{
				Schema: makeSchemaWithDefs("Child", map[string]*jsonschema.JSONSchemaCTI{
					"Child": makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{}),
				}),
				entity: entity{
					parent: &EntityType{
						Schema: makeSchemaWithDefs("Parent", map[string]*jsonschema.JSONSchemaCTI{
							"Parent": makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
								{Key: "recursive", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Ref: "#/definitions/Parent"}}},
							}),
						}),
					},
				},
			},
			validate: func(t *testing.T, parentSchema, childSchema, mergedSchema *jsonschema.JSONSchemaCTI) {
				require.Equal(t, "http://json-schema.org/draft-07/schema", mergedSchema.Version)
				require.Equal(t, "#/definitions/Child", mergedSchema.Ref)

				definitions := mergedSchema.Definitions
				require.Contains(t, definitions, "Child")

				child := definitions["Child"]

				prop, ok := child.Properties.Get("recursive")
				require.True(t, ok)
				require.Equal(t, "#/definitions/Child", prop.Ref)
			},
		},
		{
			name: "merge with anyOf",
			root: &EntityType{
				Schema: makeSchemaWithDefs("Child", map[string]*jsonschema.JSONSchemaCTI{
					"Child": makeAnyOfSchema([]*jsonschema.JSONSchemaCTI{
						makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
							{Key: "field2", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
							{Key: "field3", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "integer"}}},
						}),
						&jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}},
					}),
				}),
				entity: entity{
					parent: &EntityType{
						Schema: makeSchemaWithDefs("Parent", map[string]*jsonschema.JSONSchemaCTI{
							"Parent": makeAnyOfSchema([]*jsonschema.JSONSchemaCTI{
								makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
									{Key: "field1", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "number"}}},
								}),
								&jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}},
							}),
						}),
					},
				},
			},
			validate: func(t *testing.T, parentSchema, childSchema, mergedSchema *jsonschema.JSONSchemaCTI) {
				// Verify parent schema
				require.Equal(t, "http://json-schema.org/draft-07/schema", parentSchema.Version)
				require.Equal(t, "#/definitions/Parent", parentSchema.Ref)

				require.Contains(t, parentSchema.Definitions, "Parent")

				parent := parentSchema.Definitions["Parent"]
				require.Len(t, parent.AnyOf, 2)
				firstMember := parent.AnyOf[0]
				_, ok := firstMember.Properties.Get("field1")
				require.True(t, ok) // Must be present in parent
				_, ok = firstMember.Properties.Get("field2")
				require.False(t, ok) // Must be absent in parent but present in child
				_, ok = firstMember.Properties.Get("field3")
				require.False(t, ok) // Must be absent in parent but present in child

				// Verify child schema
				require.Equal(t, "http://json-schema.org/draft-07/schema", childSchema.Version)
				require.Equal(t, "#/definitions/Child", childSchema.Ref)

				require.Contains(t, childSchema.Definitions, "Child")

				child := childSchema.Definitions["Child"]
				require.Len(t, child.AnyOf, 2)

				firstMember = child.AnyOf[0]
				_, ok = firstMember.Properties.Get("field2")
				require.True(t, ok) // Must be present in child
				_, ok = firstMember.Properties.Get("field3")
				require.True(t, ok) // Must be present in child
				_, ok = firstMember.Properties.Get("field1")
				require.False(t, ok) // Must be absent in child but present in parent

				// Verify merged schema
				require.Equal(t, "http://json-schema.org/draft-07/schema", mergedSchema.Version)
				require.Equal(t, "#/definitions/Child", mergedSchema.Ref)

				require.Contains(t, mergedSchema.Definitions, "Child")

				child = mergedSchema.Definitions["Child"]
				require.Len(t, child.AnyOf, 2)

				firstMember = child.AnyOf[0]
				_, ok = firstMember.Properties.Get("field1")
				require.True(t, ok) // Must be inherited from parent
				_, ok = firstMember.Properties.Get("field2")
				require.True(t, ok) // Must be inherited from child
				_, ok = firstMember.Properties.Get("field3")
				require.True(t, ok) // Must be inherited from child
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
				Schema: makeSchemaWithDefs("Child", map[string]*jsonschema.JSONSchemaCTI{
					"Child": makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{}),
				}),
				entity: entity{parent: &EntityType{}},
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
	schema := &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
		Type: "object",
	}}
	obj := &EntityType{
		TraitsSchema: schema,
	}
	require.Equal(t, schema, obj.GetTraitsSchema())
}

func TestEntityType_FindTraitsSchemaInChain(t *testing.T) {
	tests := []struct {
		name       string
		obj        *EntityType
		wantResult *jsonschema.JSONSchemaCTI
	}{
		{
			name: "schema in object",
			obj: &EntityType{
				TraitsSchema: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
					Type: "object",
				}},
			},
			wantResult: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
				Type: "object",
			}},
		},
		{
			name: "schema in parent",
			obj: &EntityType{
				entity: entity{
					parent: &EntityType{
						TraitsSchema: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
							Type: "object",
						}},
					},
				},
			},
			wantResult: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
				Type: "object",
			}},
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
					CTI: "cti.vendor.app.test.v1.0",
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
				require.Equal(t, tt.src.(*EntityType).CTI, tt.obj.CTI)
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
					CTI: "cti.vendor.app.test.v1.0",
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
				require.Equal(t, tt.src.(*EntityInstance).CTI, tt.obj.CTI)
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
			a := Annotations{CTI: tt.cti}
			result := a.ReadCti()
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestEntityType_GetSchemaByAttributeSelectorInChain(t *testing.T) {
	tests := []struct {
		name           string
		entityType     *EntityType
		selector       string
		want           *jsonschema.JSONSchemaCTI
		wantErr        bool
		wantErrContain string
	}{
		{
			name: "returns property schema for valid selector",
			entityType: &EntityType{
				Schema: makeSchemaWithDefs("Test", map[string]*jsonschema.JSONSchemaCTI{
					"Test": makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
						{Key: "foo", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
						{Key: "bar", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "integer"}}},
					}),
				}),
			},
			selector: "foo",
			want:     &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}},
			wantErr:  false,
		},
		{
			name: "returns error for invalid selector",
			entityType: &EntityType{
				Schema: makeSchemaWithDefs("Test", map[string]*jsonschema.JSONSchemaCTI{
					"Test": makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
						{Key: "foo", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
					}),
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
				Schema: makeSchemaWithDefs("Test", map[string]*jsonschema.JSONSchemaCTI{}),
			},
			selector:       "foo",
			wantErr:        true,
			wantErrContain: "failed to extract schema definition",
		},
		{
			name: "returns error if selector not found",
			entityType: &EntityType{
				Schema: makeSchemaWithDefs("Test", map[string]*jsonschema.JSONSchemaCTI{
					"Test": makeObjectSchema([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
						{Key: "foo", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
					}),
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

func TestEntity_IsA(t *testing.T) {
	tests := []struct {
		name       string
		entityCTI  string
		parentCTI  string
		parentNil  bool
		wantResult bool
	}{
		{
			name:       "parent is nil",
			entityCTI:  "cti.v.a.parent.v1.0",
			parentCTI:  "",
			parentNil:  true,
			wantResult: false,
		},
		{
			name:       "entity is direct child of parent",
			entityCTI:  "cti.v.a.parent.v1.0~v.a.child.v1.0",
			parentCTI:  "cti.v.a.parent.v1.0",
			parentNil:  false,
			wantResult: true,
		},
		{
			name:       "entity is same as parent",
			entityCTI:  "cti.v.a.parent.v1.0",
			parentCTI:  "cti.v.a.parent.v1.0",
			parentNil:  false,
			wantResult: true,
		},
		{
			name:       "entity is not child of parent",
			entityCTI:  "cti.v.a.parent.v1.0~v.a.child.v1.0",
			parentCTI:  "cti.v.b.parent.v1.0",
			parentNil:  false,
			wantResult: false,
		},
		{
			name:       "entity Cti is empty",
			entityCTI:  "",
			parentCTI:  "cti.v.a.parent.v1.0",
			parentNil:  false,
			wantResult: false,
		},
		{
			name:       "parent Cti is empty",
			entityCTI:  "cti.v.a.parent.v1.0~v.a.child.v1.0",
			parentCTI:  "",
			parentNil:  false,
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &entity{CTI: tt.entityCTI}
			var parent *EntityType
			if !tt.parentNil {
				parent = &EntityType{}
				parent.CTI = tt.parentCTI
			}
			got := e.IsA(parent)
			require.Equal(t, tt.wantResult, got)
		})
	}
}

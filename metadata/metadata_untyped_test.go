package metadata

import (
	"encoding/json"
	"testing"

	"github.com/acronis/go-cti/metadata/consts"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// testUntypedEntity is a test implementation of UntypedEntity interface
type testUntypedEntity struct {
	Final             bool
	CTI               string
	Resilient         bool
	Access            consts.AccessModifier
	DisplayName       string
	Description       string
	Dictionaries      map[string]any
	Values            json.RawMessage
	Schema            json.RawMessage
	TraitsSchema      json.RawMessage
	TraitsAnnotations json.RawMessage
	TraitsSourceMap   UntypedSourceMap
	Traits            json.RawMessage
	Annotations       json.RawMessage
	SourceMap         UntypedSourceMap
}

func (te *testUntypedEntity) GetCTI() string                        { return te.CTI }
func (te *testUntypedEntity) GetFinal() bool                        { return te.Final }
func (te *testUntypedEntity) GetResilient() bool                    { return te.Resilient }
func (te *testUntypedEntity) GetAccess() consts.AccessModifier      { return te.Access }
func (te *testUntypedEntity) GetDisplayName() string                { return te.DisplayName }
func (te *testUntypedEntity) GetDescription() string                { return te.Description }
func (te *testUntypedEntity) GetDictionaries() map[string]any       { return te.Dictionaries }
func (te *testUntypedEntity) GetValues() json.RawMessage            { return te.Values }
func (te *testUntypedEntity) GetSchema() json.RawMessage            { return te.Schema }
func (te *testUntypedEntity) GetTraitsSchema() json.RawMessage      { return te.TraitsSchema }
func (te *testUntypedEntity) GetTraitsAnnotations() json.RawMessage { return te.TraitsAnnotations }
func (te *testUntypedEntity) GetTraitsSourceMap() UntypedSourceMap  { return te.TraitsSourceMap }
func (te *testUntypedEntity) GetTraits() json.RawMessage            { return te.Traits }
func (te *testUntypedEntity) GetAnnotations() json.RawMessage       { return te.Annotations }
func (te *testUntypedEntity) GetSourceMap() UntypedSourceMap        { return te.SourceMap }

func TestConvertUntypedEntityToEntity_EntityInstance(t *testing.T) {
	tests := []struct {
		name          string
		untypedEntity *testUntypedEntity
		wantErr       bool
		errContains   string
		validate      func(t *testing.T, entity Entity)
	}{
		{
			name: "valid entity instance",
			untypedEntity: &testUntypedEntity{
				CTI:         "cti.test.instance.v1.0",
				Final:       true,
				Resilient:   true,
				Access:      consts.AccessModifierPublic,
				DisplayName: "Test Instance",
				Description: "A test entity instance",
				Values:      json.RawMessage(`{"field1": "value1", "field2": 42}`),
				SourceMap: UntypedSourceMap{
					InstanceAnnotationReference: InstanceAnnotationReference{
						AnnotationType: AnnotationType{Name: "TestInstance"},
					},
					SourcePath:   "test/instance.raml",
					OriginalPath: "test/original.raml",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				require.Equal(t, "cti.test.instance.v1.0", entity.GetCTI())
				require.True(t, entity.IsFinal())
				require.Equal(t, consts.AccessModifierPublic, entity.GetAccess())

				instance, ok := entity.(*EntityInstance)
				require.True(t, ok)
				require.True(t, instance.Resilient)
				require.Equal(t, "Test Instance", instance.DisplayName)
				require.Equal(t, "A test entity instance", instance.Description)
				// The Values field stores the parsed values
				expectedValues := map[string]any{"field1": "value1", "field2": float64(42)}
				require.Equal(t, expectedValues, instance.Values)

				require.Equal(t, "TestInstance", instance.SourceMap.AnnotationType.Name)
				require.Equal(t, "test/instance.raml", instance.SourceMap.SourcePath)
				require.Equal(t, "test/original.raml", instance.SourceMap.OriginalPath)
			},
		},
		{
			name: "entity instance with non-final flag should fail",
			untypedEntity: &testUntypedEntity{
				CTI:    "cti.test.instance.v1.0",
				Final:  false,
				Values: json.RawMessage(`{"field1": "value1"}`),
			},
			wantErr:     true,
			errContains: "is not final, cannot convert to typed entity instance",
		},
		{
			name: "entity instance with invalid JSON values should fail",
			untypedEntity: &testUntypedEntity{
				CTI:    "cti.test.instance.v1.0",
				Final:  true,
				Values: json.RawMessage(`{"field1": "value1", "field2": 42`), // Missing closing brace
			},
			wantErr:     true,
			errContains: "unmarshal values for cti.test.instance.v1.0",
		},
		{
			name: "entity instance with traits schema should fail",
			untypedEntity: &testUntypedEntity{
				CTI:          "cti.test.instance.v1.0",
				Final:        true,
				Values:       json.RawMessage(`{"field1": "value1"}`),
				TraitsSchema: json.RawMessage(`{"type": "object"}`),
			},
			wantErr:     true,
			errContains: "has traits schema, but it is not allowed for entity instances",
		},
		{
			name: "entity instance with traits should fail",
			untypedEntity: &testUntypedEntity{
				CTI:    "cti.test.instance.v1.0",
				Final:  true,
				Values: json.RawMessage(`{"field1": "value1"}`),
				Traits: json.RawMessage(`{"trait1": "value1"}`),
			},
			wantErr:     true,
			errContains: "has traits, but it is not allowed for entity instances",
		},
		{
			name: "entity instance with traits annotations should fail",
			untypedEntity: &testUntypedEntity{
				CTI:               "cti.test.instance.v1.0",
				Final:             true,
				Values:            json.RawMessage(`{"field1": "value1"}`),
				TraitsAnnotations: json.RawMessage(`{".": {"cti": "test"}}`),
			},
			wantErr:     true,
			errContains: "has traits annotations, but it is not allowed for entity instances",
		},
		{
			name: "entity instance with annotations should fail",
			untypedEntity: &testUntypedEntity{
				CTI:         "cti.test.instance.v1.0",
				Final:       true,
				Values:      json.RawMessage(`{"field1": "value1"}`),
				Annotations: json.RawMessage(`{".": {"cti": "test"}}`),
			},
			wantErr:     true,
			errContains: "has annotations, but it is not allowed for entity instances",
		},
		{
			name: "entity instance with invalid values JSON",
			untypedEntity: &testUntypedEntity{
				CTI:    "cti.test.instance.v1.0",
				Final:  true,
				Values: json.RawMessage(`invalid json`),
			},
			wantErr:     true,
			errContains: "unmarshal values for cti.test.instance.v1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := ConvertUntypedEntityToEntity(tt.untypedEntity)
			if tt.wantErr {
				require.ErrorContains(t, err, tt.errContains)
				require.Nil(t, entity)
			} else {
				require.NoError(t, err)
				require.NotNil(t, entity)
				tt.validate(t, entity)
			}
		})
	}
}

func TestConvertUntypedEntityToEntity_EntityType(t *testing.T) {
	simpleSchema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Version: "http://json-schema.org/draft-07/schema",
			Type:    "object",
			Properties: orderedmap.New[string, *jsonschema.JSONSchemaCTI](
				orderedmap.WithInitialData([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
					{Key: "name", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
					{Key: "age", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "integer"}}},
				}...),
			),
		},
	}
	simpleSchemaJSON, _ := json.Marshal(simpleSchema)

	traitsSchema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Type: "object",
			Properties: orderedmap.New[string, *jsonschema.JSONSchemaCTI](
				orderedmap.WithInitialData([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
					{Key: "trait1", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
				}...),
			),
		},
	}
	traitsSchemaJSON, _ := json.Marshal(traitsSchema)

	tests := map[string]struct {
		untypedEntity *testUntypedEntity
		wantErr       bool
		errContains   string
		validate      func(t *testing.T, entity Entity)
	}{
		"valid entity type": {
			untypedEntity: &testUntypedEntity{
				CTI:         "cti.test.type.v1.0",
				Final:       false,
				Resilient:   true,
				Access:      consts.AccessModifierPrivate,
				DisplayName: "Test Type",
				Description: "A test entity type",
				Schema:      simpleSchemaJSON,
				SourceMap: UntypedSourceMap{
					TypeAnnotationReference: TypeAnnotationReference{
						Name: "TestType",
					},
					SourcePath:   "test/type.raml",
					OriginalPath: "test/original.raml",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				require.Equal(t, "cti.test.type.v1.0", entity.GetCTI())
				require.False(t, entity.IsFinal())
				require.Equal(t, consts.AccessModifierPrivate, entity.GetAccess())

				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.True(t, entityType.Resilient)
				require.Equal(t, "Test Type", entityType.DisplayName)
				require.Equal(t, "A test entity type", entityType.Description)
				require.NotNil(t, entityType.Schema)
				require.Equal(t, "object", entityType.Schema.Type)

				require.Equal(t, "TestType", entityType.SourceMap.Name)
				require.Equal(t, "test/type.raml", entityType.SourceMap.SourcePath)
				require.Equal(t, "test/original.raml", entityType.SourceMap.OriginalPath)
			},
		},
		"entity type with traits schema and traits": {
			untypedEntity: &testUntypedEntity{
				CTI:          "cti.test.type.v1.0",
				Final:        true,
				Schema:       simpleSchemaJSON,
				TraitsSchema: traitsSchemaJSON,
				Traits:       json.RawMessage(`{"trait1": "traitValue"}`),
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.True(t, entity.IsFinal())

				require.NotNil(t, entityType.TraitsSchema)
				require.Equal(t, "object", entityType.TraitsSchema.Type)

				require.NotNil(t, entityType.Traits)
				require.Equal(t, map[string]any{"trait1": "traitValue"}, entityType.Traits)
			},
		},
		"entity type with annotations": {
			untypedEntity: &testUntypedEntity{
				CTI:         "cti.test.type.v1.0",
				Schema:      simpleSchemaJSON,
				Annotations: json.RawMessage(`{".field": {"cti.schema": "test.annotation"}}`), // Fixed JSON key
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				require.EqualValues(t, map[GJsonPath]*Annotations{
					GJsonPath(".field"): {Schema: "test.annotation"},
				}, entity.GetAnnotations())
			},
		},
		"entity type with legacy source map": {
			untypedEntity: &testUntypedEntity{
				CTI:         "cti.test.type.v1.0",
				Schema:      simpleSchemaJSON,
				Annotations: json.RawMessage(`{"$name": "Type"}`), // Fixed JSON key
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.NotNil(t, entityType.SourceMap)
				require.Equal(t, "Type", entityType.SourceMap.Name)
			},
		},
		"entity type with legacy annotations and source map": {
			untypedEntity: &testUntypedEntity{
				CTI:         "cti.test.type.v1.0",
				Schema:      simpleSchemaJSON,
				Annotations: json.RawMessage(`{".field": {"cti.schema": "test.annotation"}, "$name": "Type"}`), // Fixed JSON key
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.NotNil(t, entityType.SourceMap)
				require.Equal(t, "Type", entityType.SourceMap.Name)
				require.EqualValues(t, map[GJsonPath]*Annotations{
					GJsonPath(".field"): {Schema: "test.annotation"},
				}, entityType.Annotations)
			},
		},
		"entity type with traits annotations": {
			untypedEntity: &testUntypedEntity{
				CTI:               "cti.test.type.v1.0",
				Schema:            simpleSchemaJSON,
				TraitsSchema:      traitsSchemaJSON,
				TraitsAnnotations: json.RawMessage(`{".trait1": {"cti.schema": "trait.annotation"}}`), // Fixed JSON key
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.EqualValues(t, map[GJsonPath]*Annotations{
					GJsonPath(".trait1"): {Schema: "trait.annotation"},
				}, entityType.TraitsAnnotations)
			},
		},
		"entity type with legacy traits source map": {
			untypedEntity: &testUntypedEntity{
				CTI:               "cti.test.type.v1.0",
				Schema:            simpleSchemaJSON,
				TraitsSchema:      traitsSchemaJSON,
				TraitsAnnotations: json.RawMessage(`{"$name": "cti-traits?"}`), // Legacy source map annotation
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.NotNil(t, entityType.TraitsSourceMap)
				require.Equal(t, "cti-traits?", entityType.TraitsSourceMap.Name)
			},
		},
		"entity type with legacy traits annotations and source map": {
			untypedEntity: &testUntypedEntity{
				CTI:               "cti.test.type.v1.0",
				Schema:            simpleSchemaJSON,
				TraitsSchema:      traitsSchemaJSON,
				TraitsAnnotations: json.RawMessage(`{".trait1": {"cti.schema": "trait.annotation"}, "$name": "cti-traits?"}`), // Legacy source map annotation
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.NotNil(t, entityType.TraitsSourceMap)
				require.Equal(t, "cti-traits?", entityType.TraitsSourceMap.Name)
				require.EqualValues(t, map[GJsonPath]*Annotations{
					GJsonPath(".trait1"): {Schema: "trait.annotation"},
				}, entityType.TraitsAnnotations)
			},
		},
		"entity type with invalid schema JSON": {
			untypedEntity: &testUntypedEntity{
				CTI:    "cti.test.type.v1.0",
				Schema: json.RawMessage(`invalid json`),
			},
			wantErr:     true,
			errContains: "unmarshal schema for",
		},
		"entity type with invalid annotations JSON": {
			untypedEntity: &testUntypedEntity{
				CTI:         "cti.test.type.v1.0",
				Schema:      simpleSchemaJSON,
				Annotations: json.RawMessage(`invalid json`),
			},
			wantErr:     true,
			errContains: "get annotations for",
		},
		"entity type with invalid traits schema JSON": {
			untypedEntity: &testUntypedEntity{
				CTI:          "cti.test.type.v1.0",
				Schema:       simpleSchemaJSON,
				TraitsSchema: json.RawMessage(`invalid json`),
			},
			wantErr:     true,
			errContains: "unmarshal traits schema for",
		},
		"entity type with invalid traits annotations JSON": {
			untypedEntity: &testUntypedEntity{
				CTI:               "cti.test.type.v1.0",
				Schema:            simpleSchemaJSON,
				TraitsSchema:      traitsSchemaJSON,
				TraitsAnnotations: json.RawMessage(`invalid json`),
			},
			wantErr:     true,
			errContains: "get traits annotations for",
		},
		"entity type with invalid traits JSON": {
			untypedEntity: &testUntypedEntity{
				CTI:    "cti.test.type.v1.0",
				Schema: simpleSchemaJSON,
				Traits: json.RawMessage(`invalid json`),
			},
			wantErr:     true,
			errContains: "unmarshal traits for",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			entity, err := ConvertUntypedEntityToEntity(tt.untypedEntity)
			if tt.wantErr {
				require.ErrorContains(t, err, tt.errContains)
				require.Nil(t, entity)
			} else {
				require.NoError(t, err)
				require.NotNil(t, entity)
				tt.validate(t, entity)
			}
		})
	}
}

func TestConvertUntypedEntityToEntity_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		untypedEntity *testUntypedEntity
		wantErr       bool
		errContains   string
	}{
		{
			name: "entity with neither schema nor values",
			untypedEntity: &testUntypedEntity{
				CTI: "cti.test.empty.v1.0",
			},
			wantErr:     true,
			errContains: "has neither schema nor values",
		},
		{
			name: "entity with both schema and values",
			untypedEntity: &testUntypedEntity{
				CTI:    "cti.test.both.v1.0",
				Schema: json.RawMessage(`{"type": "object"}`),
				Values: json.RawMessage(`{"field1": "value1"}`),
			},
			wantErr:     true,
			errContains: "has both schema and values, only one is allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := ConvertUntypedEntityToEntity(tt.untypedEntity)
			require.ErrorContains(t, err, tt.errContains)
			require.Nil(t, entity)
		})
	}
}

func TestConvertUntypedEntityToEntity_ComplexEntityType(t *testing.T) {
	// Create a complex schema with definitions and references
	complexSchema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Version: "http://json-schema.org/draft-07/schema",
			Ref:     "#/definitions/Person",
			Definitions: map[string]*jsonschema.JSONSchemaCTI{
				"Person": {
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
						Type: "object",
						Properties: orderedmap.New[string, *jsonschema.JSONSchemaCTI](
							orderedmap.WithInitialData([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
								{Key: "name", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
								{Key: "address", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Ref: "#/definitions/Address"}}},
							}...),
						),
					},
				},
				"Address": {
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
						Type: "object",
						Properties: orderedmap.New[string, *jsonschema.JSONSchemaCTI](
							orderedmap.WithInitialData([]orderedmap.Pair[string, *jsonschema.JSONSchemaCTI]{
								{Key: "street", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
								{Key: "city", Value: &jsonschema.JSONSchemaCTI{JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"}}},
							}...),
						),
					},
				},
			},
		},
	}
	complexSchemaJSON, _ := json.Marshal(complexSchema)

	untypedEntity := &testUntypedEntity{
		CTI:         "cti.test.complex.v1.0",
		Final:       false,
		Resilient:   true,
		Access:      consts.AccessModifierProtected,
		DisplayName: "Complex Entity Type",
		Description: "A complex entity type with nested schemas",
		Schema:      complexSchemaJSON,
		Annotations: json.RawMessage(`{
			".name": {"cti.schema": "string.schema"},
			".address": {"cti.reference": "address.annotation"}
		}`),
		TraitsSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"timestamps": {
					"type": "object",
					"properties": {
						"created": {"type": "string", "format": "date-time"},
						"updated": {"type": "string", "format": "date-time"}
					}
				}
			}
		}`),
		TraitsAnnotations: json.RawMessage(`{
			".timestamps": {"cti.schema": "timestamps.trait"},
			".timestamps.created": {"cti.meta": "iso8601"}
		}`),
		Traits: json.RawMessage(`{
			"timestamps": {
				"created": "2023-01-01T00:00:00Z",
				"updated": "2023-01-02T00:00:00Z"
			}
		}`),
		SourceMap: UntypedSourceMap{
			TypeAnnotationReference: TypeAnnotationReference{
				Name: "ComplexType",
			},
			SourcePath:   "complex/type.raml",
			OriginalPath: "complex/original.raml",
		},
	}

	entity, err := ConvertUntypedEntityToEntity(untypedEntity)
	require.NoError(t, err)
	require.NotNil(t, entity)

	// Validate basic properties
	require.Equal(t, "cti.test.complex.v1.0", entity.GetCTI())
	require.False(t, entity.IsFinal())
	require.Equal(t, consts.AccessModifierProtected, entity.GetAccess())

	// Validate it's an EntityType
	entityType, ok := entity.(*EntityType)
	require.True(t, ok)
	require.True(t, entityType.Resilient)
	require.Equal(t, "Complex Entity Type", entityType.DisplayName)
	require.Equal(t, "A complex entity type with nested schemas", entityType.Description)

	// Validate complex schema
	require.NotNil(t, entityType.Schema)
	require.Equal(t, "#/definitions/Person", entityType.Schema.Ref)
	require.Contains(t, entityType.Schema.Definitions, "Person")
	require.Contains(t, entityType.Schema.Definitions, "Address")

	// Validate annotations
	annotations := entity.GetAnnotations()
	require.NotNil(t, annotations)
	require.EqualValues(t, map[GJsonPath]*Annotations{
		GJsonPath(".name"):    {Schema: "string.schema"},
		GJsonPath(".address"): {Reference: "address.annotation"},
	}, annotations)

	// Validate traits schema
	require.NotNil(t, entityType.TraitsSchema)
	require.Equal(t, "object", entityType.TraitsSchema.Type)

	// Validate traits annotations
	require.EqualValues(t, map[GJsonPath]*Annotations{
		GJsonPath(".timestamps"):         {Schema: "timestamps.trait"},
		GJsonPath(".timestamps.created"): {Meta: "iso8601"},
	}, entityType.TraitsAnnotations)

	// Validate traits
	require.NotNil(t, entityType.Traits)
	timestamps, ok := entityType.Traits["timestamps"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "2023-01-01T00:00:00Z", timestamps["created"])
	require.Equal(t, "2023-01-02T00:00:00Z", timestamps["updated"])

	// Validate source map
	require.Equal(t, "ComplexType", entityType.SourceMap.Name)
	require.Equal(t, "complex/type.raml", entityType.SourceMap.SourcePath)
	require.Equal(t, "complex/original.raml", entityType.SourceMap.OriginalPath)
}

func TestConvertUntypedEntityToEntity_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		untypedEntity *testUntypedEntity
		wantErr       bool
		errContains   string
		validate      func(t *testing.T, entity Entity)
	}{
		{
			name: "entity type with empty traits and annotations",
			untypedEntity: &testUntypedEntity{
				CTI:               "cti.test.type.v1.0",
				Schema:            json.RawMessage(`{"type": "object"}`),
				Traits:            json.RawMessage(`{}`),
				TraitsAnnotations: json.RawMessage(`{}`),
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.Empty(t, entityType.Traits)
				require.Empty(t, entity.GetAnnotations())
				require.Empty(t, entityType.TraitsAnnotations)
			},
		},
		{
			name: "entity with minimal required fields only",
			untypedEntity: &testUntypedEntity{
				CTI:    "cti.minimal.v1.0",
				Schema: json.RawMessage(`{"type": "string"}`),
			},
			wantErr: false,
			validate: func(t *testing.T, entity Entity) {
				require.Equal(t, "cti.minimal.v1.0", entity.GetCTI())
				require.False(t, entity.IsFinal())
				require.Equal(t, consts.AccessModifier(""), entity.GetAccess())

				entityType, ok := entity.(*EntityType)
				require.True(t, ok)
				require.False(t, entityType.Resilient)
				require.Equal(t, "", entityType.DisplayName)
				require.Equal(t, "", entityType.Description)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := ConvertUntypedEntityToEntity(tt.untypedEntity)
			if tt.wantErr {
				require.ErrorContains(t, err, tt.errContains)
				require.Nil(t, entity)
			} else {
				require.NoError(t, err)
				require.NotNil(t, entity)
				tt.validate(t, entity)
			}
		})
	}
}

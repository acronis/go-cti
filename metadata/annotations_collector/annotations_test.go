package annotations_collector

import (
	"reflect"
	"testing"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestAnnotationsCollector_Collect_NoAnnotations(t *testing.T) {
	collector := New()
	schema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Type: "object",
			Properties: func() *orderedmap.OrderedMap[string, *jsonschema.JSONSchemaCTI] {
				props := orderedmap.New[string, *jsonschema.JSONSchemaCTI]()
				props.Set("foo", &jsonschema.JSONSchemaCTI{
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"},
					Annotations:       jsonschema.Annotations{CTICTI: "test.annotation"},
				})
				return props
			}(),
		},
	}
	require.Empty(t, collector.Collect(schema))
}

func TestAnnotationsCollector_Collect_SingleAnnotation(t *testing.T) {
	collector := New()
	schema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Type: "object",
			Properties: func() *orderedmap.OrderedMap[string, *jsonschema.JSONSchemaCTI] {
				props := orderedmap.New[string, *jsonschema.JSONSchemaCTI]()
				props.Set("foo", &jsonschema.JSONSchemaCTI{
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"},
					Annotations:       jsonschema.Annotations{CTISchema: "test.annotation"},
				})
				return props
			}(),
		},
	}
	result := collector.Collect(schema)
	require.EqualValues(t, map[metadata.GJsonPath]*metadata.Annotations{
		".foo": {Schema: "test.annotation"},
	}, result)

	expectedKey := metadata.GJsonPath(".foo")
	if _, ok := result[expectedKey]; !ok {
		t.Errorf("expected annotation at %q", expectedKey)
	}
}

func TestAnnotationsCollector_Collect_MultipleAnnotations(t *testing.T) {
	collector := New()
	schema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Type: "object",
			Properties: func() *orderedmap.OrderedMap[string, *jsonschema.JSONSchemaCTI] {
				props := orderedmap.New[string, *jsonschema.JSONSchemaCTI]()
				props.Set("bar", &jsonschema.JSONSchemaCTI{
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "integer"},
					Annotations: jsonschema.Annotations{
						CTIDisplayName: &[]bool{true}[0],
					},
				})
				return props
			}(),
		},
	}
	result := collector.Collect(schema)
	expectedKey := metadata.GJsonPath(".bar")
	ann, ok := result[expectedKey]
	if !ok {
		t.Fatalf("expected annotation at %q", expectedKey)
	}
	if ann.DisplayName == nil || *ann.DisplayName != true {
		t.Errorf("expected DisplayName annotation to be true, got %+v", ann.DisplayName)
	}
}

func TestAnnotationsCollector_Collect_UnionShape(t *testing.T) {
	collector := New()
	schema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			AnyOf: []*jsonschema.JSONSchemaCTI{
				{
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"},
					Annotations:       jsonschema.Annotations{CTIID: &[]bool{true}[0]},
				},
				{
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "integer"},
					Annotations:       jsonschema.Annotations{CTIOverridable: &[]bool{false}[0]},
				},
			},
		},
	}
	result := collector.Collect(schema)
	foundID := false
	foundOverridable := false
	for _, ann := range result {
		if ann.ID != nil && *ann.ID == true {
			foundID = true
		}
		if ann.Overridable != nil && *ann.Overridable == false {
			foundOverridable = true
		}
	}
	assert.True(t, foundID, "expected to find ID annotation with value true")
	assert.True(t, foundOverridable, "expected to find Overridable annotation with value false")
}

func TestAnnotationsCollector_Collect_ArrayShape(t *testing.T) {
	collector := New()
	schema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Type: "array",
			Items: &jsonschema.JSONSchemaCTI{
				JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"},
				Annotations:       jsonschema.Annotations{CTIID: &[]bool{true}[0]},
			},
		},
	}
	result := collector.Collect(schema)
	expectedKey := metadata.GJsonPath(".#")
	ann, ok := result[expectedKey]
	if !ok {
		t.Fatalf("expected annotation at %q", expectedKey)
	}
	if ann.ID == nil || *ann.ID != true {
		t.Errorf("expected ID annotation to be true, got %+v", ann.ID)
	}
}

func TestAnnotationsCollector_Collect_PropertyNamesAnnotation(t *testing.T) {
	collector := New()
	propertyNames := map[string]any{"foo": "bar"}
	schema := &jsonschema.JSONSchemaCTI{
		JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{
			Type: "object",
			Properties: func() *orderedmap.OrderedMap[string, *jsonschema.JSONSchemaCTI] {
				props := orderedmap.New[string, *jsonschema.JSONSchemaCTI]()
				props.Set("baz", &jsonschema.JSONSchemaCTI{
					JSONSchemaGeneric: jsonschema.JSONSchemaGeneric{Type: "string"},
					Annotations:       jsonschema.Annotations{CTIPropertyNames: propertyNames},
				})
				return props
			}(),
		},
	}
	result := collector.Collect(schema)
	expectedKey := metadata.GJsonPath(".baz")
	ann, ok := result[expectedKey]
	if !ok {
		t.Fatalf("expected annotation at %q", expectedKey)
	}
	if !reflect.DeepEqual(ann.PropertyNames, propertyNames) {
		t.Errorf("expected PropertyNames annotation to be %v, got %v", propertyNames, ann.PropertyNames)
	}
}

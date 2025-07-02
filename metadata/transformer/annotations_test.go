package transformer

import (
	"reflect"
	"testing"

	"github.com/acronis/go-cti/metadata"
)

func TestAnnotationsCollector_Collect_NoAnnotations(t *testing.T) {
	collector := NewAnnotationsCollector()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"foo": map[string]any{"type": "string"},
		},
	}
	result := collector.Collect(schema)
	if len(result) != 0 {
		t.Errorf("expected no annotations, got %d", len(result))
	}
}

func TestAnnotationsCollector_Collect_SingleAnnotation(t *testing.T) {
	collector := NewAnnotationsCollector()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"foo": map[string]any{
				"type":          "string",
				metadata.XFinal: true,
			},
		},
	}
	result := collector.Collect(schema)
	expectedKey := metadata.GJsonPath(".foo")
	if ann, ok := result[expectedKey]; !ok {
		t.Errorf("expected annotation at %q", expectedKey)
	} else if ann.Final == nil || *ann.Final != true {
		t.Errorf("expected Final annotation to be true, got %+v", ann.Final)
	}
}

func TestAnnotationsCollector_Collect_MultipleAnnotations(t *testing.T) {
	collector := NewAnnotationsCollector()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bar": map[string]any{
				"type":                "integer",
				metadata.XFinal:       false,
				metadata.XResilient:   true,
				metadata.XDisplayName: true,
			},
		},
	}
	result := collector.Collect(schema)
	expectedKey := metadata.GJsonPath(".bar")
	ann, ok := result[expectedKey]
	if !ok {
		t.Fatalf("expected annotation at %q", expectedKey)
	}
	if ann.Final == nil || *ann.Final != false {
		t.Errorf("expected Final annotation to be false, got %+v", ann.Final)
	}
	if ann.Resilient == nil || *ann.Resilient != true {
		t.Errorf("expected Resilient annotation to be true, got %+v", ann.Resilient)
	}
	if ann.DisplayName == nil || *ann.DisplayName != true {
		t.Errorf("expected DisplayName annotation to be true, got %+v", ann.DisplayName)
	}
}

func TestAnnotationsCollector_Collect_UnionShape(t *testing.T) {
	collector := NewAnnotationsCollector()
	schema := map[string]any{
		"anyOf": []any{
			map[string]any{
				"type":          "string",
				metadata.XFinal: true,
			},
			map[string]any{
				"type":              "integer",
				metadata.XResilient: false,
			},
		},
	}
	result := collector.Collect(schema)
	foundFinal := false
	foundResilient := false
	for _, ann := range result {
		if ann.Final != nil && *ann.Final == true {
			foundFinal = true
		}
		if ann.Resilient != nil && *ann.Resilient == false {
			foundResilient = true
		}
	}
	if !foundFinal {
		t.Error("expected to find Final annotation with value true")
	}
	if !foundResilient {
		t.Error("expected to find Resilient annotation with value false")
	}
}

func TestAnnotationsCollector_Collect_ArrayShape(t *testing.T) {
	collector := NewAnnotationsCollector()
	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type":          "string",
			metadata.XFinal: true,
		},
	}
	result := collector.Collect(schema)
	expectedKey := metadata.GJsonPath(".#")
	ann, ok := result[expectedKey]
	if !ok {
		t.Fatalf("expected annotation at %q", expectedKey)
	}
	if ann.Final == nil || *ann.Final != true {
		t.Errorf("expected Final annotation to be true, got %+v", ann.Final)
	}
}

func TestAnnotationsCollector_Collect_PropertyNamesAnnotation(t *testing.T) {
	collector := NewAnnotationsCollector()
	propertyNames := map[string]any{"foo": "bar"}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"baz": map[string]any{
				"type":                  "string",
				metadata.XPropertyNames: propertyNames,
			},
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

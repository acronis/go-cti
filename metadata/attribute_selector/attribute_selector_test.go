package attribute_selector

import (
	"reflect"
	"testing"

	"github.com/acronis/go-raml"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestNewAttributeSelector(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPath  []string
		wantError string
	}{
		{
			name:      "empty input",
			input:     "",
			wantPath:  []string{},
			wantError: "",
		},
		{
			name:      "only leading dot",
			input:     ".",
			wantPath:  []string{},
			wantError: "empty token at position 0",
		},
		{
			name:      "single token no dot",
			input:     "foo",
			wantPath:  []string{"foo"},
			wantError: "",
		},
		{
			name:      "multiple tokens no leading dot",
			input:     "foo.bar.baz",
			wantPath:  []string{"foo", "bar", "baz"},
			wantError: "",
		},
		{
			name:      "empty token in middle",
			input:     "foo..bar",
			wantPath:  nil,
			wantError: "empty token at position 1",
		},
		{
			name:      "empty token at end",
			input:     "foo.bar.",
			wantPath:  nil,
			wantError: "empty token at position 2",
		},
		{
			name:      "empty token at start (double leading dot)",
			input:     ".foo.bar",
			wantPath:  nil,
			wantError: "empty token at position 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as, err := NewAttributeSelector(tt.input)
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantError)
				}
				if err.Error() != tt.wantError {
					t.Fatalf("expected error %q, got %q", tt.wantError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(as.Path, tt.wantPath) {
					t.Errorf("expected path %v, got %v", tt.wantPath, as.Path)
				}
			}
		})
	}
}

func TestAttributeSelector_WalkBaseShape(t *testing.T) {
	// makeBaseObjectShape is a helper function to create a BaseShape with given properties.
	makeBaseObjectShape := func(props ...orderedmap.Pair[string, raml.Property]) *raml.BaseShape {
		return &raml.BaseShape{
			Shape: &raml.ObjectShape{
				ObjectFacets: raml.ObjectFacets{
					Properties: orderedmap.New[string, raml.Property](orderedmap.WithInitialData(props...)),
				},
			},
		}
	}

	// makeBaseNonObjectShape is a helper function to create a BaseShape that is not an object.
	makeBaseNonObjectShape := func() *raml.BaseShape {
		return &raml.BaseShape{Shape: &raml.NilShape{}}
	}

	// Compose a nested structure: foo.bar.baz
	baz := makeBaseObjectShape()
	bar := makeBaseObjectShape(
		orderedmap.Pair[string, raml.Property]{
			Key: "baz",
			Value: raml.Property{
				Base: baz,
			},
		},
	)
	foo := makeBaseObjectShape(
		orderedmap.Pair[string, raml.Property]{
			Key: "bar",
			Value: raml.Property{
				Base: bar,
			},
		},
	)

	tests := []struct {
		name      string
		selector  *AttributeSelector
		root      *raml.BaseShape
		want      *raml.BaseShape
		wantError string
	}{
		{
			name:      "single level found",
			selector:  &AttributeSelector{Path: []string{"bar"}},
			root:      foo,
			want:      bar,
			wantError: "",
		},
		{
			name:      "two levels found",
			selector:  &AttributeSelector{Path: []string{"bar", "baz"}},
			root:      foo,
			want:      baz,
			wantError: "",
		},
		{
			name:      "key not found",
			selector:  &AttributeSelector{Path: []string{"missing"}},
			root:      foo,
			want:      nil,
			wantError: `key "missing" not found`,
		},
		{
			name:      "cannot descend into non-object",
			selector:  &AttributeSelector{Path: []string{"bar", "baz"}},
			root:      makeBaseNonObjectShape(),
			want:      nil,
			wantError: `cannot descend via "bar" into *raml.BaseShape`,
		},
		{
			name:      "descend into non-object after first step",
			selector:  &AttributeSelector{Path: []string{"bar", "baz"}},
			root:      makeBaseObjectShape(orderedmap.Pair[string, raml.Property]{Key: "bar", Value: raml.Property{Base: makeBaseNonObjectShape()}}),
			want:      nil,
			wantError: `cannot descend via "baz" into *raml.BaseShape`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.selector.WalkBaseShape(tt.root)
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantError)
				}
				if err.Error() != tt.wantError {
					t.Fatalf("expected error %q, got %q", tt.wantError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("expected %v, got %v", tt.want, got)
				}
			}
		})
	}
}

package attribute_selector

import (
	"testing"

	"github.com/acronis/go-raml/v2"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestNewAttributeSelector(t *testing.T) {
	tests := map[string]struct {
		input     string
		wantPath  []string
		wantError string
	}{
		"empty input": {
			input:     "",
			wantPath:  []string{},
			wantError: "",
		},
		"only leading dot": {
			input:     ".",
			wantPath:  []string{},
			wantError: "empty token at position 0",
		},
		"single token no dot": {
			input:     "foo",
			wantPath:  []string{"foo"},
			wantError: "",
		},
		"multiple tokens no leading dot": {
			input:     "foo.bar.baz",
			wantPath:  []string{"foo", "bar", "baz"},
			wantError: "",
		},
		"empty token in middle": {
			input:     "foo..bar",
			wantPath:  nil,
			wantError: "empty token at position 1",
		},
		"empty token at end": {
			input:     "foo.bar.",
			wantPath:  nil,
			wantError: "empty token at position 2",
		},
		"empty token at start (double leading dot)": {
			input:     ".foo.bar",
			wantPath:  nil,
			wantError: "empty token at position 0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			as, err := NewAttributeSelector(tt.input)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantPath, as.Path)
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

	tests := map[string]struct {
		selector  *AttributeSelector
		root      *raml.BaseShape
		want      *raml.BaseShape
		wantError string
	}{
		"single level found": {
			selector:  &AttributeSelector{Path: []string{"bar"}},
			root:      foo,
			want:      bar,
			wantError: "",
		},
		"two levels found": {
			selector:  &AttributeSelector{Path: []string{"bar", "baz"}},
			root:      foo,
			want:      baz,
			wantError: "",
		},
		"key not found": {
			selector:  &AttributeSelector{Path: []string{"missing"}},
			root:      foo,
			want:      nil,
			wantError: `key "missing" not found`,
		},
		"cannot descend into non-object": {
			selector:  &AttributeSelector{Path: []string{"bar", "baz"}},
			root:      makeBaseNonObjectShape(),
			want:      nil,
			wantError: `cannot descend via "bar" into *raml.BaseShape`,
		},
		"descend into non-object after first step": {
			selector:  &AttributeSelector{Path: []string{"bar", "baz"}},
			root:      makeBaseObjectShape(orderedmap.Pair[string, raml.Property]{Key: "bar", Value: raml.Property{Base: makeBaseNonObjectShape()}}),
			want:      nil,
			wantError: `cannot descend via "baz" into *raml.BaseShape`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.selector.WalkBaseShape(tt.root)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

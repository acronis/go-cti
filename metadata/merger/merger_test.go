package merger

import (
	"testing"

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

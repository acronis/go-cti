package transformer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/acronis/go-cti/metadata"
	"github.com/acronis/go-cti/metadata/jsonschema"
	"github.com/acronis/go-cti/metadata/registry"
)

func newRegistryWith(t *testing.T, ctis ...string) *registry.MetadataRegistry {
	t.Helper()
	r := registry.New()
	for _, id := range ctis {
		ent, err := metadata.NewEntityType(id, &jsonschema.JSONSchemaCTI{}, nil)
		require.NoError(t, err)
		require.NoError(t, r.Add(ent))
	}
	return r
}

func TestFilterOrphanCTISchemas(t *testing.T) {
	truthy := true
	const (
		presentReq  = "cti.a.p.acgw.request.v1.1~vendor.app.v1.0"
		presentResA = "cti.a.p.acgw.response.v1.1~vendor.app.a.v1.0"
		presentResB = "cti.a.p.acgw.response.v1.1~vendor.app.b.v1.0"
		orphanA     = "cti.a.p.acgw.request.v1.1~missing.a.v1.0"
		orphanB     = "cti.a.p.acgw.request.v1.1~missing.b.v1.0"
	)

	tests := []struct {
		name    string
		present []string
		input   map[metadata.GJsonPath]*metadata.Annotations
		want    map[metadata.GJsonPath]*metadata.Annotations
	}{
		{
			name:  "orphan single string: entry dropped",
			input: map[metadata.GJsonPath]*metadata.Annotations{".request": {Schema: orphanA}},
			want:  map[metadata.GJsonPath]*metadata.Annotations{},
		},
		{
			name:    "present single string: entry kept untouched",
			present: []string{presentReq},
			input:   map[metadata.GJsonPath]*metadata.Annotations{".request": {Schema: presentReq}},
			want:    map[metadata.GJsonPath]*metadata.Annotations{".request": {Schema: presentReq}},
		},
		{
			name:  "list all orphans: entry dropped",
			input: map[metadata.GJsonPath]*metadata.Annotations{".responses.#": {Schema: []any{orphanA, orphanB}}},
			want:  map[metadata.GJsonPath]*metadata.Annotations{},
		},
		{
			name:    "list mixed with one survivor: collapses to string",
			present: []string{presentResA},
			input:   map[metadata.GJsonPath]*metadata.Annotations{".responses.#": {Schema: []any{orphanA, presentResA}}},
			want:    map[metadata.GJsonPath]*metadata.Annotations{".responses.#": {Schema: presentResA}},
		},
		{
			name:    "list mixed with multiple survivors: keeps list",
			present: []string{presentResA, presentResB},
			input:   map[metadata.GJsonPath]*metadata.Annotations{".responses.#": {Schema: []any{orphanA, presentResA, presentResB}}},
			want:    map[metadata.GJsonPath]*metadata.Annotations{".responses.#": {Schema: []any{presentResA, presentResB}}},
		},
		{
			name:  "orphan schema with other field set: schema cleared, entry kept",
			input: map[metadata.GJsonPath]*metadata.Annotations{".request": {Schema: orphanA, Overridable: &truthy}},
			want:  map[metadata.GJsonPath]*metadata.Annotations{".request": {Overridable: &truthy}},
		},
		{
			name:  "nil schema with other field set: no-op",
			input: map[metadata.GJsonPath]*metadata.Annotations{".request": {Overridable: &truthy}},
			want:  map[metadata.GJsonPath]*metadata.Annotations{".request": {Overridable: &truthy}},
		},
		{
			name:  "nil entry value: no panic, entry kept",
			input: map[metadata.GJsonPath]*metadata.Annotations{".weird": nil},
			want:  map[metadata.GJsonPath]*metadata.Annotations{".weird": nil},
		},
		{
			name:    "multiple entries handled independently",
			present: []string{presentReq},
			input: map[metadata.GJsonPath]*metadata.Annotations{
				".callback_with_request":    {Schema: presentReq},
				".callback_without_request": {Schema: orphanA},
			},
			want: map[metadata.GJsonPath]*metadata.Annotations{
				".callback_with_request": {Schema: presentReq},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := New(newRegistryWith(t, tt.present...))
			tr.filterOrphanCTISchemas(tt.input)
			require.Equal(t, tt.want, tt.input)
		})
	}
}

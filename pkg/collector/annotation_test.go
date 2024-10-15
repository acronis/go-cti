package collector

import (
	"testing"

	"github.com/acronis/go-cti/pkg/cti"
	"github.com/acronis/go-raml"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func Test_AnnotationCollector(t *testing.T) {
	type testCase struct {
		name  string
		shape raml.Shape
		fn    func(e raml.Shape) any
		want  any
	}

	testCases := []testCase{
		{
			name: "get cti.cti from object",
			shape: &raml.ObjectShape{
				BaseShape: &raml.BaseShape{
					CustomDomainProperties: orderedmap.New[string, *raml.DomainExtension](
						orderedmap.WithInitialData(
							orderedmap.Pair[string, *raml.DomainExtension]{
								Key: "cti.cti",
								Value: &raml.DomainExtension{
									Name: "cti.cti",
									Extension: &raml.Node{
										Value: "cti.vendor.app.test.v1.0",
									},
								},
							},
						),
					),
				},
			},
			fn: func(e raml.Shape) any {
				c := NewAnnotationsCollector()
				obj := e.(*raml.ObjectShape)
				c.Collect(obj)
				return c.annotations
			},
			want: map[cti.GJsonPath]cti.Annotations{".": {
				Cti: "cti.vendor.app.test.v1.0",
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fn(tc.shape)
			require.Equal(t, got, tc.want)
		})
	}
}

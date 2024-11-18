package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GJsonPathGetValue(t *testing.T) {
	type testCase struct {
		name   string
		entity *Entity
		fn     func(e *Entity) any
		want   any
	}

	testCases := []testCase{
		{
			name: "get root by .",
			entity: &Entity{
				Annotations: map[GJsonPath]Annotations{
					".": {},
				},
			},
			fn: func(e *Entity) any {
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
			entity: &Entity{
				Annotations: map[GJsonPath]Annotations{
					".val": {},
				},
			},
			fn: func(e *Entity) any {
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
			entity: &Entity{
				Annotations: map[GJsonPath]Annotations{
					".val.#": {},
				},
			},
			fn: func(e *Entity) any {
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
			entity: &Entity{
				Annotations: map[GJsonPath]Annotations{
					".val.nested.#": {},
				},
			},
			fn: func(e *Entity) any {
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
			entity: &Entity{
				Annotations: map[GJsonPath]Annotations{
					".val.arr.#": {},
				},
			},
			fn: func(e *Entity) any {
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

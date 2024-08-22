package packager

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndexJSON(t *testing.T) {
	indexJson := IndexJSON{
		Type: "cti.a.p.app.package.v1.0",
		Entities: []string{
			"entities/acronis-column/1-acronis/constructors.raml",
			"entities/acronis-column/1-acronis/dictionaries.raml",
		},
		Schema:       "",
		Apis:         nil,
		Examples:     nil,
		Assets:       nil,
		Dictionaries: nil,
	}

	actual := reflect.TypeOf(indexJson)
	expected := reflect.TypeOf(IndexJSON{})
	require.Equal(t, expected, actual)
}

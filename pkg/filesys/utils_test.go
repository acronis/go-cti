package filesys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const MockPath = "../.platform/dts/types.raml"

func TestGetBaseName(t *testing.T) {
	directory := GetBaseName(MockPath)
	require.Equal(t, directory, "types")
}

func TestGetDirName(t *testing.T) {
	directory := GetDirName(MockPath)
	require.Equal(t, directory, "dts")
}

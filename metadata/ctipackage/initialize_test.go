package ctipackage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Extract(t *testing.T) {
	require.NoError(t, extractRAMLxSpec("testdata"))
}

package packager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	packagerErr := New("errCode", "errMessage")
	require.Equal(t, packagerErr.Code, "errCode")
	require.Equal(t, packagerErr.Message, "errMessage")
}

func TestPackageError_Error(t *testing.T) {
	packagerErr := New("errCode", "errMessage")
	require.Equal(t, packagerErr.Code, "errCode")
	require.Equal(t, packagerErr.Message, "errMessage")

	err := packagerErr.Error()
	require.Equal(t, err, "errCode: errMessage")
}

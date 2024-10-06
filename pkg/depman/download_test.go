package depman

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Download(t *testing.T) {
	dm, err := New(WithStorage(&mockStorage{}), WithPackagesCache("./fixtures/_packages"))
	require.NoError(t, err)

	res, err := dm.Download(map[string]string{"mock@b1": "v1.0.0"})
	require.NoError(t, err)

	require.Len(t, res, 1)
}

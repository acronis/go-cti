package filesys

import (
	"os"
	"testing"

	"github.com/acronis/go-cti/metadata/testsupp"
	"github.com/stretchr/testify/require"
)

func Test_CopyWithReplace(t *testing.T) {
	testsupp.ManualTest(t, "replacing directories with copy")
	defer os.RemoveAll("fixtures/dir3")

	hasFile := func(name string) error {
		_, err := os.Stat(name)
		return err
	}

	require.NoError(t, ReplaceWithCopy("fixtures/dir2", "fixtures/dir3"))
	require.NoError(t, hasFile("fixtures/dir3/test2.txt"))

	require.NoError(t, ReplaceWithCopy("fixtures/dir1", "fixtures/dir2"))
	require.NoError(t, hasFile("fixtures/dir2/test1.txt"))

	require.NoError(t, ReplaceWithCopy("fixtures/dir3", "fixtures/dir2"))
	require.NoError(t, hasFile("fixtures/dir2/test2.txt"))
}

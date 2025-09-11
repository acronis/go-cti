package testsupp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type PackageTestCase struct {
	PkgId    string
	Entities []string
	Files    map[string]string
}

func InitTestPackageFiles(t *testing.T, name string, tc PackageTestCase) string {
	t.Helper()

	testDir := filepath.Join("./testdata", strings.ReplaceAll(name, " ", "_"))
	require.NoError(t, os.RemoveAll(testDir))
	require.NoError(t, os.MkdirAll(testDir, os.ModePerm))

	for name, content := range tc.Files {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(testDir, name)), os.ModePerm))
		require.NoError(t, os.WriteFile(filepath.Join(testDir, name), []byte(content), os.ModePerm))
	}

	return testDir
}

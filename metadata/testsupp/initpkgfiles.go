package testsupp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type PackageTestCase struct {
	Name     string
	PkgId    string
	Entities []string
	Files    map[string]string
}

func InitTestPackageFiles(t *testing.T, tc PackageTestCase) string {
	t.Helper()

	testDir := filepath.Join("./testdata", strings.ReplaceAll(tc.Name, " ", "_"))
	require.NoError(t, os.RemoveAll(testDir))
	require.NoError(t, os.MkdirAll(testDir, os.ModePerm))

	for name, content := range tc.Files {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(testDir, name)), os.ModePerm))
		require.NoError(t, os.WriteFile(filepath.Join(testDir, name), []byte(content), os.ModePerm))
	}

	return testDir
}

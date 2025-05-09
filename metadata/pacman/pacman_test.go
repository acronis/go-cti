package pacman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/stretchr/testify/require"
)

func Test_Add(t *testing.T) {
	type testcase struct {
		pkgId   string
		depends map[string]string
	}

	testcases := map[string]testcase{
		"single dependency": {
			pkgId:   "xyz.mock",
			depends: map[string]string{"mock@b1": "1.0.0"},
		},
		"chained dependency": {
			pkgId:   "xyz.mock",
			depends: map[string]string{"mock@b2": "0.0.0-20210101120000-abcdef123456"},
		},
		"multiple dependencies": {
			pkgId: "xyz.mock",
			depends: map[string]string{
				"mock@b1": "1.0.0",
				"mock@b3": "3.4.5",
			},
		},
	}

	for tc_name, tc := range testcases {
		t.Run(tc_name, func(t *testing.T) {
			test_dir := filepath.Join("./testdata", strings.ReplaceAll(tc_name, " ", "_"))
			require.NoError(t, os.RemoveAll(test_dir))

			cacheDir := filepath.Join(test_dir, "_cache")
			packagePath := filepath.Join(test_dir, "local")
			require.NoError(t, os.MkdirAll(packagePath, os.ModePerm))

			pm, err := New(
				WithStorage(&mockStorage{}),
				WithPackagesCache(cacheDir))
			require.NoError(t, err)

			pkg, err := ctipackage.New(packagePath,
				ctipackage.WithID(tc.pkgId))
			require.NoError(t, err)
			require.NoError(t, pkg.Initialize())

			require.NoError(t, pm.Add(pkg, tc.depends))
		})
	}
}

package depman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acronis/go-cti/pkg/bundle"
	"github.com/stretchr/testify/require"
)

func Test_Add(t *testing.T) {
	type testcase struct {
		app_code string
		depends  map[string]string
	}

	testcases := map[string]testcase{
		"single dependency": {
			app_code: "app.mock",
			depends:  map[string]string{"mock@b1": "v1.0.0"},
		},
		"chained dependency": {
			app_code: "app.mock",
			depends:  map[string]string{"mock@b2": "v0.0.0-20210101120000-abcdef123456"},
		},
		"multiple dependencies": {
			app_code: "app.mock",
			depends: map[string]string{
				"mock@b1": "v1.0.0",
				"mock@b3": "v3.4.5",
			},
		},
	}

	for tc_name, tc := range testcases {
		t.Run(tc_name, func(t *testing.T) {
			test_dir := filepath.Join("./testdata", strings.ReplaceAll(tc_name, " ", "_"))
			require.NoError(t, os.RemoveAll(test_dir))

			cacheDir := filepath.Join(test_dir, "_cache")
			bundlePath := filepath.Join(test_dir, "local")
			require.NoError(t, os.MkdirAll(bundlePath, os.ModePerm))

			dm, err := New(
				WithDownloader(&mockDownloader{}),
				WithBundlesCache(cacheDir))
			require.NoError(t, err)

			bd := bundle.New(bundlePath,
				bundle.WithAppCode(tc.app_code))
			require.NoError(t, bd.Initialize())

			require.NoError(t, dm.Add(bd, tc.depends))
		})
	}
}

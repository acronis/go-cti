package ctipackage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ValidatePackageID(t *testing.T) {
	type testcase struct {
		id    string
		valid bool
	}

	testcases := map[string]bool{
		// Valid
		"xyz.mock":    true,
		"xyz2.mock12": true,
		// Invalid
		"xyz.mock.":   false,
		".xyz.mock":   false,
		"2xyz.mock":   false,
		"xyz.2mock":   false,
		"xyz.mock@b1": false,
	}

	for tc_name, tc := range testcases {
		t.Run(tc_name, func(t *testing.T) {
			if tc {
				require.NoError(t, ValidatePackageID(tc_name))
			} else {
				require.Error(t, ValidatePackageID(tc_name))
			}
		})
	}

}

func Test_ValidateDependencyName(t *testing.T) {
	testcases := map[string]bool{
		"abc":         true,
		"abc-123":     true,
		"abc/def":     true,
		"abc_def":     true,
		"abc.def":     true,
		"abc$":        true,
		"123":         true,
		"abc-def/ghi": true,
		"abc def":     true,
		"":            false, // empty string is not valid
		"ABC":         false, // uppercase not allowed
		"abcDEF":      false, // mixed case, uppercase not allowed
	}

	for tc_name, tc := range testcases {
		t.Run(tc_name, func(t *testing.T) {
			if tc {
				require.NoError(t, ValidateDependencyName(tc_name))
			} else {
				require.Error(t, ValidateDependencyName(tc_name))
			}
		})
	}
}

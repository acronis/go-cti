package ctipackage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Id(t *testing.T) {
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
				require.NoError(t, ValidateID(tc_name))
			} else {
				require.Error(t, ValidateID(tc_name))
			}
		})
	}

}

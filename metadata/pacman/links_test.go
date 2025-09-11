package pacman

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_replaceCaptureGroup(t *testing.T) {
	for _, tc := range []struct {
		input  string
		output string
	}{
		{
			input:  "123123 .dep/123/323",
			output: "123123 ../.dep/123/323",
		},
		{
			input:  ".dep/123/dsd",
			output: "../.dep/123/dsd",
		},
		{
			input:  "123123 ../.dep/123/323",
			output: "123123 ../../.dep/123/323",
		},
		{
			input:  "123123 .dep/123/323 ../.dep/123/323",
			output: "123123 ../.dep/123/323 ../../.dep/123/323",
		},
		{
			input:  "123123 .dep123/323",
			output: "123123 .dep123/323",
		},
		{
			input:  "123123",
			output: "123123",
		},
		{
			input:  ".dep/",
			output: "../.dep/",
		},
		{
			input:  "  package_1: ../.dep/mock.package1/foo.raml",
			output: "  package_1: ../../.dep/mock.package1/foo.raml",
		},
	} {
		t.Run(tc.input, func(t *testing.T) {
			require.Equal(t, tc.output, replaceCaptureGroup(patchDepsRe, tc.input, "../.dep", 1))
		})
	}
}

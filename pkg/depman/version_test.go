package depman

import (
	"testing"

	"golang.org/x/mod/semver"
)

func checkVersion(version string) error {

	semver.IsValid(version)

	return nil
}

func Test_Version(t *testing.T) {

}

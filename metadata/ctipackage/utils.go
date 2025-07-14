package ctipackage

import (
	"fmt"
	"regexp"
)

var dependsRe = regexp.MustCompile(`^[^A-Z]+$`)
var packageIdRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}\.[a-z][a-z0-9_]{0,49}$`)

func ValidatePackageID(id string) error {
	if !packageIdRe.MatchString(id) {
		return fmt.Errorf("invalid package ID: %s, id must conform the following regex: %q", id, packageIdRe.String())
	}
	return nil
}

func ValidateDependencyName(s string) error {
	if !dependsRe.MatchString(s) {
		return fmt.Errorf("invalid dependency name: %s, name must conform the following regex: %q", s, dependsRe.String())
	}
	return nil
}

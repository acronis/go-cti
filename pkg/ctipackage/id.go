package ctipackage

import (
	"fmt"
	"regexp"
)

var packageIdRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}\.[a-z][a-z0-9_]{0,49}$`)

func ValidateID(id string) error {
	if !packageIdRe.MatchString(id) {
		return fmt.Errorf("invalid package ID: %s, id should conform following regex: %q", id, packageIdRe.String())
	}
	return nil
}

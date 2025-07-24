package packcmd

import (
	"errors"
	"strings"
)

type PackFormat string

const (
	PackFormatTgz PackFormat = "tgz"
	PackFormatZip PackFormat = "zip"
)

var ListPackFormats = []string{string(PackFormatTgz), string(PackFormatZip)}

// String is used both by fmt.Print and by Cobra in help text
func (e *PackFormat) String() string {
	return string(*e)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (e *PackFormat) Set(v string) error {
	switch v {
	case string(PackFormatTgz), string(PackFormatZip):
		*e = PackFormat(v)
		return nil
	default:
		return errors.New(`must be one of ` + strings.Join(ListPackFormats, ","))
	}
}

// Type is only used in help text
func (e *PackFormat) Type() string {
	return "packFormat"
}

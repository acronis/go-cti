package ramlx

import (
	"embed"
)

//go:embed spec_v1/*.raml
var RamlFiles embed.FS

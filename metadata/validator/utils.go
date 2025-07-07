package validator

import (
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

func MustCompileSchema(schema string) *gojsonschema.Schema {
	s, err := gojsonschema.NewSchemaLoader().Compile(gojsonschema.NewStringLoader(schema))
	if err != nil {
		panic(fmt.Errorf("failed to compile schema: %w", err))
	}
	return s
}

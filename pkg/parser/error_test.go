package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParserErrors_Error(t *testing.T) {
	parserErrors := ParserErrors{Errors: nil}

	error := parserErrors.Error()
	require.Equal(t, error, "validation errors: []")
}

func TestParserErrors_Error2(t *testing.T) {
	var errors []ParserError
	errors = append(errors, ParserError{
		Message:  "invalid inheritance",
		Location: "./acgw/types.raml",
		Level:    "1",
		Node:     "",
	})
	parserErrors := ParserErrors{Errors: errors}

	err := parserErrors.Error()
	require.Equal(t, err, "validation errors: [invalid inheritance: ./acgw/types.raml: 1: ]")
}

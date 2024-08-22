package parser

import "fmt"

type ParserError struct {
	Message  string `json:"message"`
	Location string `json:"location"`
	Level    string `json:"level,omitempty"`
	Node     string `json:"node,omitempty"`
}

type ParserErrors struct {
	Errors []ParserError `json:"errors"`
}

func (e ParserErrors) Error() string {
	errors := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		errors[i] = fmt.Sprintf("%s: %s: %s: %s", err.Message, err.Location, err.Level, err.Node)
	}
	return fmt.Sprintf("validation errors: %v", errors)
}

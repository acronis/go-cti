package execx

import "fmt"

type ExecutionError struct {
	Inner   error
	Message string
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("error executing %s: %v", e.Message, e.Inner)
}

func (e *ExecutionError) Unwrap() error {
	return e.Inner
}

func NewExecutionError(err error, msg string) error {
	return &ExecutionError{Inner: err, Message: msg}
}

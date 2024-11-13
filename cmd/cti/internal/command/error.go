package command

import (
	"fmt"
)

type Error struct {
	Inner error
	Msg   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Msg, e.Inner)
}

func (e *Error) Unwrap() error {
	return e.Inner
}

func WrapError(err error) error {
	if err == nil {
		return nil
	}

	return &Error{
		Inner: err,
		Msg:   "command failed",
	}
}

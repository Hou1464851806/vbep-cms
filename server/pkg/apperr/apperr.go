package apperr

import "fmt"

// Error is a custom error that holds structured fields for JSON logging.
type Error struct {
	Err    error
	Msg    string
	Fields []interface{}
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *Error) Unwrap() error {
	return e.Err
}

// GetFields implements the logger.FieldProvider interface
func (e *Error) GetFields() []interface{} {
	return e.Fields
}

// New creates a new structured error.
func New(err error, msg string, fields ...interface{}) error {
	return &Error{
		Err:    err,
		Msg:    msg,
		Fields: fields,
	}
}

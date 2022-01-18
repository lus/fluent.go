package parser

import "fmt"

// Error represents an error raised by the parser
type Error struct {
	Span    [2]uint
	Message string
}

// Error turns the error into a string
func (err *Error) Error() string {
	return err.Message
}

// newError creates a new error
func newError(start, end uint, msgFormat string, replacements ...interface{}) *Error {
	return &Error{
		Span:    [2]uint{start, end},
		Message: fmt.Sprintf(msgFormat, replacements...),
	}
}

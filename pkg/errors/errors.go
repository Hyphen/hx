package errors

import (
	"errors"
	"fmt"
)

// Error wraps the original error and adds a user-friendly message
type Error struct {
	OriginalError error
	UserMessage   string
}

// Error returns the user-friendly message
func (e *Error) Error() string {
	return e.UserMessage
}

// Unwrap returns the original error
func (e *Error) Unwrap() error {
	return e.OriginalError
}

// New creates a new Error with just a user message
func New(userMessage string) *Error {
	return &Error{
		UserMessage: userMessage,
	}
}

// Wrap creates a new Error that wraps an existing error with a user-friendly message
func Wrap(err error, userMessage string) *Error {
	return &Error{
		OriginalError: err,
		UserMessage:   userMessage,
	}
}

// Wrapf creates a new Error that wraps an existing error with a formatted user-friendly message
func Wrapf(err error, format string, args ...interface{}) *Error {
	return &Error{
		OriginalError: err,
		UserMessage:   fmt.Sprintf(format, args...),
	}
}

// Is reports whether any error in err's chain matches target.
func (e *Error) Is(target error) bool {
	if target == nil {
		return e == target
	}

	err, ok := target.(*Error)
	if !ok {
		return errors.Is(e.OriginalError, target)
	}

	return e.UserMessage == err.UserMessage
}

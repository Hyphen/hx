package errors

import (
	"errors"
	"fmt"
	"io"
	"net/http"
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

func HandleHTTPError(resp *http.Response) *Error {
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusBadRequest:
		return Wrapf(New("BadRequest"), "bad request: %s", string(body))
	case http.StatusUnauthorized:
		return New("unauthorized: please check your credentials")
	case http.StatusForbidden:
		return New("forbidden: you don't have permission to perform this action")
	case http.StatusNotFound:
		return New("not found: the requested resource does not exist")
	case http.StatusConflict:
		return Wrapf(New("Conflict"), "conflict: %s", string(body))
	case http.StatusTooManyRequests:
		return New("rate limit exceeded: please try again later")
	case http.StatusInternalServerError:
		return New("internal server error: please try again later")
	default:
		return Wrapf(New("UnexpectedError"), "unexpected error (status code %d): %s", resp.StatusCode, string(body))
	}
}

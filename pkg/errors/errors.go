package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Sentinel errors
var (
	ErrBadRequest          = New("BadRequest")
	ErrUnauthorized        = New("Unauthorized")
	ErrForbidden           = New("Forbidden")
	ErrNotFound            = New("NotFound")
	ErrConflict            = New("Conflict")
	ErrTooManyRequests     = New("TooManyRequests")
	ErrInternalServerError = New("InternalServerError")
	ErrUnexpected          = New("UnexpectedError")
	ErrGeneric             = New("GenericError")
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

// Is reports whether any error in err's chain matches target.
// This function is similar to the standard errors.Is, but works with our custom Error type.
func Is(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	for {
		if customErr, ok := err.(*Error); ok {
			if customErr.Is(target) {
				return true
			}
			err = customErr.Unwrap()
		} else {
			return errors.Is(err, target)
		}

		if err == nil {
			return false
		}
	}
}

func HandleHTTPError(resp *http.Response) *Error {
	body, _ := io.ReadAll(resp.Body)
	errorMessage := body

	// attempt to decode the response body as key/value JSON to see if we have a message field to use
	var responseBody map[string]string
	if err := json.Unmarshal(body, &responseBody); err == nil {
		if msg, ok := responseBody["message"]; ok {
			errorMessage = []byte(msg)
		}
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		return Wrapf(ErrBadRequest, "bad request: %s", errorMessage)
	case http.StatusUnauthorized:
		return Wrap(ErrUnauthorized, "unauthorized: please authenticate with `auth` and try again")
	case http.StatusForbidden:
		return Wrap(ErrForbidden, "forbidden: you don't have permission to perform this action")
	case http.StatusNotFound:
		return Wrapf(ErrNotFound, "not found: %s", errorMessage)
	case http.StatusConflict:
		return Wrapf(ErrConflict, "conflict: %s", errorMessage)
	case http.StatusTooManyRequests:
		return Wrap(ErrTooManyRequests, "rate limit exceeded: please try again later")
	case http.StatusInternalServerError:
		return Wrap(ErrInternalServerError, "internal server error: please try again later")
	default:
		return Wrapf(ErrUnexpected, "unexpected error (status code %d): %s", resp.StatusCode, errorMessage)
	}
}

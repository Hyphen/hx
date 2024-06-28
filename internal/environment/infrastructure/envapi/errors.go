package envapi

// Error wraps the original error and adds a user-friendly message
type Error struct {
	OriginalError error
	UserMessage   string
}

func (e *Error) Error() string {
	return e.UserMessage
}

func WrapError(err error, userMessage string) error {
	return &Error{
		OriginalError: err,
		UserMessage:   userMessage,
	}
}

func (e *Error) Unwrap() error {
	return e.OriginalError
}

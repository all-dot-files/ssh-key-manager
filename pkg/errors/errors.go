package errors

import (
	"fmt"
)

// Standard error codes
const (
	ErrInternal     = "INTERNAL"
	ErrNotFound     = "NOT_FOUND"
	ErrInvalidInput = "INVALID_INPUT"
	ErrUnauthorized = "UNAUTHORIZED"
	ErrConflict     = "CONFLICT"
)

// AppError is a standardized error type for the application
type AppError struct {
	Code       string
	Message    string
	Op         string // Operation where the error occurred
	Err        error  // Underlying error
	Suggestion string // Actionable suggestion for the user
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s (cause: %v)", e.Code, e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Op, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(code, op, message string) *AppError {
	return &AppError{
		Code:    code,
		Op:      op,
		Message: message,
	}
}

// Wrap wraps an existing error into an AppError
func Wrap(err error, code, op, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// WrapWithSuggestion wraps an existing error with a suggestion
func WrapWithSuggestion(err error, code, op, message, suggestion string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:       code,
		Op:         op,
		Message:    message,
		Err:        err,
		Suggestion: suggestion,
	}
}

// WithSuggestion adds a suggestion to an existing AppError
func (e *AppError) WithSuggestion(suggestion string) *AppError {
	e.Suggestion = suggestion
	return e
}

// IsCode checks if the error has the specific code
func IsCode(err error, code string) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

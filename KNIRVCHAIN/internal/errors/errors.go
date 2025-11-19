package errors

import (
	"fmt"
)

// AppError represents a custom application error
type AppError struct {
	Message string
	Err     error
}

// Error returns the error message
func (e AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e AppError) Unwrap() error {
	return e.Err
}

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
func New(message string) error {
	return &Error{message: message}
}

type Error struct {
	message string
}

func (e *Error) Error() string {
	return e.message
}

// Define specific error types
var InvalidConfigError = AppError{Message: "Config is invalid"}
var BoltDBError = AppError{Message: "BoltDB Error"}
var FileNotFoundError = AppError{Message: "File not found"}
var InvalidURIError = AppError{Message: "Invalid URI"}
var agentError = AppError{Message: "agent error"}

// Blockchain specific errors
var ErrBlockNotFound = AppError{Message: "block not found"}
var ErrInvalidSignature = AppError{Message: "invalid signature"}
var ErrInvalidBlock = AppError{Message: "invalid block"}
var ErrInvalidTransaction = AppError{Message: "invalid transaction"}

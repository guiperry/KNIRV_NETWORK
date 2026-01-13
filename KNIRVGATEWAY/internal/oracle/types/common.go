package types

import "errors"

// Common errors
var (
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInvalidChainID      = errors.New("invalid chain ID")
	ErrTransferFailed      = errors.New("transfer failed")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrNotFound            = errors.New("not found")
	ErrAlreadyExists       = errors.New("already exists")
	ErrInvalidProposal     = errors.New("invalid proposal")
	ErrProposalNotFound    = errors.New("proposal not found")
	ErrVotingClosed        = errors.New("voting period closed")
	ErrInvalidVote         = errors.New("invalid vote")
	ErrChannelNotFound     = errors.New("IBC channel not found")
	ErrConnectionFailed    = errors.New("connection failed")
	ErrTimeout             = errors.New("operation timed out")
)

// Result is a generic result type
type Result[T any] struct {
	Value T
	Error error
}

// Ok creates a successful Result
func Ok[T any](value T) Result[T] {
	return Result[T]{Value: value, Error: nil}
}

// Err creates a failed Result
func Err[T any](err error) Result[T] {
	var zero T
	return Result[T]{Value: zero, Error: err}
}

// IsOk returns true if the result is successful
func (r Result[T]) IsOk() bool {
	return r.Error == nil
}

// IsErr returns true if the result is an error
func (r Result[T]) IsErr() bool {
	return r.Error != nil
}

// Unwrap returns the value or panics if there's an error
func (r Result[T]) Unwrap() T {
	if r.Error != nil {
		panic(r.Error)
	}
	return r.Value
}

// UnwrapOr returns the value or the provided default
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.Error != nil {
		return defaultValue
	}
	return r.Value
}

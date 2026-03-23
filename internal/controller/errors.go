package controller

import "fmt"

// RetryableError represents an error that the controller should retry.
type RetryableError struct {
	Err    error
	Reason string
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("%s: %v", e.Reason, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// PermanentError represents an error that should not be retried.
type PermanentError struct {
	Err    error
	Reason string
}

func (e *PermanentError) Error() string {
	return fmt.Sprintf("%s: %v", e.Reason, e.Err)
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error.
func NewRetryableError(reason string, err error) *RetryableError {
	return &RetryableError{Reason: reason, Err: err}
}

// NewPermanentError creates a new permanent (non-retryable) error.
func NewPermanentError(reason string, err error) *PermanentError {
	return &PermanentError{Reason: reason, Err: err}
}

// IsRetryable returns true if the error should be retried.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*RetryableError)
	return ok
}

// IsPermanent returns true if the error should not be retried.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*PermanentError)
	return ok
}

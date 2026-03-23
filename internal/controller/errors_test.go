package controller

import (
	"errors"
	"testing"
)

func TestRetryableError(t *testing.T) {
	inner := errors.New("connection refused")
	err := NewRetryableError("IP allocation failed", inner)

	if err.Error() != "IP allocation failed: connection refused" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !errors.Is(err, inner) {
		t.Error("expected Unwrap to return inner error")
	}

	if !IsRetryable(err) {
		t.Error("expected IsRetryable to return true")
	}

	if IsPermanent(err) {
		t.Error("expected IsPermanent to return false for retryable error")
	}
}

func TestPermanentError(t *testing.T) {
	inner := errors.New("invalid IP range format")
	err := NewPermanentError("validation failed", inner)

	if err.Error() != "validation failed: invalid IP range format" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !errors.Is(err, inner) {
		t.Error("expected Unwrap to return inner error")
	}

	if !IsPermanent(err) {
		t.Error("expected IsPermanent to return true")
	}

	if IsRetryable(err) {
		t.Error("expected IsRetryable to return false for permanent error")
	}
}

func TestNilErrors(t *testing.T) {
	if IsRetryable(nil) {
		t.Error("expected IsRetryable(nil) to return false")
	}
	if IsPermanent(nil) {
		t.Error("expected IsPermanent(nil) to return false")
	}
}

func TestRegularError(t *testing.T) {
	err := errors.New("some error")
	if IsRetryable(err) {
		t.Error("expected IsRetryable to return false for regular error")
	}
	if IsPermanent(err) {
		t.Error("expected IsPermanent to return false for regular error")
	}
}

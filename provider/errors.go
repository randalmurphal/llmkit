package provider

import (
	"errors"
	"fmt"
)

// Sentinel errors for provider operations.
var (
	// ErrUnknownProvider indicates the requested provider is not registered.
	ErrUnknownProvider = errors.New("unknown provider")

	// ErrUnavailable indicates the LLM service is unavailable.
	ErrUnavailable = errors.New("LLM service unavailable")

	// ErrContextTooLong indicates the input exceeds the context window.
	ErrContextTooLong = errors.New("context exceeds maximum length")

	// ErrRateLimited indicates the request was rate limited.
	ErrRateLimited = errors.New("rate limited")

	// ErrInvalidRequest indicates the request is malformed.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrTimeout indicates the request timed out.
	ErrTimeout = errors.New("request timed out")

	// ErrCLINotFound indicates the CLI binary was not found in PATH.
	ErrCLINotFound = errors.New("CLI binary not found")

	// ErrCredentialsNotFound indicates credentials are missing.
	ErrCredentialsNotFound = errors.New("credentials not found")

	// ErrCredentialsExpired indicates credentials have expired.
	ErrCredentialsExpired = errors.New("credentials expired")

	// ErrCapabilityNotSupported indicates the provider doesn't support the requested capability.
	ErrCapabilityNotSupported = errors.New("capability not supported by provider")
)

// Error wraps provider errors with context.
type Error struct {
	Provider  string // Provider name ("claude", "gemini", etc.)
	Op        string // Operation that failed ("complete", "stream")
	Err       error  // Underlying error
	Retryable bool   // Whether the error is likely transient
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("%s %s: %v", e.Provider, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new provider error.
func NewError(provider, op string, err error, retryable bool) *Error {
	return &Error{
		Provider:  provider,
		Op:        op,
		Err:       err,
		Retryable: retryable,
	}
}

// IsRetryable checks if an error is likely transient and worth retrying.
func IsRetryable(err error) bool {
	var provErr *Error
	if errors.As(err, &provErr) {
		return provErr.Retryable
	}

	// Check for known retryable sentinel errors
	return errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrUnavailable) ||
		errors.Is(err, ErrTimeout)
}

// IsCapabilityError checks if an error is due to missing provider capability.
func IsCapabilityError(err error) bool {
	return errors.Is(err, ErrCapabilityNotSupported)
}

// IsAuthError checks if an error is authentication-related.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrCredentialsNotFound) ||
		errors.Is(err, ErrCredentialsExpired)
}

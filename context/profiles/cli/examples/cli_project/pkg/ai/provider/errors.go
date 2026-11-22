package provider

import "time"

// TransientError signals a failure that callers may retry. Category carries
// a short label describing the underlying cause (e.g., "network",
// "provider_error").
type TransientError struct {
	Err      error
	Category string
}

// Error implements error by prefixing the category when present.
func (e *TransientError) Error() string {
	if e == nil || e.Err == nil {
		return "transient error"
	}
	if e.Category != "" {
		return "transient[" + e.Category + "]: " + e.Err.Error()
	}
	return "transient: " + e.Err.Error()
}

// Unwrap returns the wrapped error for errors.Is / errors.As.
func (e *TransientError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// PermanentError signals a failure that retry will not resolve. Category
// carries a short label describing the underlying cause (e.g., "validation",
// "auth").
type PermanentError struct {
	Err      error
	Category string
}

// Error implements error by prefixing the category when present.
func (e *PermanentError) Error() string {
	if e == nil || e.Err == nil {
		return "permanent error"
	}
	if e.Category != "" {
		return "permanent[" + e.Category + "]: " + e.Err.Error()
	}
	return "permanent: " + e.Err.Error()
}

// Unwrap returns the wrapped error for errors.Is / errors.As.
func (e *PermanentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// RateLimitError signals the provider refused a request because the rate
// limit has been reached. RetryAfter carries the minimum delay before
// retrying. A zero RetryAfter means the caller must choose its own backoff.
type RateLimitError struct {
	Err        error
	RetryAfter time.Duration
}

// Error implements error and surfaces the retry-after hint when present.
func (e *RateLimitError) Error() string {
	if e == nil {
		return "rate limit error"
	}
	msg := "rate limited"
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	if e.RetryAfter > 0 {
		msg += " (retry after " + e.RetryAfter.String() + ")"
	}
	return msg
}

// Unwrap returns the wrapped error for errors.Is / errors.As.
func (e *RateLimitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

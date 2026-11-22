// Package retry provides a typed retry executor with configurable backoff,
// jitter strategies, error classification, Retry-After parsing, and a
// sequential try-each-in-order helper.
package retry

import "time"

// ErrorClassification labels a failure so retry logic can decide whether to
// back off, give up, or propagate the error.
type ErrorClassification string

const (
	// ClassRateLimit indicates the caller exceeded a provider quota (HTTP 429).
	ClassRateLimit ErrorClassification = "rate_limit"
	// ClassServerError indicates an upstream 5xx response; retry is useful.
	ClassServerError ErrorClassification = "server_error"
	// ClassTimeout indicates an HTTP 408 or an I/O timeout.
	ClassTimeout ErrorClassification = "timeout"
	// ClassNetwork indicates a transport-level failure (DNS, connection reset).
	ClassNetwork ErrorClassification = "network_error"
	// ClassClient indicates an HTTP 4xx response (excluding 408/429); not retried.
	ClassClient ErrorClassification = "client_error"
	// ClassCancelled indicates context.Canceled; propagated, not retried.
	ClassCancelled ErrorClassification = "canceled"
	// ClassDeadline indicates context.DeadlineExceeded; propagated, not retried.
	ClassDeadline ErrorClassification = "deadline"
)

// RetryConfig controls the retry loop timing and termination thresholds.
type RetryConfig struct {
	// MaxDuration bounds the total time spent across attempts.
	MaxDuration time.Duration
	// InitialDelay is the base delay before the second attempt.
	InitialDelay time.Duration
	// MaxDelay caps the delay per attempt after backoff expansion.
	MaxDelay time.Duration
	// MaxRetries bounds the number of retries after the initial attempt.
	MaxRetries int
	// Jitter selects how random noise is applied to each delay.
	Jitter JitterStrategy
}

// JitterStrategy selects the delay randomization policy.
type JitterStrategy string

const (
	// JitterFull randomizes the delay uniformly in [0, exponentialCap].
	// AWS-recommended default; spreads contention across retrying clients.
	JitterFull JitterStrategy = "full"
	// JitterGradual maps attempt bands to widening ranges; hand-tuned for
	// long-running batch loads where early retries should be quick.
	JitterGradual JitterStrategy = "gradual"
	// JitterNone applies pure exponential backoff with no randomness.
	JitterNone JitterStrategy = "none"
)

// RetryResult captures the outcome of a Retry call. Value is the zero value
// of T when Err is non-nil.
type RetryResult[T any] struct {
	Value T
	Err   error
}

// ClassifiedError is implemented by errors that know their own retry
// classification and Retry-After hint. The classifier and Retry-After parser
// check this interface before falling back to heuristics.
type ClassifiedError interface {
	error
	// Classification reports the error category used by the retry loop.
	Classification() ErrorClassification
	// RetryAfter returns a server-supplied delay hint. Zero means no hint.
	RetryAfter() time.Duration
}

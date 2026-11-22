package retry

import (
	"errors"
	"regexp"
	"strconv"
	"time"

	"example.com/cliproj/pkg/ai/provider"
)

// retryAfterSecondsRE captures plain-integer Retry-After style hints
// embedded in error messages. Matches patterns like "retry after 30" or
// "retry after 30 seconds" (case-insensitive via the (?i) flag).
var retryAfterSecondsRE = regexp.MustCompile(`(?i)retry[\s_-]?after[:\s]+(\d+)`)

// retryAfterHeaderRE matches explicit HTTP header-style hints like
// "Retry-After: 30" or "retry-after=30".
var retryAfterHeaderRE = regexp.MustCompile(`(?i)retry-after[:\s=]+(\d+)`)

// parseRetryAfter extracts a Retry-After hint from err. The function checks,
// in order:
//  1. ClassifiedError.RetryAfter() if implemented.
//  2. provider.RateLimitError.RetryAfter.
//  3. HTTP Retry-After header echoed in the error message.
//  4. Free-text "retry after N seconds" patterns.
//
// Returns 0 when no hint is present.
func parseRetryAfter(err error) time.Duration {
	if err == nil {
		return 0
	}

	var classified ClassifiedError
	if errors.As(err, &classified) {
		if d := classified.RetryAfter(); d > 0 {
			return d
		}
	}

	var rateLimitErr *provider.RateLimitError
	if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > 0 {
		return rateLimitErr.RetryAfter
	}

	msg := err.Error()
	if m := retryAfterHeaderRE.FindStringSubmatch(msg); len(m) == 2 {
		if secs, parseErr := strconv.Atoi(m[1]); parseErr == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	if m := retryAfterSecondsRE.FindStringSubmatch(msg); len(m) == 2 {
		if secs, parseErr := strconv.Atoi(m[1]); parseErr == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 0
}

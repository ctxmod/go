package retry

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"syscall"

	"example.com/cliproj/pkg/ai/provider"
)

// classifyError maps an error to one of the defined ErrorClassification
// labels. The classifier inspects (in order):
//  1. Context sentinels (Canceled, DeadlineExceeded).
//  2. Explicit ClassifiedError implementations.
//  3. Known provider error shapes (RateLimitError, PermanentError,
//     TransientError) -- TransientError.Category is respected.
//  4. Network error types (syscall.Errno, net.OpError, net.Error timeouts).
//  5. HTTP status codes embedded in error messages.
//
// Unknown errors fall through to ClassClient (non-retriable). Programming
// bugs (nil pointers, type assertions, etc.) must not be retried.
func classifyError(err error) ErrorClassification {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return ClassCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ClassDeadline
	}

	if classified, ok := errors.AsType[ClassifiedError](err); ok {
		return classified.Classification()
	}

	if _, ok := errors.AsType[*provider.RateLimitError](err); ok {
		return ClassRateLimit
	}
	if _, ok := errors.AsType[*provider.PermanentError](err); ok {
		return ClassClient
	}
	if transientErr, ok := errors.AsType[*provider.TransientError](err); ok {
		return classifyTransientCategory(transientErr.Category)
	}

	// Network-layer errors.
	if errno := extractErrno(err); errno != 0 {
		switch errno {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.EHOSTUNREACH,
			syscall.ENETUNREACH, syscall.EPIPE:
			return ClassNetwork
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ClassTimeout
	}
	if _, ok := errors.AsType[*net.OpError](err); ok {
		return ClassNetwork
	}
	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return ClassNetwork
	}
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		if urlErr.Timeout() {
			return ClassTimeout
		}
		return ClassNetwork
	}

	// Fallback: HTTP-status heuristics against the error message. Upstream
	// SDKs commonly format "status code: 429" or "429 Too Many Requests".
	msg := strings.ToLower(err.Error())
	switch {
	case containsStatus(msg, "429") || strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "rate limit"):
		return ClassRateLimit
	case containsStatus(msg, "408") || strings.Contains(msg, "request timeout"):
		return ClassTimeout
	case containsStatus(msg, "502"), containsStatus(msg, "503"), containsStatus(msg, "504"):
		return ClassServerError
	case containsStatus(msg, "400"), containsStatus(msg, "401"), containsStatus(msg, "403"),
		containsStatus(msg, "404"), containsStatus(msg, "422"):
		return ClassClient
	}

	// Unknown errors are treated as client-side (non-retriable). Programming
	// bugs should never trigger retry.
	return ClassClient
}

// isRetryable reports whether a classified error warrants another attempt.
func isRetryable(class ErrorClassification) bool {
	switch class {
	case ClassRateLimit, ClassServerError, ClassTimeout, ClassNetwork:
		return true
	default:
		return false
	}
}

// classifyTransientCategory maps a TransientError.Category string to an
// ErrorClassification. Unknown categories default to ClassServerError
// (retriable) since TransientError already signals the error is temporary.
func classifyTransientCategory(cat string) ErrorClassification {
	switch cat {
	case "rate_limit":
		return ClassRateLimit
	case "network":
		return ClassNetwork
	case "timeout":
		return ClassTimeout
	default:
		return ClassServerError
	}
}

// extractErrno walks the error chain for a syscall.Errno.
func extractErrno(err error) syscall.Errno {
	if errno, ok := errors.AsType[syscall.Errno](err); ok {
		return errno
	}
	return 0
}

// containsStatus reports whether msg contains code as a standalone token,
// avoiding false matches against substrings like "1429" or "42940".
func containsStatus(msg, code string) bool {
	idx := strings.Index(msg, code)
	if idx < 0 {
		return false
	}
	// Check left boundary.
	if idx > 0 {
		prev := msg[idx-1]
		if prev >= '0' && prev <= '9' {
			return false
		}
	}
	// Check right boundary.
	end := idx + len(code)
	if end < len(msg) {
		next := msg[end]
		if next >= '0' && next <= '9' {
			return false
		}
	}
	return true
}

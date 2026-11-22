package retry

import (
	"math/rand/v2"
	"time"
)

// calculateDelay produces the wait duration for the given attempt index,
// where attempt=0 is the delay before the first retry (i.e. after the
// initial attempt failed). The result is bounded by config.MaxDelay.
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	// exponentialCap = min(InitialDelay * 2^attempt, MaxDelay)
	exponentialCap := exponentialBackoff(config.InitialDelay, attempt, config.MaxDelay)

	switch config.Jitter {
	case JitterFull:
		// AWS-recommended full jitter: uniform in [0, exponentialCap].
		if exponentialCap <= 0 {
			return 0
		}
		return time.Duration(rand.Int64N(int64(exponentialCap) + 1)) //nolint:gosec // G404: math/rand is intentional for retry jitter, not cryptographic use
	case JitterGradual:
		return gradualJitter(attempt)
	case JitterNone:
		return exponentialCap
	default:
		return exponentialCap
	}
}

// exponentialBackoff computes initial * 2^attempt, capped at max. Uses shift
// arithmetic to avoid float conversion and handles overflow by clamping to
// max once the shift crosses the underlying int64 boundary.
func exponentialBackoff(initial time.Duration, attempt int, max time.Duration) time.Duration {
	if initial <= 0 {
		return 0
	}
	// Cap attempt at 62 to avoid int64 shift UB.
	if attempt > 62 {
		return max
	}
	scaled := int64(initial) << uint(attempt)
	// Overflow detection: a left-shift that loses high bits yields a value
	// less than the original when reinterpreted.
	if scaled < int64(initial) || time.Duration(scaled) > max {
		return max
	}
	return time.Duration(scaled)
}

// gradualJitter maps attempt bands to widening uniform ranges. Tuned for
// long-running batch workloads where early retries should be quick and
// later retries should space out.
func gradualJitter(attempt int) time.Duration {
	var lo, hi time.Duration
	switch {
	case attempt <= 4:
		lo, hi = 2*time.Second, 4*time.Second
	case attempt <= 19:
		lo, hi = 4*time.Second, 10*time.Second
	default:
		lo, hi = 10*time.Second, 20*time.Second
	}
	return lo + time.Duration(rand.Int64N(int64(hi-lo)+1)) //nolint:gosec // G404: math/rand is intentional for retry jitter, not cryptographic use
}

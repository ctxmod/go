package retry

import (
	"context"
	"time"

	govretry "github.com/sethvargo/go-retry"
)

// Retry executes fn with the given configuration, returning the first
// success or the last error after all attempts are exhausted. The retry
// loop:
//  1. Classifies each error via classifyError.
//  2. Stops immediately on non-retryable classifications (canceled,
//     deadline, client errors) and propagates the underlying error.
//  3. Computes the next delay via calculateDelay, overriding with the
//     Retry-After hint if the server supplied one that exceeds the
//     computed delay.
//  4. Honors context cancellation between attempts.
//
// The go-retry package provides the outer attempt loop; we compute delays
// per-attempt ourselves via a BackoffFunc wrapper so that jitter and
// Retry-After integrate cleanly.
func Retry[T any](ctx context.Context, config RetryConfig, fn func(ctx context.Context) (T, error)) RetryResult[T] {
	var lastHint time.Duration
	attempt := 0

	backoff := govretry.BackoffFunc(func() (time.Duration, bool) {
		delay := calculateDelay(attempt, config)
		attempt++
		// Honor server Retry-After if it exceeds our computed delay.
		if lastHint > delay {
			delay = lastHint
		}
		if delay > config.MaxDelay && config.MaxDelay > 0 {
			// Cap extreme Retry-After hints at MaxDelay+hint itself; preserve
			// server instruction when larger, but keep jitter from exploding
			// past intentional bounds.
			if lastHint <= 0 {
				delay = config.MaxDelay
			}
		}
		lastHint = 0
		return delay, false
	})

	var b govretry.Backoff = backoff
	if config.MaxRetries > 0 {
		b = govretry.WithMaxRetries(uint64(config.MaxRetries), b)
	}
	if config.MaxDuration > 0 {
		b = govretry.WithMaxDuration(config.MaxDuration, b)
	}

	value, err := govretry.DoValue(ctx, b, func(ctx context.Context) (T, error) {
		v, err := fn(ctx)
		if err == nil {
			return v, nil
		}
		class := classifyError(err)
		if !isRetryable(class) {
			// Propagate without retry wrapping.
			return v, err
		}
		lastHint = parseRetryAfter(err)
		return v, govretry.RetryableError(err)
	})

	return RetryResult[T]{Value: value, Err: err}
}

package retry

import (
	"context"
	"errors"
	"fmt"
)

// TryEach walks items in order, invoking fn for each. The first successful
// call (nil error) returns its value immediately. If fn returns a
// non-retryable classification (context cancellation, deadline, or a
// client-side permanent error), TryEach returns that error without trying
// further items. When every call returns a retryable error, TryEach
// returns errors.Join of them so callers can inspect every failure.
//
// Early termination (cancellation, deadline, or client error) labels each
// attempted item with its zero-based index ("item 0: ...", "item 1: ...").
// Items that were never reached are not represented in the returned error.
// Callers can determine the count of skipped items by comparing the number
// of joined errors against len(items).
//
// A zero items slice returns the zero value of R and an "empty items"
// error; callers are expected to pre-validate input when that condition
// is unexpected.
func TryEach[T, R any](ctx context.Context, items []T, fn func(ctx context.Context, item T) (R, error)) (R, error) {
	var zero R
	if len(items) == 0 {
		return zero, errors.New("try each: empty items")
	}

	errs := make([]error, 0, len(items))
	for i, item := range items {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			return zero, errors.Join(errs...)
		}
		value, err := fn(ctx, item)
		if err == nil {
			return value, nil
		}
		class := classifyError(err)
		// Halt early on non-retryable classifications: continuing the chain
		// would silently swallow cancellations or client validation errors.
		if class == ClassCancelled || class == ClassDeadline {
			errs = append(errs, err)
			return zero, errors.Join(errs...)
		}
		if class == ClassClient {
			errs = append(errs, fmt.Errorf("item %d: %w", i, err))
			return zero, errors.Join(errs...)
		}
		errs = append(errs, fmt.Errorf("item %d: %w", i, err))
	}
	return zero, errors.Join(errs...)
}

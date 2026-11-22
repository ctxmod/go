package retry

import "time"

// DEFAULT is the standard retry profile for interactive provider calls.
// Two hours of wall budget accommodates long queues behind rate limits; the
// three-retry cap prevents infinite loops on misclassified permanent errors.
var DEFAULT = RetryConfig{
	MaxDuration:  2 * time.Hour,
	InitialDelay: 2 * time.Second,
	MaxDelay:     20 * time.Second,
	MaxRetries:   3,
	Jitter:       JitterFull,
}

// conservative trades responsiveness for persistence: more retries spread
// over a longer window, with gradual jitter to avoid synchronized retry
// storms on shared providers.
var conservative = RetryConfig{
	MaxDuration:  1 * time.Hour,
	InitialDelay: 3 * time.Second,
	MaxDelay:     60 * time.Second,
	MaxRetries:   5,
	Jitter:       JitterGradual,
}

// SPEEDY is the interactive/TTY profile: fail fast so humans aren't stuck
// waiting on transient upstream glitches. Deterministic delays keep the
// behavior predictable when a developer re-runs a command.
var SPEEDY = RetryConfig{
	MaxDuration:  30 * time.Second,
	InitialDelay: 500 * time.Millisecond,
	MaxDelay:     5 * time.Second,
	MaxRetries:   2,
	Jitter:       JitterNone,
}

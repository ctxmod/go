package demo

import (
	"context"
	"time"

	"example.com/cliproj/pkg/runtime"
)

// Cmd simulates a multi-phase operation that exercises every output
// surface: progress updates with embedded percentages, informational and
// checkpoint messages, a final result payload, and the ndjson envelope.
type Cmd struct {
	Fast bool `help:"Use shorter sleeps for quick iteration." short:"f"`
}

// sleep waits for duration d or until ctx is canceled, whichever comes
// first. Replaces time.Sleep so SIGINT interrupts the demo between phases.
func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

func (c *Cmd) Run(ctx context.Context) error {
	rt, err := runtime.FromContext(ctx)
	if err != nil {
		return runtime.Exitf(runtime.ExitSoftware, "runtime context: %v", err)
	}

	step := 500 * time.Millisecond
	if c.Fast {
		step = 50 * time.Millisecond
	}

	// Phase 1: Initializing.
	rt.Progress("Initializing 0%%")
	sleep(ctx, step)
	rt.Progress("Initializing 50%%")
	sleep(ctx, step)
	rt.Info("loaded configuration")
	rt.Progress("Initializing 100%%")
	sleep(ctx, step)

	// Phase 2: Downloading.
	rt.Progress("Downloading 0%%")
	sleep(ctx, step)
	rt.Progress("Downloading 25%%")
	sleep(ctx, step)
	rt.Progress("Downloading 50%%")
	sleep(ctx, step)
	rt.Checkpoint("download", "retrieved remote artifacts")
	rt.Progress("Downloading 75%%")
	sleep(ctx, step)
	rt.Progress("Downloading 100%%")
	sleep(ctx, step)

	// Phase 3: Processing.
	rt.Progress("Processing 0%%")
	sleep(ctx, step)
	rt.Progress("Processing 33%%")
	sleep(ctx, step)
	rt.Info("transformed 1024 records")
	rt.Progress("Processing 66%%")
	sleep(ctx, step)
	rt.Progress("Processing 100%%")
	sleep(ctx, step)

	// Phase 4: Finalizing.
	rt.Progress("Finalizing 0%%")
	sleep(ctx, step)
	rt.Progress("Finalizing 100%%")
	sleep(ctx, step)
	rt.Success("demo completed across 4 phases")

	result := map[string]string{
		"phases": "4",
		"status": "complete",
	}
	rt.Result(result)
	return nil
}

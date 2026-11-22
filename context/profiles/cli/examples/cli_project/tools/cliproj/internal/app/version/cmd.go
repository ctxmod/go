package version

import (
	"context"

	"example.com/cliproj/pkg/runtime"
)

// Cmd prints version information.
type Cmd struct{}

func (c *Cmd) Run(ctx context.Context) error {
	rt, err := runtime.FromContext(ctx)
	if err != nil {
		return runtime.Exitf(runtime.ExitSoftware, "runtime context: %v", err)
	}
	rt.Result(map[string]string{
		"version": runtime.BuildVersion,
		"commit":  runtime.BuildCommit,
	})
	return nil
}

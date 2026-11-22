package list

import (
	"context"

	"example.com/cliproj/pkg/runtime"
)

// Cmd retrieves and displays items.
type Cmd struct {
	All bool `help:"Include hidden items" short:"a"`
}

func (c *Cmd) Run(ctx context.Context) error {
	rt, err := runtime.FromContext(ctx)
	if err != nil {
		return runtime.Exitf(runtime.ExitSoftware, "runtime context: %v", err)
	}
	items := []string{"alpha", "bravo", "charlie"}
	if c.All {
		items = append(items, "delta-hidden")
	}

	rt.Info("found %d items", len(items))
	rt.Checkpoint("discovery", "resolved item list")

	rt.Result(items)
	rt.Success("listed %d items", len(items))
	return nil
}

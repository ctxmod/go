package service

import (
	"context"
	"fmt"

	"example.com/cliproj/pkg/runtime"
)

// Deployment is what the service command reads from its parent group. The
// parent declares --target and --deploy-profile once; this command reads
// them through the bound parent instead of redeclaring them.
type Deployment interface {
	EffectiveTarget() string
}

// Cmd is the service subcommand under deploy.
type Cmd struct {
	Scale int `help:"Number of instances." default:"1"`
}

// Run reads the parent's flags through the bound root. Kong binds the
// root grammar struct, which the app package passes in as a Deployment
// lookup.
func (c *Cmd) Run(ctx context.Context, parent Deployment) error {
	rt, err := runtime.FromContext(ctx)
	if err != nil {
		return runtime.Exitf(runtime.ExitSoftware, "runtime context: %v", err)
	}
	target := parent.EffectiveTarget()
	rt.Info("scaling service to %d instances (target: %s)", c.Scale, target)
	rt.Result(map[string]string{
		"target": target,
		"scale":  fmt.Sprintf("%d", c.Scale),
	})
	return nil
}

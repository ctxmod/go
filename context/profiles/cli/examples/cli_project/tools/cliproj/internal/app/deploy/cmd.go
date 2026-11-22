package deploy

import (
	"context"

	"github.com/alecthomas/kong"

	"example.com/cliproj/pkg/runtime"
	"example.com/cliproj/tools/cliproj/internal/app/deploy/service"
)

// Cmd is the deploy command group. Its flags describe the deployment and
// are in scope for every subcommand beneath it (`deploy service --target
// prod` parses against these), so the subcommands never redeclare them: a
// duplicate long or short flag makes Kong panic at parser construction.
// DeployProfile derives --deploy-profile, keeping clear of the global
// --profile (the config profile).
type Cmd struct {
	Target        string      `help:"Deployment target." env:"DEPLOY_TARGET"`
	DeployProfile string      `help:"Deployment profile." env:"DEPLOY_PROFILE"`
	Service       service.Cmd `cmd:"" help:"Service operations."`
}

// Run is also called on the `deploy service` path: Kong runs every node of
// the selected path that has a Run method, leaf first. The guard keeps the
// group's own behavior to the bare `deploy` invocation.
func (c *Cmd) Run(ctx context.Context, kctx *kong.Context) error {
	if kctx.Selected() != nil && kctx.Selected().Name != "deploy" {
		return nil
	}
	rt, err := runtime.FromContext(ctx)
	if err != nil {
		return runtime.Exitf(runtime.ExitSoftware, "runtime context: %v", err)
	}
	rt.Info("deploying to %s (profile %q)", c.EffectiveTarget(), c.DeployProfile)
	rt.Result(map[string]string{"target": c.EffectiveTarget(), "profile": c.DeployProfile})
	return nil
}

// EffectiveTarget applies the default target.
func (c *Cmd) EffectiveTarget() string {
	if c.Target == "" {
		return "default"
	}
	return c.Target
}

// Package aisimple narrows the shared runtime to what the AI commands read:
// the base runtime plus the model settings from configuration. It shows the
// domain-runtime pattern — a command that needs a few more accessors gets a
// small wrapper around *runtime.Runtime instead of widening the base type.
package aisimple

import (
	"context"
	"fmt"

	"example.com/cliproj/pkg/runtime"
)

// AISimpleRuntime is the base runtime plus AI settings accessors.
type AISimpleRuntime struct {
	*runtime.Runtime
}

// AIProvider returns the configured provider name.
func (r AISimpleRuntime) AIProvider() string { return r.Global().AI.Provider }

// AIModel returns the configured model ID.
func (r AISimpleRuntime) AIModel() string { return r.Global().AI.Model }

// AIModelClass returns the configured model class (genius|pro|flash|nano).
func (r AISimpleRuntime) AIModelClass() string { return r.Global().AI.ModelClass }

// AIAPIKey returns the configured API key.
func (r AISimpleRuntime) AIAPIKey() string { return r.Global().AI.APIKey }

// AIMaxSteps returns the configured agent-loop step budget.
func (r AISimpleRuntime) AIMaxSteps() int { return r.Global().AI.MaxSteps }

// FromContext wraps the runtime stored in ctx.
func FromContext(ctx context.Context) (AISimpleRuntime, error) {
	rt, err := runtime.FromContext(ctx)
	if err != nil {
		return AISimpleRuntime{}, fmt.Errorf("aisimple: %w", err)
	}
	return AISimpleRuntime{Runtime: rt}, nil
}

// MustGetAISimpleRuntime is FromContext for commands that cannot run without
// a runtime; the panic is recovered and reported by runtime.RunCommand.
func MustGetAISimpleRuntime(ctx context.Context) AISimpleRuntime {
	art, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return art
}

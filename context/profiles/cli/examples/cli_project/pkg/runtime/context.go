package runtime

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
)

type runtimeKey struct{}

// NewContext returns a context that carries rt.
func NewContext(parent context.Context, rt *Runtime) context.Context {
	return context.WithValue(parent, runtimeKey{}, rt)
}

// FromContext returns the Runtime stored in ctx, or an error when none is.
func FromContext(ctx context.Context) (*Runtime, error) {
	rt, ok := ctx.Value(runtimeKey{}).(*Runtime)
	if !ok {
		return nil, errors.New("runtime: context does not contain a Runtime value")
	}
	return rt, nil
}

// NewSignalContext returns a context that is canceled on SIGINT or SIGTERM
// and carries rt. Callers must call cancel to release the signal handler.
func NewSignalContext(parent context.Context, rt *Runtime) (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
	return NewContext(ctx, rt), cancel
}

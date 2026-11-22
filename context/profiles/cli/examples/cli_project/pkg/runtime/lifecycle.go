package runtime

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
)

// Hook runs around a command. A Before hook that fails stops the command.
type Hook func(ctx context.Context, rt *Runtime, cmd string) error

// Hooks holds the functions RunCommand calls around the command.
type Hooks struct {
	Before []Hook
	After  []Hook
}

// RunCommand runs cmdFunc inside the lifecycle: Before hooks in order,
// Started, the command, After hooks in reverse order, then Completed on
// success or Failed on error. A panic is recovered, reported as Failed, and
// converted to an ExitSoftware error.
func RunCommand(ctx context.Context, rt *Runtime, cmd string, cmdFunc func(ctx context.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("panic: %v\n%s", r, debug.Stack())
			_ = rt.Failed("panic", fmt.Sprint(r), nil)
			err = Exit(ExitSoftware, errors.New(msg))
		}
	}()

	for _, h := range rt.hooks.Before {
		if err := h(ctx, rt, cmd); err != nil {
			_ = rt.Failed("before_hook", err.Error(), nil)
			return err
		}
	}
	_ = rt.Started(cmd, map[string]any{"session": rt.SessionID()})

	err = cmdFunc(ctx)

	for i := len(rt.hooks.After) - 1; i >= 0; i-- {
		if herr := rt.hooks.After[i](ctx, rt, cmd); herr != nil && err == nil {
			err = herr
		}
	}

	switch {
	case err == nil:
		_ = rt.Completed(nil)
	case IsSilentExit(err):
		_ = rt.Completed(map[string]any{"silent_exit": err.Error()})
	default:
		_ = rt.Failed("command_failed", err.Error(), nil)
	}
	return err
}

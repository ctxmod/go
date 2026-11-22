// Package runtime is the per-invocation dependency every command receives
// through its context: the renderer, the logger, the resolved globals, the
// working directories, and the lifecycle hooks.
package runtime

import (
	"log/slog"
	"os"

	"example.com/cliproj/pkg/config"
	"example.com/cliproj/pkg/output"
)

// BuildVersion and BuildCommit are injected via -ldflags; the defaults keep
// a plain `go run` working.
var (
	BuildVersion = "0.1.0"
	BuildCommit  = "dev"
)

// Runtime is a concrete struct, not an interface: commands get the full
// method set, and new methods never break them. It embeds *output.Output, so
// commands call rt.Info, rt.Result, and rt.Error directly.
type Runtime struct {
	*output.Output

	logger *slog.Logger
	global config.Global
	paths  Paths
	hooks  Hooks
}

// NewRuntime constructs the runtime from its already-built components.
func NewRuntime(logger *slog.Logger, out *output.Output, global config.Global, paths Paths) *Runtime {
	return &Runtime{Output: out, logger: logger, global: global, paths: paths}
}

// Close flushes the output and removes the session's Ephemeral directory.
// Teardown is best-effort, so it reports failures as warnings, not errors.
func (r *Runtime) Close() {
	if r.Output != nil {
		_ = r.Output.Close()
	}
	if r.paths.Ephemeral != "" {
		if err := os.RemoveAll(r.paths.Ephemeral); err != nil && r.logger != nil {
			r.logger.Warn("ephemeral cleanup failed", "path", r.paths.Ephemeral, "error", err)
		}
	}
}

// Paths returns the invocation's working directories.
func (r *Runtime) Paths() Paths { return r.paths }

// Global returns the resolved shared settings.
func (r *Runtime) Global() config.Global { return r.global }

// SessionID returns the identifier that scopes this invocation's directories.
func (r *Runtime) SessionID() string { return r.global.SessionID }

// AppName returns the application name.
func (r *Runtime) AppName() string { return r.global.AppName }

// Debug reports whether debug mode is on.
func (r *Runtime) Debug() bool { return r.global.Debug }

// Logger returns the run's slog logger.
func (r *Runtime) Logger() *slog.Logger { return r.logger }

// Hooks returns the lifecycle hooks RunCommand runs around the command;
// the bootstrap appends to them before dispatch.
func (r *Runtime) Hooks() *Hooks { return &r.hooks }

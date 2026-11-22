# Runtime

A context-bound runtime is the sole dependency injected into every command: one concrete `Runtime` struct that embeds the output renderer and adds the resolved globals, path resolution, structured logging, and lifecycle hooks. Commands receive it from `context.Context` rather than as a direct parameter, report failures through typed exit codes, and run inside a lifecycle wrapper that recovers panics and emits completion events. The example project (`examples/cli_project`, packages `pkg/runtime` and `pkg/runtime/aisimple`) implements everything below.

## Runtime Type

```go
type Runtime struct {
    *output.Output

    logger *slog.Logger
    global config.Global
    paths  Paths
    hooks  Hooks
}

func NewRuntime(logger *slog.Logger, out *output.Output, global config.Global, paths Paths) *Runtime

func (r *Runtime) Close()               // flush output, remove the Ephemeral directory
func (r *Runtime) Paths() Paths
func (r *Runtime) Global() config.Global
func (r *Runtime) SessionID() string
func (r *Runtime) AppName() string
func (r *Runtime) Debug() bool
func (r *Runtime) Logger() *slog.Logger
func (r *Runtime) Hooks() *Hooks
```

The runtime is a concrete struct, not an interface, for the reason the core constructor rule gives: commands get the full method set, and adding a method never breaks a consumer. A consumer that needs a narrow seam declares its own small interface. Embedding `*output.Output` means commands call `rt.Info()`, `rt.Result()`, `rt.Error()` directly on the runtime; no separate output parameter is needed.

Close has a no-error shape on purpose: teardown is best-effort, so a failing output sink or a failed cleanup is logged as a warning, never turned into the command's error.

### Domain runtimes

A command family that needs a few more accessors wraps the base runtime instead of widening it. The example's `aisimple.AISimpleRuntime` embeds `*runtime.Runtime` and adds the AI settings (`AIProvider()`, `AIModel()`, `AIModelClass()`, `AIAPIKey()`, `AIMaxSteps()`); `aisimple.MustGetAISimpleRuntime(ctx)` retrieves it, panicking when the context holds no runtime — a programming error the lifecycle wrapper recovers and reports. The same narrowing works the other way: a consumer that only needs repository identity and a worker count declares a three-method interface rather than depending on the full runtime.

## Context Injection

Commands receive the runtime via `context.Context`:

```go
func (c *Cmd) Run(ctx context.Context) error {
    rt, err := runtime.FromContext(ctx)
    if err != nil {
        return runtime.Exitf(runtime.ExitSoftware, "runtime context: %v", err)
    }
    // use rt.Info(), rt.Result(), etc.
}
```

The bootstrap wires the runtime into context via `runtime.NewContext(ctx, rt)`, or `runtime.NewSignalContext(parent, rt)`, which also cancels on SIGINT or SIGTERM. Kong's bind system injects the context into `Run()` methods.

## Paths

Four path categories with distinct lifetime semantics:

| Path | Lifetime | Purpose |
|------|----------|---------|
| `Ephemeral` | Removed on exit | Temp files, alt-screen captures |
| `Retained` | Persists after exit | Logs, side files from JSON batch mode, debug captures |
| `UserArtifacts` | Persists after exit | User-visible outputs the caller consumes |
| `CrossSession` | Shared across invocations | Stable cache |

All paths root under the application name and session ID. `CrossSession` excludes the session ID because it is shared. Explicit flags override the platform defaults (XDG on Linux):

- `--temp-dir` overrides the ephemeral root
- `--cache-dir` overrides the retained and cross-session root
- `--output-dir` overrides the user artifacts root

`ResolvePaths` computes all four but creates only `CrossSession`, a single stable per-application root. The session-scoped directories are unique per invocation; creating them eagerly leaks one empty directory per run — a frequently polled tool accumulates tens of thousands of them within days. Their consumers create them lazily on first use instead: `Paths.EnsureEphemeral()` creates the ephemeral directory idempotently (call it before handing the path to anything that writes temp files without creating the directory, such as SQLite's temp directory), and the output capture creates the retained directory when it opens its file. A clean invocation leaves nothing behind.

When directory creation fails because a regular file sits in the path (`ENOTDIR`), name the offending file in the error message (and keep `errors.Is(err, syscall.ENOTDIR)` matching) so the operator can remove it; a bare errno is not actionable.

## Session ID

Each invocation gets a session ID. Resolution order: the `--session` flag (or its config key), then an auto-generated 8-character hex string from `crypto/rand` — uniform and not sequentially guessable. The session ID scopes the ephemeral, retained, and user-artifacts directories.

## Exit Codes

A typed `ExitCode` following sysexits(3) conventions where one fits:

| Constant | Code | Meaning |
|----------|------|---------|
| `ExitOK` | 0 | Success |
| `ExitNotFound` | 1 | The requested thing does not exist |
| `ExitUsage` | 2 | Command-line usage error |
| `ExitUnavailable` | 69 | Service unavailable (interrupted) |
| `ExitSoftware` | 70 | Internal software error |
| `ExitPartial` | 71 | Mixed results: some operations succeeded, at least one failed |
| `ExitTempFail` | 75 | Transient upstream failure (5xx, 429, timeout, DNS/TLS): a rerun may succeed |
| `ExitConfig` | 78 | Configuration error |

Distinct codes let wrappers react differently: `ExitPartial` separates "some things failed" from "everything failed" (`ExitSoftware`), and `ExitTempFail` (sysexits `EX_TEMPFAIL`) separates "bad request" from "upstream having a moment". Add domain codes the same way when a caller's remediation differs — for example a code for "blocked by a challenge page: solve it, do not retry".

## ExitError

`ExitError` carries a semantic exit code alongside the underlying error. `Exit(code, err)` wraps an error; `Exitf(code, format, args...)` formats a message and preserves the last error argument as the cause for `errors.Is` traversal.

```go
return runtime.Exitf(runtime.ExitConfig, "config: %w", err)
```

`ExitSilent(code, message)` sets the exit code but suppresses error rendering, for expected non-error outcomes — denial, warnings, "no results" — whose payload the command already emitted via `Result`; terminal handlers check `IsSilentExit(err)` and skip the error envelope.

## MultiError

`MultiError` aggregates errors from batch operations. `ExitCode()` returns the highest exit code from contained errors; an error without a code counts as `ExitSoftware`. `Append`, `ErrorOrNil`, and `NewMultiError` (which filters nil arguments) keep collection code short.

## Mapping errors to the process exit code

`ExitCodeFromError(ctx, err)` returns, in order: `ExitOK` for a nil error; the highest code from a `*MultiError`; the embedded code from an `*ExitError`; `ExitOK` when the context is canceled and the error is that cancellation (the user pressed Ctrl-C and the command stopped cleanly); `ExitUnavailable` when the context is canceled but the error is unrelated; `ExitSoftware` otherwise. Unwrap before checking cancellation: a coded failure whose chain happens to include the context's cancellation must keep its code, not exit 0 while its own log line says the command failed.

## Lifecycle Hooks

`Hooks` holds `Before` and `After` functions called around command execution:

```go
type Hooks struct {
    Before []Hook
    After  []Hook
}

type Hook func(ctx context.Context, rt *Runtime, cmd string) error
```

`RunCommand(ctx, rt, cmd, fn)` executes the before hooks (a failing one stops the command), emits `Started`, runs the command function, executes the after hooks in reverse order, then emits `Completed` or `Failed` automatically. Panics are recovered, reported through `Failed`, and converted to an `ExitSoftware` error. Normal errors are returned to the caller to render.

## Bootstrap Sequence

A clean bootstrap in `app.Run()`:

1. Build the parser with `kong.New` (`kong.Writers(os.Stdout, os.Stderr)`); a construction error is a grammar defect in the binary, not a usage error
2. Parse the arguments
3. Load config via `config.Load()` with the extracted user flags, and decode the globals
4. Resolve the session ID and paths
5. Construct the output renderer via `output.New()` with the configured mode
6. Construct the slog logger via `output.NewSlogHandler()`
7. Construct the runtime via `runtime.NewRuntime()`
8. Wire context with `runtime.NewSignalContext()`
9. Bind the runtime and context into Kong
10. Execute the command via `runtime.RunCommand()` and map the result with `ExitCodeFromError`

```go
rt := runtime.NewRuntime(logger, renderer, global, paths)
defer rt.Close()
ctx, cancel := runtime.NewSignalContext(context.Background(), rt)
defer cancel()
kctx.Bind(rt)
kctx.BindTo(ctx, (*context.Context)(nil))
err = runtime.RunCommand(ctx, rt, kctx.Command(), func(ctx context.Context) error {
    return kctx.Run()
})
return int(runtime.ExitCodeFromError(ctx, err))
```

## Related

[Kong Command Parsing](../../grammar/reference/kong_parsing.md)
[Output Interface](output_modes.md)
[Constructor Return Types](../../../../core/packages/api_design/rules/constructor_return_types.md)

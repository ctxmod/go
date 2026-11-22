# CLI Project Scaffolding

Scaffold a Go CLI project using Kong for command parsing and shared `config`, `output`, and `runtime` packages for configuration, rendering, and dependency injection, wired through context-based dependency injection and formatting interfaces. The procedure produces a working project with clean separation between parsing, configuration, runtime plumbing, output rendering, and application logic. The complete, buildable result is the profile's example project, `examples/cli_project` (module `example.com/cliproj`); copy from it rather than from the fragments below.

## Directory Layout

```
go.mod                      # one module for the repository
tools/{toolname}/
    cmd/
        main.go              # Entry point: os.Exit(app.Run(os.Args))
    internal/
        app/
            root.go          # Root CLI struct, Run() bootstrap, loadConfig, setupOutput
            {command}/
                cmd.go       # Command struct and Run(ctx) method
        config/             # (optional) tool-specific config extensions
        db/                 # (optional) tool-local persistence, e.g. a keyed-record store
pkg/                        # Packages shared by every tool in the repository
    config/                 # Config loading with koanf, Kong flag extraction
    output/                 # Mode-aware renderer and formatting interfaces
    runtime/                # Runtime struct with paths, exit codes, and lifecycle
```

A single-tool repository can drop the `tools/{toolname}/` level and keep `cmd/` and `internal/` at the root; the shared packages then live under `internal/` too. Where several repositories share the `config`, `output`, and `runtime` packages, import them from one library module instead of copying them.

The `cmd/main.go` file is minimal. The `internal/app/root.go` contains the CLI struct, the bootstrap sequence, and helper functions. Each command lives in its own package under `internal/app/{command}/` with a `cmd.go` file containing the command struct and its `Run(ctx context.Context) error` method.

## Step 1: Define the Root Grammar

Create the root CLI struct in `internal/app/root.go` with global flags for configuration, debugging, logging, and output:

```go
type Root struct {
    ConfigFiles []string `name:"config" help:"Config files to load in order." type:"path" sep:","`
    EnvFile     []string `help:"Dotenv files to load in order." type:"path" sep:","`
    Profile     string   `help:"Named config profile to activate." env:"APP_PROFILE"`

    Debug    bool   `help:"Enable debug run artifacts." koanf:"debug.enabled"`
    DebugDir string `help:"Directory for debug runs." type:"path" koanf:"debug.dir"`

    LogLevel string `help:"Log level." enum:"debug,info,warn,error," default:"" koanf:"log.level"`
    Output   string `help:"Output format." enum:"text,json,ndjson,yamlish,quiet,terminal," default:"" koanf:"output.mode"`
    Verbose  int    `help:"Increase stderr verbosity." short:"v" type:"counter" koanf:"verbose"`

    List    list.Cmd    `cmd:"" help:"List available items"`
    Version version.Cmd `cmd:"" help:"Print version information"`
}
```

Config-loading flags (`--config`, `--env-file`, `--profile`) feed into the config loader. Every other global flag names its config key in a `koanf` tag and overrides that key only when the user sets it (`--output`, `--log-level`, `--verbose`); none carries a real Kong default, because a default would fill the field unconditionally and the config layers could never decide. Choose field names whose derived flag is the intended spelling (`EnvFile` derives `--env-file`) instead of dashed `name` tags.

[Kong Command Parsing](../../grammar/reference/kong_parsing.md)

## Step 2: Implement the Bootstrap in app.Run()

The bootstrap runs in one direction with no backtracking:

```go
func Run(args []string) int {
    var cli Root
    parser, err := kong.New(&cli,
        kong.Name(appName),
        kong.Writers(os.Stdout, os.Stderr),
    )
    if err != nil { /* grammar defect: report, return ExitSoftware */ }

    kctx, err := parser.Parse(args[1:])
    if err != nil { /* parser.Errorf, return ExitUsage */ }

    global, paths, code := loadConfig(&cli, kctx)
    if code != 0 { return code }

    renderer, logger, code := setupOutput(global, paths)
    if code != 0 { return code }

    rt := runtime.NewRuntime(logger, renderer, global, paths)
    defer rt.Close()

    ctx, cancel := runtime.NewSignalContext(context.Background(), rt)
    defer cancel()
    kctx.Bind(rt)
    kctx.BindTo(ctx, (*context.Context)(nil))

    err = runtime.RunCommand(ctx, rt, kctx.Command(), func(ctx context.Context) error {
        return kctx.Run()
    })
    /* render err unless silent; return the ExitCodeFromError mapping */
}
```

`loadConfig` extracts user flags via `config.ExtractUserFlags(kctx)`, loads with `config.Load()`, decodes the globals, and resolves the session ID and paths. `setupOutput` constructs the renderer and logger. The runtime binds into Kong context.

## Step 3: Load Configuration

Use the shared `config.Load()`, which applies all sources in precedence order:

```
defaults < configs/default.yaml < configs/{profile}.yaml < user config
    < project-local config < --config files < .env files < --env-file files
    < process env < CLI flag overrides
```

Flag overrides are extracted automatically by `config.ExtractUserFlags(kctx)`, which walks Kong's parsed context to collect only explicitly-set flags (on the command line or through their env bindings).

[Configuration Layering with koanf](../../config/reference/config_layering.md)

## Step 4: Wire Output and Logging

Construct the output renderer with the resolved mode and retained directory:

```go
mode, err := output.ParseMode(global.OutputMode)
renderer, err := output.New(
    os.Stdout, os.Stderr,
    output.WithMode(mode),
    output.WithColor(output.ColorAuto(os.Stderr)),
    output.WithVerbosity(global.Verbose),
    output.WithRetainedDir(paths.Retained), // debug runs only
)
logger := slog.New(output.NewSlogHandler(renderer, global.LogLevel))
```

[Output Interface](../../output/reference/output_modes.md)

## Step 5: Implement Commands

Each command is a struct with a `Run(ctx context.Context) error` method in its own package:

```go
package list

type Cmd struct {
    All bool `help:"Include hidden items" short:"a"`
}

func (c *Cmd) Run(ctx context.Context) error {
    rt, err := runtime.FromContext(ctx)
    if err != nil {
        return runtime.Exitf(runtime.ExitSoftware, "runtime: %v", err)
    }
    items := c.fetch()
    rt.Info("found %d items", len(items))
    rt.Checkpoint("discovery", "resolved item list")
    rt.Result(items)
    rt.Success("listed %d items", len(items))
    return nil
}
```

Commands use `runtime.FromContext(ctx)` to obtain the runtime, then call rendering methods directly. Results implement formatting interfaces (`TextFormattable`, `DataFormattable`) for mode-aware rendering. Key-value diagnostics go through the logger (`rt.Logger().Info("agent invocation", "model", id)`), whose handler routes them back through the renderer.

## Step 6: Wire Runtime and Lifecycle

The shared `runtime` package handles lifecycle hooks, panic recovery, and automatic `Started`/`Completed`/`Failed` events via `runtime.RunCommand()`:

```go
err = runtime.RunCommand(ctx, rt, kctx.Command(), func(ctx context.Context) error {
    return kctx.Run()
})
```

This wraps the command in before/after hooks, emits lifecycle events, and recovers panics. Register hooks on `rt.Hooks()` before dispatch — the example adds a debug-only before hook that logs the dispatch.

[Runtime](../../output/reference/runtime_injection.md)

## Step 7: Add Interactive Commands (Optional)

For commands requiring terminal UI, create a Bubble Tea model and launch from `Run()`:

```go
func (c *WatchCmd) Run(ctx context.Context) error {
    rt, _ := runtime.FromContext(ctx)
    if !rt.Interactive() {
        return fmt.Errorf("watch requires interactive terminal")
    }
    m := newWatchModel(c.Interval)
    p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
    _, err := p.Run()
    return err
}
```

[Bubble Tea and Bubbles](../../tui/reference/bubbletea_bubbles.md)

## Verification

After scaffolding, confirm:

- `go build ./...` and `go vet ./...` succeed
- `{toolname} --help` prints help with all commands and global flags — building the parser is where duplicate-flag panics surface, so also run the bare parent of every command group and each subcommand
- `{toolname} list --output json` emits JSON on stdout with no stderr noise
- `{toolname} list --output text` emits human-readable output
- `{toolname} list 2>/dev/null` produces clean stdout suitable for piping
- `{toolname} list --output yamlish` emits dense structured text

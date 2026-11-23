# Bubble Tea and Bubbles

Bubble Tea (`charm.land/bubbletea/v2`) implements the Elm Architecture for terminal applications. Bubbles (`charm.land/bubbles/v2`) provides reusable components that follow the same architecture. Together they handle interactive terminal rendering -- progress bars, input forms, scrollable lists -- while the output interface handles non-interactive data emission.

## Elm Architecture

The architecture enforces unidirectional data flow. User events become messages, `Update` produces a new model and optional commands, `View` renders from model state. Three methods plus initialization define a complete program:

```go
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() View
}
```

- **Model**: Go struct holding all application state. State changes happen exclusively in `Update`.
- **Update**: Pure reducer receiving a message and current model, returning new model and optional command. No side effects.
- **View**: Pure function returning a `tea.View` built from model state — `tea.NewView(s)` for the content, plus declarative terminal state as fields (see below). No side effects.
- **Cmd**: Function performing side effects (I/O, timers, HTTP) that returns a `Msg` when complete. Return `nil` for no-op.

The single-threaded event loop eliminates race conditions without mutexes. Both `Update` and `View` are pure functions of their inputs, making them directly testable by passing inputs and checking outputs.

## Message Types

Messages can be any Go type (`tea.Msg` is `any`). Key builtin messages:

| Message | Trigger |
|---------|---------|
| `KeyPressMsg` / `KeyReleaseMsg` | Keyboard input; `KeyMsg` is the interface both satisfy. Match with `msg.String()` (`"ctrl+c"`, `"enter"`, `"space"`) or the `Code`/`Text`/`Mod` fields |
| `MouseClickMsg`, `MouseReleaseMsg`, `MouseWheelMsg`, `MouseMotionMsg` | Mouse events, when the view sets a `MouseMode`; `MouseMsg` is their interface and `.Mouse()` yields `X`, `Y`, `Button`, `Mod` |
| `PasteMsg` (with `PasteStartMsg` / `PasteEndMsg`) | Bracketed paste |
| `WindowSizeMsg` | Terminal resize, sent on start and SIGWINCH |
| `QuitMsg` | Program exit signal |
| `FocusMsg` / `BlurMsg` | Terminal focus changes, when the view sets `ReportFocus` |

Custom messages carry domain-specific data:

```go
type fetchCompleteMsg struct {
    items []Item
    err   error
}
```

## Command Combinators

Commands manage side effects and their sequencing:

| Combinator | Behavior |
|-----------|----------|
| `Batch(cmds...)` | Run concurrently, no ordering guarantees |
| `Sequence(cmds...)` | Run sequentially, one at a time |
| `Tick(duration, fn)` | Interval timer independent of system clock |
| `Every(duration, fn)` | Interval timer synced to system clock |
| `Exec(cmd, fn)` | Blocking subprocess execution |
| `RequestWindowSize` | Query current terminal dimensions (a `Msg` in v2, not a `Cmd` factory) |

Use `Batch` to start multiple independent operations from `Init`. Use `Sequence` when operations have ordering dependencies.

## Program Lifecycle

`tea.NewProgram(model, opts...)` creates a program. `.Run()` blocks while managing terminal setup/teardown, event polling, and frame-rate-limited rendering, and returns the final model; run it in a goroutine when other code injects events with `.Send(msg)` (v1's `Start()` is gone).

Key program options:

| Option | Effect |
|--------|--------|
| `WithOutput(w)` | Redirect rendering to writer (use `os.Stderr`) |
| `WithFPS(n)` | Frame rate limit |
| `WithoutRenderer()` | Batch mode, no terminal rendering |
| `WithContext(ctx)` | Cancellation support |
| `WithFilter(fn)` | Event filtering |
| `WithWindowSize(w, h)`, `WithColorProfile(p)` | Fixed initial size and color profile — deterministic rendering in tests |

### Declarative terminal state (v2)

Bubble Tea v2 moved terminal modes from program options and commands onto the `tea.View` the model returns each frame, so the mode follows model state instead of being toggled imperatively:

```go
func (m model) View() tea.View {
    v := tea.NewView(m.render())
    v.AltScreen = m.fullscreen            // was WithAltScreen / EnterAltScreen
    v.MouseMode = tea.MouseModeCellMotion // was WithMouseCellMotion
    v.ReportFocus = true                  // was WithReportFocus
    v.WindowTitle = "cliproj"             // was SetWindowTitle
    return v
}
```

The cursor (`Cursor`, `tea.NewCursor(x, y)`), bracketed-paste opt-out, and keyboard enhancements are `View` fields too. Code ported from v1 that still passes `WithAltScreen()` or `WithMouseCellMotion()` no longer compiles against `charm.land/bubbletea/v2`; the upgrade guide maps every removed option and command.

For CLI applications that mix interactive and non-interactive commands, pass `tea.WithOutput(os.Stderr)` so interactive views render on stderr while stdout remains available for data output.

## Bubbles Component Catalog

Each component follows the Elm Architecture with `New()`, `Update(msg)`, and `View()` methods; a component's `View()` returns a plain `string` that the parent composes into its own `tea.View`. Embed component models in parent structs, delegate messages from the parent `Update`, and compose views in `View`.

| Component | Package | Purpose |
|-----------|---------|---------|
| Spinner | `spinner` | Animated loading indicators |
| Text Input | `textinput` | Single-line input with Unicode, clipboard, echo modes |
| Text Area | `textarea` | Multi-line input |
| List | `list` | Scrollable, filterable, selectable item lists |
| Progress | `progress` | Horizontal progress bars |
| Table | `table` | Tabular data display |
| Viewport | `viewport` | Scrollable content regions |
| File Picker | `filepicker` | File system browser |
| Help | `help` | Key binding help menus |
| Timer | `timer` | Countdown timer |
| Stopwatch | `stopwatch` | Elapsed time tracking |

## Component Integration Pattern

Embed Bubbles components as fields and delegate messages. Multiple components compose via `tea.Batch`:

```go
type model struct {
    spinner  spinner.Model
    progress progress.Model
    items    []string
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    var cmd tea.Cmd

    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)

    m.progress, cmd = m.progress.Update(msg)
    cmds = append(cmds, cmd)

    return m, tea.Batch(cmds...)
}
```

## Bubble Tea Integration (rule context)

Extracted verbatim from the `cli-framework-usage` rule so that rule can stay a pointer.

### Bubble Tea Integration

Use Bubble Tea when a command requires interactive input, progress visualization, or rich terminal rendering. Simple non-interactive commands should use the output interface directly.

| Scenario | Approach |
|----------|----------|
| List results, single JSON blob | Output interface via `Result()` |
| Long-running operation with progress | Output interface via `Progress()` |
| Multi-step interactive wizard | Bubble Tea model with Bubbles components on stderr |
| Streaming output | Output interface via NDJSON mode on stdout |

Pass `tea.WithOutput(os.Stderr)` to `tea.NewProgram` so interactive views render on stderr. This keeps stdout clean for data payloads.

## Related

[Kong Command Parsing](../../grammar/reference/kong_parsing.md)
[Output Interface](../../output/reference/output_modes.md)

## Sources

- https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md — v1 to v2 changes: `charm.land` import paths, `View() tea.View`, removed program options and commands, key/mouse/paste message split, `RequestWindowSize`, `Start()` removal (accessed 2026-11-22)
- https://pkg.go.dev/charm.land/bubbletea/v2 — `Model`, `View`, `KeyPressMsg`, mouse messages, program options (v2.1.2, published 2026-11-18; accessed 2026-11-22)
- https://pkg.go.dev/charm.land/bubbles/v2/spinner — component `View() string` and `Update` shape (v2.2.2, published 2026-11-20; accessed 2026-11-22)

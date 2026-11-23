# Output Interface

The output interface pattern separates command logic from rendering by defining an `Output` abstraction injected into commands via the runtime. It supports text, JSON, NDJSON, yamlish, quiet, and terminal modes, with formatting interfaces, slog integration, and stream discipline. Commands call methods without knowing whether output renders as human-readable text, a JSON object, a yamlish view, or an NDJSON event stream. Mode selection happens once at construction; all downstream code operates against the abstraction.

The example project's `pkg/output` (`examples/cli_project`) is a compact, working implementation of the core of this pattern: the six modes, the stream split, `TextFormattable`/`DataFormattable`, the JSON envelope written at `Close()`, the slog handler, and the capture file. Sections below that go beyond it (run states, interim record streaming, tiered and elided records) describe what a fuller shared renderer adds once several tools depend on it.

## Interface Design

The contract provides methods for messaging, structured output, progress, lifecycle events, and terminal detection. Implement it as a concrete renderer type whose method set is the contract (the example's `*output.Output`), and let a consumer that wants a narrower seam declare its own interface; the sketch below lists the method set:

```go
type Output interface {
    // Messaging
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    Fatal(err error)
    Status(msg string)

    // Structured output
    Success(msg string, args ...any)
    Result(data any)

    // Progress
    Progress(msg string, args ...any)
    Checkpoint(id, msg string)

    // Events (true no-ops only in quiet/yamlish modes, which embed
    // NoOpOutput; json/text/terminal emit them as stderr diagnostics;
    // in ndjson mode they are stderr kind-lines and Failed/Error feed
    // the terminal record's status instead of stdout records)
    Started(operation string, metadata map[string]any) error
    Completed(result map[string]any) error
    Failed(code, message string, details map[string]any) error

    // Lifecycle
    Interactive() bool
    Close()
}
```

## Output Modes

Six modes cover the range of CLI output needs:

| Mode | Value | stdout Content | stderr Content | Use Case |
|------|-------|---------------|----------------|----------|
| Text | `text` | Human-readable result | Info, warnings, errors, progress | Default interactive terminal use |
| JSON | `json` | One JSON document (the envelope written at `Close()`) | Diagnostics as kind-discriminated JSON lines (info/warn/error/started/completed/…) | Script consumption, API integration |
| NDJSON | `ndjson` | `kind:result` records as they arrive, then one terminal `kind:completed` record carrying `status` (ok/failed) | Kind-discriminated diagnostic lines; `kind:error` never reaches stdout | Streaming pipelines, real-time processing |
| Yamlish | `yamlish` | Dense YAML-like text | Info, warnings, errors | Human-readable structured data |
| Quiet | `quiet` | Nothing (stdout stays zero bytes) | Fatal diagnostics only (the exit code is the outcome surface) | Exit code only |
| Terminal | `terminal` | Interactive alt-screen UI; at teardown a dump of the captured view | Live footer and interactive UI rendering | Rich terminal rendering |

A fuller renderer also carries a **run state** — `complete`, `vacuous`, `partial`, or `withheld`, plus an optional reason and an exit-cause token — and renders it in every mode, not only in the JSON envelope: text appends a final `run_state=<state>` marker line to stdout (always emitted, `complete` included, with ` reason="<quoted>"` when one was declared, and never filtered by verbosity tiers); json carries `run_state` (always) plus `run_state_reason` and `cause` when declared; ndjson emits an interim `kind:run_state` stdout record at declaration time and repeats the state on the terminal `kind:completed` record (a declared state is never silently dropped after the terminal record); yamlish ends with top-level `run_state:`/`run_state_reason:` keys; quiet writes one `kind:run_state` record to stderr at `Close()`; terminal adds a `state <value>` part to its live footer and the text marker after the teardown dump. The first non-`complete` declaration wins.

Terminal mode is TTY-gated once selected: when stderr is not a TTY (or the
terminal delegate cannot start) mode selection falls back to the text
delegate with a diagnostic.

### Interim streaming

In a fuller renderer, `RecordStreamer()` returns a non-nil streamer in every payload mode —
text, json, yamlish, terminal, ndjson — and nil only in quiet. Streaming
is a flush-discipline difference, not a presence difference: ndjson
renders each streamed record as a stdout `kind:result` line; text,
yamlish, and terminal render the record's derived text on stdout
immediately, line-complete per record; json routes each streamed record
to stderr as `{"kind":"record","value":…,"version":1}` (plus a one-line
`kind:info` "record stream open" marker at streamer construction) so
stdout stays exactly one document. Streamed records never enter json's
buffered aggregate — a tool whose complete view belongs in the stdout
document emits it through a final `Result`. Producers that never stream
keep the single-`Result` fallback; it stays valid in every mode.

### Default mode selection

The mode is flag-driven, not TTY-detected: the bootstrap reads the
output-format setting (the `--output` flag or its config key, default
`text`) and parses it into an `output.Mode` once. Do not select the mode
by detecting whether stdout is a terminal: a TTY-detection helper makes
piped and interactive invocations of the same command render
differently, which is exactly what scripts cannot tolerate. With
flag-driven selection, piped and interactive invocations of the same
command select the same mode deterministically. The `terminal` mode
remains flag-opt-in (TTY-gated with a text fallback once selected).

## Formatting Interfaces

Types implement formatting interfaces to control how they render per mode. The output system resolves data through two per-mode fallback chains:

```
TEXT:     TextFormattable → RecordStreamProvider (DeriveText) → built-in types
DATA:     DataFormattable → raw marshal (JSON / NDJSON / YAMLISH)
```

Derivation precedence: text resolution keeps **TextFormattable
first** — a payload implementing both `TextFormattable` and
`RecordStreamProvider` renders its own `FormatText` in text/terminal, and
only a payload without `FormatText` gets the engine-derived view from its
records. Structured modes run a parallel stream branch with the same
gating (`recordStreamOf` skips `DataFormattable` payloads): json places
the tier-projected records as the envelope item, ndjson emits them as
per-record lines. One documented divergence: yamlish consults its stream
branch **before** its `FormatYamlish` path, so a
`TextFormattable`+`RecordStreamProvider` payload renders the derived view
in yamlish where text/terminal render `FormatText` — a deliberate
ordering (yamlish is `FormatYamlish`-first, never TextFormattable-first),
not a precedence regression.

`TextFormattable.FormatText(args FmtArgs)` owns its color behavior: when
`args.UseColor` is false the implementation must return plain text, and
text resolution strips ANSI as a safety net for misbehaving
implementations. `DataFormattable.FormatData(args FmtArgs)` may branch on
`args.Format` to reshape the payload per structured format.

The example implements the first two; a fuller renderer recognizes the whole set:

| Interface | Method | Purpose |
|-----------|--------|---------|
| `TextFormattable` | `FormatText(args FmtArgs) string` | Text rendering; color controlled by `args.UseColor` |
| `DataFormattable` | `FormatData(args FmtArgs) any` | Canonical payload for all structured modes (JSON, NDJSON, yamlish) |
| `Countable` | `ResultCount() int` | Logical item count for the envelope's `result_records` |
| `TotalProvider` | `TotalMatching() int` | Pre-pagination total for the envelope's `total_matching` |
| `FailureStateful` | `FailureState() (bool, string)` | Payload self-reports failure for the envelope's `status` |
| `RunStateProvider` | `RunState() RunState` | Payload self-reports run state for the resolved run-state rendering (all six modes, not only the json envelope) |
| `NDJSONStreamable` | `StreamItems() []any`, `StreamMetadata() map[string]any` | NDJSON per-item streaming plus terminal-record metadata |
| `Tiered` | `RecordTier() Tier` | Importance tier for tier-filtered projections (ceiling from `FmtArgs.TierCeiling`) |
| `Identifiable` | `RecordIdentity() Identity` | Elision-reachable identity for collapse and summaries |
| `RecordStreamProvider` | `RecordStream() []any` | Record stream every view derives from when no own-format method exists |

`fmt`-arg threading: implementations receive the resolved `FmtArgs`
(verbosity, color, format flags) rather than consulting global state.

## Constructor Pattern

Mode selection happens at the application entry point via functional options:

```go
renderer, err := output.New(
    os.Stdout, os.Stderr,
    output.WithMode(mode),
    output.WithColor(output.ColorAuto(os.Stderr)),
    output.WithVerbosity(global.Verbose),
    output.WithRetainedDir(paths.Retained),
)
```

A verbosity option controls whether `Info`, `Warn`, `Success`, `Progress`, `Checkpoint`, and `Status` delegate to the mode-specific output or are captured only; `Error` and `Fatal` always delegate regardless of verbosity. Keep one explicit verbosity control; when an older boolean verbose option survives as a deprecated alias, the explicit setting outranks it.

## Stream Separation Convention

| Stream | Purpose |
|--------|---------|
| stdout | Final result payload (human text, JSON, yamlish) |
| stderr | Diagnostics, warnings, errors, progress, interactive UI |

This convention enables `command | jq '.items'` pipelines. Bubble Tea programs must render to stderr via `tea.WithOutput(os.Stderr)`.

## slog Integration

`NewSlogHandler(out, level)` returns an `slog.Handler` that routes records through the Output interface. Info/Warn/Error map to corresponding Output methods. Debug records bypass the mode delegate and write directly to stderr: they only exist when debug logging was explicitly requested, and a quiet delegate that suppresses all log methods would otherwise drop them silently (the example's `Output.Debug` does this). Attributes format inline (`key=value`) for consistency with Result output.

## Capture File

When `WithRetainedDir` is set, all output method calls are captured to a structured log file in the retained directory. This provides post-mortem diagnostics without polluting stdout or stderr. The capture file uses yamlish-formatted entries with timestamps.

## Testing JSON Output

JSON mode aggregates `Result` payloads into an internal envelope and writes nothing to the underlying writer until `Close()`. A test asserting on captured stdout before the output is closed reads an empty buffer — indistinguishable from a command that produced nothing. Wrap the pattern in a close-then-assert helper:

```go
func captureJSON(fn func(out *output.Output) error) (string, error) {
    var stdout, stderr bytes.Buffer
    out, err := output.New(&stdout, &stderr, output.WithMode(output.ModeJSON))
    if err != nil {
        return "", err
    }
    runErr := fn(out)
    if err := out.Close(); err != nil { // writes the envelope
        return "", err
    }
    return stdout.String(), runErr
}
// stdout now carries the JSON envelope; assert on it directly
```

Text mode does not buffer this way, which is why the trap only bites JSON assertions.

Envelope shape (single JSON object on stdout; a fuller renderer can also
emit it early via `ResultNow`, identically): `version` (1), `value`
(always an array), `result_records`, `status` (ok|failed), and — in a
renderer with run states — `run_state`
(complete|vacuous|partial|withheld — always present, `complete` when
undeclared), `metadata.internal_events`, plus `run_state_reason` when a
run-state declaration carried one, `total_matching` when the payload
provides it, and `message`/`code` on failure. The `cause` key is emitted
by the engine when a cause was declared — via the optional
`CauseDeclarer` capability (`output.AsCauseDeclarer`) — and rides this
envelope and the ndjson terminal `completed` record. `run_state` is
declared via the optional `RunStateDeclarer` capability
(`output.AsRunStateDeclarer`) or derived from a `RunStateProvider`
payload on the first Result, and renders in every mode (see the mode
table above), not only the json envelope.

## Compression primitives

A fuller renderer's engine owns three pure compression primitives — no I/O, no global
state, input slices never mutated: `Collapse(records, specs)` groups
records under caller-supplied key functions and replaces group overflow
with an `Elision` record whose `Identities` keep every elided item
reachable in-mode (never re-execute to recover a fact); `FilterTier`/
`TierCounts` project records by declared importance (`Tiered`), with the
verbosity ladder's ceiling resolved by `TierPolicyForVerbosity` and
carried in `FmtArgs.TierCeiling` (zero value derives from Verbosity);
`Summarize` derives a bounded plain-text summary from records plus the
run state, and `DeriveText` renders the full text view from a record
stream. Payloads that implement `RecordStreamProvider` (and no
`DataFormattable`) get every view derived from their records; payloads
keeping `FormatData`/`FormatText` are unchanged.

## Output Discipline (rule context)

The rationale behind the output-discipline rule, kept here so the rule can stay a pointer.

### Rationale

Raw print calls bypass the output mode abstraction, breaking JSON and NDJSON output formats. They also prevent deterministic testing since test code cannot intercept global stdout writes. The runtime's embedded output interface centralizes stream routing and enables injection of buffers for testing.

### Stream Separation

| Stream | Content | Examples |
|--------|---------|----------|
| stdout | Final result payload (text, JSON, NDJSON, yamlish) | Query results, generated output, computed values |
| stderr | Diagnostics, progress, prompts | Spinners, progress bars, warnings, errors, interactive UI |

Bubble Tea programs render to stderr so that piped stdout remains machine-readable.

### Constraints

- Commands receive runtime via `runtime.FromContext(ctx)` -- use it for all rendering
- Use `kong.Writers(stdout, stderr)` to redirect Kong's help and error output
- Progress indicators (spinners, bars) write exclusively to stderr
- `Result()` writes exclusively to stdout
- `Fatal()` writes to stderr and exits with a non-zero code

### Examples

**Correct**:
```go
func (c *ListCmd) Run(ctx context.Context) error {
    rt, err := runtime.FromContext(ctx)
    if err != nil {
        return runtime.Exitf(runtime.ExitSoftware, "runtime: %v", err)
    }
    items, err := c.fetch()
    if err != nil {
        return err
    }
    rt.Result(items)
    return nil
}
```

**Incorrect**:
```go
func (c *ListCmd) Run(ctx context.Context) error {
    items, _ := c.fetch()
    fmt.Println(items) // Bypasses output mode, breaks JSON output
    return nil
}
```

### Exceptions

Standalone utility scripts that produce only plain text and will never need JSON output may use `fmt` directly. This exception does not apply to any command within a Kong-structured application.

## Stdout verbosity precedence

Stdout verbosity resolves with a fixed precedence — quiet > an explicit stdout-verbosity setting > a deprecated verbose alias > the default (warn); the explicit setting outranks the deprecated alias, and the stdout axis is separate from log verbosity; explicitly requested debug logging bypasses a quiet output delegate and writes straight to stderr, because a quiet delegate that suppresses all log methods would otherwise drop the records silently.

## Related

[Kong Command Parsing](../../grammar/reference/kong_parsing.md)
[Bubble Tea and Bubbles](../../tui/reference/bubbletea_bubbles.md)

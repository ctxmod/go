# Output Discipline

Route all output through the runtime's output interface, obtained via `runtime.FromContext(ctx)`. Never call `fmt.Println`, `fmt.Fprintf(os.Stdout, ...)`, or `log.Println` directly from command handlers — raw print calls bypass the output mode abstraction and break JSON and NDJSON output.

stdout carries the final result payload only (text, JSON, NDJSON, yamlish). stderr carries diagnostics, progress, prompts, and interactive UI. `Result()` writes exclusively to stdout; `Fatal()` writes to stderr and exits non-zero; progress indicators and Bubble Tea programs render to stderr.

One exception: standalone utility scripts that produce only plain text and will never need JSON output may use `fmt` directly. It does not apply to any command within a Kong-structured application.

- the full `Output` interface, the six output modes, formatting interfaces, the stream-separation and constraint tables, the precedence that stdout verbosity settings resolve by, and the worked correct/incorrect examples — [Output Interface](../reference/output_modes.md)
- how `Runtime` embeds `Output` and reaches commands through context — [Runtime](../reference/runtime_injection.md); `kong.Writers(stdout, stderr)` for redirecting Kong's own help and error output — [Kong Command Parsing](../../grammar/reference/kong_parsing.md)

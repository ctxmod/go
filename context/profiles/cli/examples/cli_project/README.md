# cliproj — example Go CLI project

A complete, buildable CLI that applies the go `cli` profile: Kong grammar,
koanf configuration layering, a mode-aware renderer, and a context-bound
runtime with typed exit codes and lifecycle hooks. The module path
`example.com/cliproj` is a placeholder; rename it when you copy the project.

```sh
go build ./...
go vet ./...
go run ./tools/cliproj/cmd --help
go run ./tools/cliproj/cmd list --output json
go run ./tools/cliproj/cmd deploy service --target prod --scale 3 --output yamlish
go run ./tools/cliproj/cmd demo --fast -v
```

## Layout

| Path | Holds |
| --- | --- |
| `tools/cliproj/cmd/main.go` | Entry point: `os.Exit(app.Run(os.Args))` |
| `tools/cliproj/internal/app/root.go` | Root grammar, global flags, and the bootstrap (`loadConfig`, `setupOutput`, `RunCommand`) |
| `tools/cliproj/internal/app/<command>/cmd.go` | One command per package, each with `Run(ctx context.Context) error` |
| `tools/cliproj/internal/db/job` | A tool-local keyed-record store (one atomic JSON file per record) |
| `pkg/config` | koanf loader in explicit precedence order, Kong flag extraction, typed globals |
| `pkg/output` | The renderer: six modes, stdout/stderr discipline, formatting interfaces, slog handler, capture file |
| `pkg/runtime` | The `Runtime` struct, paths, session IDs, exit codes, `RunCommand` lifecycle; `aisimple` is a domain runtime for the AI commands |
| `pkg/ai` | A tool abstraction defined once and adapted to an LLM agent loop (goai) and an MCP server (go-sdk), with provider resolution and retry |

## What each command demonstrates

- `list`, `version` — the basic command shape: runtime from context, `Result` on stdout, diagnostics on stderr.
- `demo` — progress, checkpoints, and success messages across phases, interruptible by Ctrl-C through the signal context.
- `deploy`, `deploy service` — a command group whose flags are declared once on the parent and read by the child through a bound interface (redeclaring them would make Kong panic at parser construction), and a parent `Run` guarded with `kctx.Selected()` because Kong runs every node of the selected path.
- `tool list|schema|run` — a nested command tree over a tool registry, JSON input from a flag or stdin, and a table payload that renders through `Result` with its own text view.
- `agent` — an agent loop with tools, settings from the `aisimple` domain runtime, and key-value logging through the slog handler.
- `serve mcp` — a long-running server with an enum-selected transport (stdio or HTTP) that shuts down when the context is canceled.

## Configuration

Settings resolve, lowest precedence first: built-in defaults, `configs/default.yaml` and `configs/profile/<profile>.yaml` next to the executable, `~/.config/cliproj/config.yaml`, `cliproj.yaml` and `.cliproj.yaml` in the working directory, `--config` files, `.env` files (working directory, then user), `--env-file` files, `CLIPROJ_*` environment variables (`CLIPROJ_OUTPUT_MODE=json` sets `output.mode`), and finally the flags you set explicitly. The AI commands read `ai.provider`, `ai.model`, `ai.class`, `ai.key`, and `ai.steps`.

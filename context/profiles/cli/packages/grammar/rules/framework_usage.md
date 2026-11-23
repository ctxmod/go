# Framework Usage

Define all commands, flags, and arguments as struct fields with Kong tags. Dispatch through Kong's `Run()` method convention. Use Bubble Tea exclusively for interactive terminal views rendered on stderr.

Each command struct lives in its own package under `internal/app/{command}/` with a `cmd.go` file; never place two commands in the same package. Pass `tea.WithOutput(os.Stderr)` to `tea.NewProgram` so interactive views render on stderr and stdout stays clean for data payloads.

No exceptions: when the go `cli` profile applies, always use Kong for command parsing and the full package structure regardless of CLI complexity.

- Kong tags, binds, hooks, conventions and worked command examples — [Kong Command Parsing](../reference/kong_parsing.md); flag-derivation and grammar traps — [Kong grammar footguns](../reference/kong_footguns.md)
- where Bubble Tea, the output interface, koanf configuration, the runtime and project scaffolding are documented — [Go CLI Reference Map](../reference/cli_reference_map.md)

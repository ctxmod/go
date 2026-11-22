# Kong Command Parsing

Kong (`github.com/alecthomas/kong`) defines CLI grammar as Go struct fields annotated with tags. Commands map to embedded structs carrying `Run()` methods, flags map to non-command fields, and arguments use the `arg` tag. This eliminates switch/case dispatch and centralizes all argument definitions in type-checked Go code.

## Struct Tag System

Tags provide declarative configuration for every aspect of a command's surface area:

| Tag | Purpose | Example |
|-----|---------|---------|
| `cmd:""` | Mark struct as subcommand | `List ListCmd \`cmd:"" help:"List items"\`` |
| `arg:""` | Positional argument | `Path string \`arg:"" help:"File path"\`` |
| `help:"..."` | Help text | Any field |
| `default:"..."` | Default value | `Port int \`default:"8080"\`` |
| `env:"..."` | Environment variable fallback (first match wins) | `Token string \`env:"API_TOKEN,TOKEN"\`` |
| `envprefix:"..."` | Prefix applied to all sub-fields | `DB Database \`envprefix:"DB_"\`` |
| `short:"..."` | Short flag | `Verbose bool \`short:"v"\`` |
| `aliases:"..."` | Command aliases | `List ListCmd \`cmd:"" aliases:"ls"\`` |
| `enum:"..."` | Restrict to values | `Format string \`enum:"text,json,yamlish" default:"text"\`` |
| `required:""` | Mandatory field | `Name string \`required:""\`` |
| `hidden:""` | Omit from help | Internal flags |
| `negatable:""` | Allow `--no-X` | Boolean flags |
| `xor:"..."` | Mutually exclusive group | `--json` vs `--text` |
| `passthrough:""` | Stop parsing, pass remaining args | Wrapper commands |
| `type:"..."` | Select named type mapper | `Path string \`type:"path" help:"Config file"\`` |

The `envprefix` tag applies a prefix to all fields of an embedded struct. A `Database` struct with `envprefix:"DB_"` automatically resolves `DB_HOST`, `DB_PORT` without repeating the prefix on each field. Environment binding uses ordered fallback where the first match wins.

## Flag Derivation Rules (live-incident traps)

Kong derives flag names from Go field names: camelCase splitting at letter/digit class boundaries, dashes between segments, lowercased. A field `P256dh` derives `--p-256-dh` (not `--p256dh`); `WtExtensionOrigins` derives the plural `--wt-extension-origins`. Never trust the Go field name as the argv spelling.

- **`arg:""` fields are POSITIONAL.** The argv is a bare value, not `--name value`.
- **Name-position tokens in `kong:"..."` tags are inert.** `kong:"--foo,help='...'"` silently ignores the bare `--foo`: the tag hydrator reads only known keys, so the flag derives from the field name. This produced a deployed binary whose real flag was plural while every design record said singular. Use the `name:"foo"` struct tag instead — a properly declared name tag wins over the field-derived name. Never put dashes in a `name:"..."` value; it is used verbatim.
- **`envprefix` is per struct.** It prefixes only the fields of the struct it tags. A nested subcommand struct's `env:"X"` binds VERBATIM (`X`), never `PREFIX_X` — a full literal like `env:"MYTOOL_RUNTIME_ROOT"` is required for namespaced nested-command env vars. The root's own embedded flag struct resolves prefixed names correctly.

Pin the derived surface in a parser-level test with two small pure helpers over the parsed Kong model: an inspector that enumerates every flag name, positional, and post-prefix env binding, and an auditor that reports inert dashed name tokens and dashed declared names. The parsed model is available without running anything — `kong.New(&Root{})` returns a `*kong.Kong` whose `Model` node tree carries each node's derived flags and positionals:

```go
parser, err := kong.New(&Root{}, kong.Name(appName), kong.Description(desc))
bindings := inspectGrammar(parser.Model) // assert exact argv shape
findings := auditGrammar(parser.Model)   // assert no derivation defects
```

## Configuration File Loading

Configuration loading uses the project's shared `config.Load()` function with koanf, not Kong's built-in config system. Parse CLI args with Kong first, then extract explicitly-set flags via `config.ExtractUserFlags(ctx)` and pass them to the config loader:

```go
userFlags := config.ExtractUserFlags(kctx)
cfg, _, err := config.Load(config.LoadOptions{
    AppName:       appName,
    Cwd:           cwd,
    ConfigFiles:   cli.ConfigFiles,
    EnvFiles:      cli.EnvFile,
    FlagOverrides: userFlags,
})
```

This gives full control over the precedence stack. Kong handles command dispatch and validation; koanf handles the configuration graph. See config-layering for the full precedence order.

[Configuration Layering with koanf](../../config/reference/config_layering.md)

## Resolver Pipeline

Kong's parsing pipeline applies stages in order: defaults, resolvers, validation, execution. Resolvers provide default values from external sources. The `ConfigFlag` type can dynamically add resolvers during the `BeforeResolve` lifecycle hook, enabling runtime config file discovery.

## Bind System

The bind system provides lightweight dependency injection into `Run()` methods and hooks:

| Function | Behavior |
|----------|----------|
| `kong.Bind(value)` | Attach concrete value |
| `kong.BindTo(impl, iface)` | Map implementation to interface type |
| `kong.BindToProvider(fn)` | Lazy construction via provider function |
| `kong.BindSingletonProvider(fn)` | Cached singleton from provider |

Dependencies resolve against the selected command's arguments, parent commands, and explicit binds. This replaces global state and manual wiring in `main`:

```go
ctx := kong.Parse(&cli,
    kong.Bind(logger),
    kong.BindTo(outputImpl, (*output.Output)(nil)),
)
err := ctx.Run()
```

## Lifecycle Hooks

Hooks execute at defined points during parsing and allow injected dependencies:

| Hook | Phase |
|------|-------|
| `BeforeReset` | Before values reset to defaults |
| `BeforeResolve` | Before resolvers apply |
| `BeforeApply` | Before CLI args apply to grammar |
| `AfterApply` | After args apply and validation completes |
| `AfterRun` | After `Run()` returns |

Hooks implement as methods on command structs or parent structs. They accept injected dependencies as variadic arguments. Common uses: `AfterApply` for logging setup, `BeforeResolve` for config file registration.

## Writers and Help

`kong.Writers(stdout, stderr)` overrides output destinations, separating help text (stdout) from diagnostics (stderr). This enables deterministic testing with `bytes.Buffer` instances:

```go
var stdout, stderr bytes.Buffer
kong.Parse(&cli, kong.Writers(&stdout, &stderr))
```

`ConfigureHelp(...)` customizes help formatting. `NoDefaultHelp()` disables the automatic `--help` flag for custom implementations.

## Type Mapping

Custom value decoding uses four mapper types with defined priority: named > kind > type > value.

| Mapper | Registration | Use Case |
|--------|-------------|----------|
| `NamedMapper` | `kong.NamedMapper("name", mapper)` | Custom types by tag name |
| `TypeMapper` | `kong.TypeMapper(reflect.Type, mapper)` | Specific Go type |
| `KindMapper` | `kong.KindMapper(reflect.Kind, mapper)` | Go kind (slice, map) |
| `ValueMapper` | Interface on the value type | Self-decoding types |

Builtin named mappers: `path` (OS path expansion), `existingfile`, `existingdir`, `counter` (repeated flag), `filecontent` (read file into string).

## Dynamic Commands

`DynamicCommand` registers commands at runtime for plugin-based CLI extension. Plugins register themselves during initialization, and Kong includes them in help output and dispatch.

## Framework Usage (rule context)

The rationale behind the framework-usage rule, kept here so the rule can stay a pointer.

### Rationale

Struct-based grammar eliminates manual argument parsing and switch/case dispatch. Kong resolves commands, flags, environment variables, and config files through a single pipeline. Separating interactive UI (Bubble Tea on stderr) from data output (stdout) preserves composability in shell pipelines.

### Kong Conventions

- Commands map to embedded structs with `cmd:""` tags
- Each command struct lives in its own package under `internal/app/{command}/` with a `cmd.go` file; never place two commands in the same package
- Flags map to struct fields with `help`, `default`, `env`, and `short` tags
- Positional arguments use the `arg:""` tag
- Dependency injection via `kong.Bind` and `kong.BindTo` passes runtime and context into `Run()` methods
- Lifecycle hooks (`BeforeApply`, `AfterApply`) handle cross-cutting concerns such as config loading and logging setup
- `kong.Writers(stdout, stderr)` separates help text from diagnostics

### Examples

**Correct** -- Kong struct with command in its own package:
```go
// internal/app/root.go
type Root struct {
    List    list.Cmd    `cmd:"" help:"List all items"`
    Watch   watch.Cmd   `cmd:"" help:"Watch items with live updates"`
}

// internal/app/list/cmd.go
type Cmd struct {
    All bool `help:"Include hidden items" short:"a"`
}

func (c *Cmd) Run(ctx context.Context) error {
    rt, err := runtime.FromContext(ctx)
    if err != nil {
        return runtime.Exitf(runtime.ExitSoftware, "runtime: %v", err)
    }
    items := c.fetch()
    rt.Result(items)
    return nil
}
```

**Incorrect** -- manual argument parsing:
```go
func main() {
    if len(os.Args) < 2 {
        fmt.Println("usage: app <command>")
        os.Exit(1)
    }
    switch os.Args[1] { // Manual dispatch duplicates what Kong provides
    case "list":
        // ...
    }
}
```

### Exceptions

None. When the go `cli` profile applies, always use Kong for command parsing and the full package structure regardless of CLI complexity.

## Related

[Output Interface](../../output/reference/output_modes.md)
[Bubble Tea and Bubbles](../../tui/reference/bubbletea_bubbles.md)
[Kong grammar footguns](kong_footguns.md)
[Configuration Layering with koanf](../../config/reference/config_layering.md)

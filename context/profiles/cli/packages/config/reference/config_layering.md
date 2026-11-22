# Configuration Layering with koanf

koanf (`github.com/knadh/koanf/v2`) loads configuration from multiple sources -- files, environment variables, dotenv files, flags, remote stores -- and merges them into a single typed result. It complements Kong: Kong owns the command grammar, help surface, and flag definitions; koanf owns the effective configuration graph. Each `Load()` call merges one source into the instance in the order you call it, so your precedence policy becomes plain Go code rather than framework convention.

## Working Directory Resolution

The loader resolves the working directory once at startup using this order:

1. `INIT_CWD` environment variable (set by most shells on `cd`)
2. `PWD` environment variable (set by POSIX shells)
3. `"."` fallback

Store the resolved directory in the `LoadOptions` struct. All relative paths in config and env file discovery use this directory as the base.

## Precedence Model

koanf does not impose ordering. You encode precedence as call order. A clean low-to-high model:

| Priority | Source | Provider | Discovery |
|----------|--------|----------|-----------|
| 1 (lowest) | Built-in defaults | `confmap.Provider` | Hardcoded |
| 2 | Bundled base config (`configs/default.yaml`) | `file.Provider` + parser | Executable-relative |
| 3 | Profile config (`configs/profile/{name}.yaml`) | `file.Provider` + parser | `--profile` flag |
| 4 | User config (`~/.config/{appname}/config.yaml`) | `file.Provider` + parser | `$XDG_CONFIG_HOME` or `~/.config` |
| 5 | Project-local files (`{cwd}/{appname}.yaml`, `{cwd}/.{appname}.yaml`) | `file.Provider` + parser | Auto-discovered from cwd |
| 6 | Explicit `--config` files in listed order | `file.Provider` + parser | User-specified |
| 7 | Auto-discovered .env files | `file.Provider` + `dotenv.Parser` | See env discovery below |
| 8 | Explicit `--env-file` files in listed order | `file.Provider` + `dotenv.Parser` | User-specified |
| 9 | Process environment variables | `env.Provider` (v2) | `os.Environ()` |
| 10 (highest) | Explicitly-set CLI flags | `confmap.Provider` bridge | Kong-parsed |

Each successive `Load()` merges over the previous state. Later sources overwrite earlier ones for scalar values; nested maps merge recursively.

## Automatic File Discovery

Config and env files are discovered automatically in addition to any user-specified paths. Auto-discovered files load before their explicitly-specified counterparts so explicit paths always win.

### Config file discovery

The loader probes these paths in order, skipping any that do not exist:

```
{exe_dir}/configs/default.yaml                          # bundled defaults
{exe_dir}/configs/profile/{profile}.yaml                # if --profile is set
~/.config/{appname}/config.yaml                         # $XDG_CONFIG_HOME or ~/.config
{cwd}/{appname}.yaml                                    # project-local, visible
{cwd}/.{appname}.yaml                                   # project-local, hidden
```

Then any paths from `--config` flags load on top.

### .env file discovery

The loader probes these paths in order, skipping any that do not exist:

```
{cwd}/.env                                              # project-local
{cwd}/.{appname}.env                                    # app-specific, hidden
~/.config/{appname}/.env                                # user-level
```

Then any paths from `--env-file` flags load on top. Missing auto-discovered files produce no error -- only missing explicitly-specified files fail.

## Providers

Providers implement a two-method interface:

```go
type Provider interface {
    ReadBytes() ([]byte, error)  // raw bytes for parser consumption
    Read() (map[string]any, error) // pre-parsed nested map (pass nil parser)
}
```

The library ships 17+ providers as separate modules under `github.com/knadh/koanf/providers/`:

| Provider | Purpose |
|----------|---------|
| `file` | Reads a local file path, returns raw bytes |
| `fs` | Reads from `fs.FS` (Go 1.16+) |
| `env/v2` | Environment variables with prefix filtering and key/value transform |
| `confmap` | In-memory `map[string]any` -- used for defaults and bridge patterns |
| `structs` | Reads a Go struct via reflection and struct tags |
| `posflag` | `spf13/pflag.FlagSet` with default-aware merging (not used with Kong) |
| `rawbytes` | Direct `[]byte` input |
| `s3` | AWS S3 object |
| `vault/v2` | HashiCorp Vault |
| `consul/v2` | HashiCorp Consul |
| `etcd/v2` | CNCF etcd |
| `appconfig/v2` | AWS AppConfig |
| `parameterstore/v2` | AWS Systems Manager Parameter Store |
| `cliflagv2` | urfave/cli v2 flags |
| `cliflagv3` | urfave/cli v3 flags |

### env/v2 Provider

The v2 environment provider consolidates configuration into an `Opt` struct:

```go
env.Provider(".", env.Opt{
    Prefix: "MYAPP_",
    TransformFunc: func(k, v string) (string, any) {
        key := strings.ReplaceAll(
            strings.ToLower(strings.TrimPrefix(k, "MYAPP_")),
            "_", ".",
        )
        return key, v
    },
})
```

`Prefix` filters and strips a case-sensitive prefix. `TransformFunc` transforms both key and value (v1 only transformed the key). `EnvironFunc` overrides `os.Environ()` for testing. The first argument (`"."`) is the delimiter for splitting transformed keys into nested hierarchy.

### confmap Provider

```go
confmap.Provider(map[string]any{
    "host": "localhost",
    "port": 8080,
}, ".")
```

Loads a `map[string]any` directly. Used for two purposes: injecting programmatic defaults at the lowest priority, and bridging Kong-parsed values at the highest priority.

## Parsers

Parsers convert between raw bytes and nested maps. All implement a bidirectional interface:

```go
type Parser interface {
    Unmarshal([]byte) (map[string]any, error)
    Marshal(map[string]any) ([]byte, error)
}
```

Available parsers under `github.com/knadh/koanf/parsers/`:

| Parser | Notes |
|--------|-------|
| `json` | Standard JSON |
| `yaml` | Standard YAML |
| `toml` | TOML via BurntSushi |
| `toml/v2` | TOML via go-toml v2 |
| `dotenv` | `.env` file format, produces flat map |
| `hcl` | HashiCorp HCL, optional slice flattening |
| `hjson` | Relaxed JSON |
| `nestedtext` | NestedText format |

Providers returning raw bytes (like `file.Provider`) require a parser argument in `Load()`. Providers returning a pre-parsed map (like `confmap.Provider`, `env.Provider`) pass `nil` as the parser.

## Load vs Merge

### Load

```go
func (ko *Koanf) Load(p Provider, pa Parser, opts ...Option) error
```

Merges provider output into the instance. Nested maps merge recursively. Scalars, slices, and other leaf values overwrite. Optional `WithMergeFunc` overrides default merge behavior.

### Merge

```go
func (ko *Koanf) Merge(in *Koanf) error
func (ko *Koanf) MergeAt(in *Koanf, path string) error
```

Combines two `*Koanf` instances. `MergeAt` places the merged result under a specific key path. Useful for composing configuration from independently-built instances.

### StrictMerge

```go
k := koanf.NewWithConf(koanf.Conf{StrictMerge: true})
```

Records the type of each key on first load. Subsequent loads that provide a different type for the same key return an error instead of silently overwriting. This catches common conflicts like JSON parsing integers as `float64` while TOML preserves them as `int`.

## UnmarshalWithConf

```go
type UnmarshalConf struct {
    Tag           string                    // default: "koanf"
    FlatPaths     bool                      // treat tags as literal dot-delimited paths
    DecoderConfig *mapstructure.DecoderConfig // full mapstructure control
}
```

Struct tags default to `koanf`:

```go
type Config struct {
    Host  string `koanf:"host"`
    Port  int    `koanf:"port"`
    Debug bool   `koanf:"debug"`
}

var cfg Config
k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"})
```

`FlatPaths` mode treats tag values as literal flattened paths, allowing a flat struct to pull from scattered nested keys:

```go
type FlatConfig struct {
    Host string `koanf:"server.web.host"`
    Port int    `koanf:"server.web.port"`
}
```

The `DecoderConfig` field exposes `mitchellh/mapstructure/v2` for custom decode hooks (string to `time.Duration`), weakly-typed input handling, and error-on-unused-field semantics.

## Kong Integration Pattern

Kong does not use `pflag`. It implements its own struct-tag-driven flag system with the `Mapper` interface. There is no `*pflag.FlagSet` to pass to `posflag.Provider`.

The integration uses `confmap.Provider` as a bridge: parse CLI args with Kong first, extract only explicitly-set flags, then feed them into koanf at the highest priority. Kong handles command dispatch, help generation, and argument validation. koanf handles the full configuration graph.

### Automatic flag extraction

`config.ExtractUserFlags(ctx *kong.Context)` walks Kong's parsed context path and collects only the flags the user explicitly provided (on the command line, or through the flag's env binding). Flags the user did not set are absent from the returned map (no zero-value injection). Loading parameters (`--config`, `--env-file`, `--profile`) are excluded automatically. Do not use Kong's `Value.Set` as the test: applying a default marks a value set too. The example project's `pkg/config` reads the parse path instead, and keys each flag by a `koanf:"..."` struct tag (Kong keeps unknown tag keys) so a flag maps onto the same key its config file and environment spellings use.

```go
userFlags := config.ExtractUserFlags(kctx)
opts := config.LoadOptions{
    AppName:       appName,
    Cwd:           cwd,
    ConfigFiles:   cli.ConfigFiles,
    EnvFiles:      cli.EnvFile,
    Profile:       cli.Profile,
    FlagOverrides: userFlags,
}
cfg, sources, err := config.Load(opts) // sources: each layer, for provenance
```

No pointer fields are needed on the CLI struct. `ExtractUserFlags` inspects Kong's internal state to determine which flags were explicitly set versus defaulted.

### Global flags and inheritance

Flags on the root CLI struct are available to all subcommands without redefinition. Kong propagates root-level fields to every nested command. This means global flags such as `--output`, `--verbose`, `--config`, `--env-file`, `--profile`, and `--log-level` work on every command automatically. Subcommand structs define only their own specific flags.

## Bootstrap Sequence

A clean bootstrap builds configuration in one direction:

1. **Resolve working directory** from `INIT_CWD`, `PWD`, or `"."`.
2. **Parse CLI with Kong** into the root struct containing file paths, profile name, and runtime flags.
3. **Create runtime I/O** early -- stdout, stderr, provisional logger.
4. **Build `config.Request`** from Kong-parsed values and the resolved working directory.
5. **Load with koanf** in explicit precedence order (see loader skeleton below).
6. **Decode into typed `config.Config`** via `UnmarshalWithConf`.
7. **Validate** the decoded config (required fields, value ranges, mutual exclusions).
8. **Create final runtime** -- logger, renderer, debug workspace from validated config.
9. **Run command** with injected runtime dependencies via Kong's bind system.

Steps 1-4 are fast and deterministic. Step 5 is where precedence policy lives as code. Steps 6-7 produce a validated typed config. Steps 8-9 use that config to build the runtime and dispatch.

## Source Provenance

koanf has no built-in mechanism to track which provider supplied each key. After merging, the internal representation is a single `map[string]any` with no provenance metadata.

The practical approach for an `inspect config` command: load each source into a separate `*Koanf` instance, merge them into the final instance, and retain the individual instances for comparison. Use `Exists()` on each instance to determine which source contributed a given key. The last instance where `Exists()` returns true for a key is the effective source.

```go
sources := []struct {
    Name string
    K    *koanf.Koanf
}{
    {"defaults", kDefaults},
    {"config-file", kFile},
    {"env", kEnv},
    {"flags", kFlags},
}

// For each key, walk sources in reverse to find effective source
for _, key := range kFinal.Keys() {
    for i := len(sources) - 1; i >= 0; i-- {
        if sources[i].K.Exists(key) {
            fmt.Printf("%s = %v (from %s)\n", key, kFinal.Get(key), sources[i].Name)
            break
        }
    }
}
```

## Loader Skeleton

A complete `Load` function demonstrating the full precedence stack with auto-discovery:

```go
func Load(req LoadOptions) (*koanf.Koanf, error) {
    k := koanf.New(".")

    // 1. Built-in defaults (lowest priority)
    if err := k.Load(confmap.Provider(map[string]any{
        "host":      "localhost",
        "port":      8080,
        "log.level": "info",
        "output":    "text",
    }, "."), nil); err != nil {
        return nil, fmt.Errorf("defaults: %w", err)
    }

    // 2. Bundled base config (executable-relative)
    if err := loadFileIfExists(k, filepath.Join(req.ExeDir, "configs", "default.yaml")); err != nil {
        return nil, fmt.Errorf("base config: %w", err)
    }

    // 3. Profile-specific config
    if req.Profile != "" {
        path := filepath.Join(req.ExeDir, "configs", "profile", req.Profile+".yaml")
        if err := loadFileIfExists(k, path); err != nil {
            return nil, fmt.Errorf("profile config: %w", err)
        }
    }

    // 4. User config ($XDG_CONFIG_HOME or ~/.config)
    userDir := configDir(req.AppName)
    if err := loadFileIfExists(k, filepath.Join(userDir, "config.yaml")); err != nil {
        return nil, fmt.Errorf("user config: %w", err)
    }

    // 5. Project-local config files (auto-discovered from cwd)
    for _, name := range []string{req.AppName + ".yaml", "." + req.AppName + ".yaml"} {
        if err := loadFileIfExists(k, filepath.Join(req.Cwd, name)); err != nil {
            return nil, fmt.Errorf("local config %s: %w", name, err)
        }
    }

    // 6. Explicit --config files (user-specified, must exist)
    for _, path := range req.ConfigFiles {
        if err := k.Load(file.Provider(path), parserForExt(path)); err != nil {
            return nil, fmt.Errorf("config file %s: %w", path, err)
        }
    }

    // 7. Auto-discovered .env files
    for _, name := range []string{".env", "." + req.AppName + ".env"} {
        loadFileIfExists(k, filepath.Join(req.Cwd, name))
    }
    loadFileIfExists(k, filepath.Join(userDir, ".env"))

    // 8. Explicit --env-file files (user-specified, must exist)
    for _, path := range req.EnvFiles {
        if err := k.Load(file.Provider(path), dotenv.Parser()); err != nil {
            return nil, fmt.Errorf("env file %s: %w", path, err)
        }
    }

    // 9. Process environment
    if err := k.Load(env.Provider(".", env.Opt{
        Prefix: strings.ToUpper(req.AppName) + "_",
        TransformFunc: func(key, val string) (string, any) {
            k := strings.ReplaceAll(
                strings.ToLower(strings.TrimPrefix(key, strings.ToUpper(req.AppName)+"_")),
                "_", ".",
            )
            return k, val
        },
    }), nil); err != nil {
        return nil, fmt.Errorf("env: %w", err)
    }

    // 10. Kong-parsed flag overrides (highest priority, only non-nil pointers)
    if len(req.FlagOverrides) > 0 {
        if err := k.Load(confmap.Provider(req.FlagOverrides, "."), nil); err != nil {
            return nil, fmt.Errorf("flag overrides: %w", err)
        }
    }

    return k, nil
}
```

The `LoadOptions` struct carries all context extracted from Kong and the environment:

```go
type LoadOptions struct {
    AppName       string         // application name for file discovery
    Cwd           string         // resolved working directory
    ExeDir        string         // executable directory for bundled configs
    Profile       string         // --profile value
    ConfigFiles   []string       // --config values (explicit paths)
    EnvFiles      []string       // --env-file values (explicit paths)
    FlagOverrides FlagOverrides  // only explicitly-set flags from Kong
}
```

Working directory resolution happens before the request is built:

```go
func resolveCwd(flag string) string {
    if flag != "" {
        return flag
    }
    if v := os.Getenv("INIT_CWD"); v != "" {
        return v
    }
    if v := os.Getenv("PWD"); v != "" {
        return v
    }
    return "."
}

func configDir(appName string) string {
    if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
        return filepath.Join(dir, appName)
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", appName)
}

func loadFileIfExists(k *koanf.Koanf, path string) error {
    if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
        return nil
    }
    return k.Load(file.Provider(path), parserForExt(path))
}

func parserForExt(path string) koanf.Parser {
    switch filepath.Ext(path) {
    case ".yaml", ".yml":
        return yaml.Parser()
    case ".toml":
        return toml.Parser()
    case ".json":
        return json.Parser()
    case ".env":
        return dotenv.Parser()
    default:
        return yaml.Parser()
    }
}
```

## Related

[Kong Command Parsing](../../grammar/reference/kong_parsing.md)
[Output Interface](../../output/reference/output_modes.md)

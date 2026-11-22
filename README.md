# go

A ctxmod context module for Go
development, published as `github.com/ctxmod/go/context`. It requires
`github.com/ctxmod/engineering/context` for the language-neutral
principles it builds on.

## Profiles

| Profile | Selected | Holds |
| --- | --- | --- |
| `core` | by default | API and interface discipline, error design and wrapping, package decomposition with planning and refactoring procedures, design for testability and the testing procedure, validation procedures (API surface, dependency analysis, documentation quality, package structure), runtime memory tuning, and silent-failure pitfalls |
| `cli` | opt-in, extends `core` | Command-line tools: Kong grammar and its footguns, koanf configuration layering, mode-aware output and stream discipline, a context-bound runtime with typed exit codes, Bubble Tea, project scaffolding, and a buildable example project (`examples/cli_project`) |
| `services` | opt-in, extends `core` | Long-lived networked services: resilience, interceptors and tracing, RPC error semantics, and contract governance |

## Selecting

Require the module from your repository's `context/module.yaml`:

```yaml
requires:
  - github.com/ctxmod/go/context
```

`core` is the module's default profile. Select an opt-in profile by its
full name, for example by extending it from one of your own profiles:

```yaml
extends:
  - github.com/ctxmod/go/context/profiles/cli
```

## Checking

```sh
ctxmod vet --root .
```

# API Surface Analysis

Validate that a Go package's exported API remains within design principle thresholds. This procedure measures exported symbol count, entry point count, options count, interface method count, and internal code percentage, then classifies each metric as pass, warn, or fail.

## Checks

Six checks compose the full API surface analysis. Five are automated; one is semi-automated.

| Check | Method | Tool | Pass | Warn | Fail |
|-------|--------|------|------|------|------|
| Exported symbol count | Automated | Custom `go/analysis` analyzer | <50 | 50-100 | >100 |
| Entry point count (New* functions) | Automated | Custom `go/analysis` analyzer | 1-2 | 3 | >3 |
| Options per constructor | Automated | Custom `go/analysis` analyzer | <10 | 10-20 | >20 |
| Interface method count | Automated | Custom `go/analysis` or `revive` | <5 | 5-10 | >10 |
| Internal code percentage | Automated | LOC script (`internal/` vs total) | >50% | 30-50% | <30% |
| Unused exports | Semi-automated | `staticcheck` | 0 | 1-5 | >5 |

Thresholds converge independently across architecture, consumer research, and infrastructure dimensions. Standard library packages validate the ranges: most expose one or a few central types with on the order of dozens of exported identifiers. Packages approaching 50-100 exports commonly split into sub-packages.

## Automated Validation

### Symbol Counting

No standard linter counts exported symbols against a threshold. Build a custom analyzer using the `go/analysis` framework. The analyzer walks `ast.File` declarations, counts exported functions, types, constants, and variables, then reports diagnostics when counts exceed configured thresholds.

```go
// Skeleton: count exported symbols per package
for _, file := range pass.Files {
    for _, decl := range file.Decls {
        if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.IsExported() {
            exportedFuncs++
        }
    }
}
```

Register the analyzer as a golangci-lint module plugin for CI integration. The module plugin system (introduced in v1.57 and carried into the v2 line) compiles custom analyzers into a custom binary without upstream merging: `.custom-gcl.yml` names the golangci-lint version and the plugin modules, `golangci-lint custom` builds the binary, and the v2 `.golangci.yml` enables the linter with `type: "module"` under `linters.settings.custom.<name>`.

### Entry Point Detection

Count functions matching the `New*` naming convention that return package-defined types. Multiple entry points signal unclear canonical paths and increase time-to-first-success for consumers.

### Options Counting

Count exported functions matching the `With*` naming convention or returning an `Option`-typed value. This check detects decision fatigue: beyond 10 options, consumers struggle to identify the relevant subset. Beyond 20, the package likely conflates multiple concerns.

### Interface Method Audit

Use `revive` with the `exported` rule or a custom analyzer to count methods per interface declaration. Single-method interfaces (`io.Reader`, `io.Writer`) compose cleanly. Interfaces exceeding 5 methods weaken the abstraction and resist consumer-defined alternatives.

### Internal Code Percentage

Measure lines of code under `internal/` as a proportion of total package LOC. Libraries should place more than 50% of code in `internal/` or unexported identifiers. Modern backend services reach 70-90%.

```sh
# Simple LOC measurement
internal_loc=$(find ./internal -name '*.go' | xargs wc -l | tail -1 | awk '{print $1}')
total_loc=$(find . -name '*.go' | xargs wc -l | tail -1 | awk '{print $1}')
echo "Internal percentage: $(( internal_loc * 100 / total_loc ))%"
```

### Unused Export Detection

Run `staticcheck` to identify exported symbols with no external consumers. Unused exports inflate API surface without serving any consumer need.

## Manual Review

The semi-automated unused export check requires judgment: a symbol may appear unused in the current repository while external consumers depend on it. Verify against known consumers before unexporting.

Review the overall API shape in `go doc` output. Automated checks catch individual threshold violations. Manual review catches qualitative problems: unclear naming hierarchies, mixed abstraction levels, and missing progressive disclosure structure.

## Remediation

Each check failure maps to specific corrective actions.

**Exported symbol count exceeds threshold**: Audit each export. Identify symbols used only within the package or its tests. Unexport them and move implementation details to `internal/` sub-packages. Target a reduction of 5-10 symbols per pass.

**Too many entry points**: Consolidate to a single canonical constructor with the options pattern. Replace undifferentiated constructors (`NewLoader`, `NewConfigManager`, `LoadWithProfiles`) with one constructor accepting options. Retain alternative constructors only when they encode genuinely different execution modes.

**Excessive options per constructor**: Group related options into config structs. Move advanced options (watching, credential resolution, connection pooling) to explicit configuration through sub-package types. Target fewer than 10 options on the primary constructor.

**Large interfaces**: Split along consumer usage boundaries. Consumers rarely call all methods on a large interface. Extract smaller interfaces matching actual call patterns, and let the concrete type satisfy multiple interfaces through implicit implementation.

**Low internal code percentage**: Move non-public implementation details behind `internal/`. Parsing utilities, format conversions, internal state management, and intermediate data types belong there.

**Unused exports**: Unexport symbols confirmed as internally-only. For symbols with uncertain external usage, deprecate first with a release cycle before removal.

## Tooling Summary

| Tool | Purpose | Integration |
|------|---------|-------------|
| Custom `go/analysis` analyzer | Symbol, entry point, and options counting | `golangci-lint run` (module plugin) |
| `revive` | Interface method count, documentation presence | `golangci-lint run` (built-in linter) |
| `staticcheck` | Unused export detection | `golangci-lint run` (built-in linter) |
| `go doc` | Manual API shape review | CLI |
| `apidiff` | Breaking change detection between versions | CI pipeline |

Three custom analyzers close the automation gap: (1) API surface analyzer for symbol and entry point counting, (2) options complexity analyzer for per-constructor option counting, (3) consumer-interface analyzer for detecting producer-defined interfaces that should be consumer-owned. Until these exist, substitute manual `go doc` review against the thresholds table.

## Sources

- https://golangci-lint.run/docs/plugins/module-plugins/ — `.custom-gcl.yml`, `golangci-lint custom`, and enabling a module plugin in a v2 configuration (golangci-lint v2 line; accessed 2026-11-22)

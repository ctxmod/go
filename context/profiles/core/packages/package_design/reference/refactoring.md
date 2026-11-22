# Package Refactoring

A step-by-step procedure for refactoring existing Go packages toward design principle compliance. The workflow prioritizes backward-compatible changes, defers breaking changes to later phases, and uses validation procedures to measure progress at each stage.

## Prerequisites

Before starting refactoring, establish three baselines.

**Validation baseline**: Run all four validation procedures to quantify current violations. Record metrics for exported symbol count, entry points, options count, interface method count, internal code percentage, dependency direction violations, and documentation coverage. This baseline measures refactoring progress.

[API Surface Analysis](../../validation/reference/api_surface.md)
[Package Structure Validation](../../validation/reference/package_structure.md)
[Dependency Analysis](../../validation/reference/dependency_analysis.md)
[Documentation Quality Validation](../../validation/reference/documentation_quality.md)

**Consumer analysis**: Identify every package that imports the refactoring target. Catalog which exported symbols each consumer actually uses. Symbols with zero external consumers are immediate candidates for unexporting. Use `goda` or module-level grep to build the consumer map.

**Breaking change assessment**: Classify each planned change as backward-compatible or breaking. Backward-compatible changes (unexporting unused symbols, adding `internal/` packages, improving documentation) proceed immediately. Breaking changes (renaming types, moving packages, removing exports) require a migration plan.

## Refactoring Workflow

Execute these steps in order. Each step produces backward-compatible changes except where noted. Run validation after each step to confirm progress.

### Step 1: Analyze Current State

Run validation procedures against the target package. Collect all warn and fail metrics into a prioritized list. The metrics with the largest overshoot relative to thresholds receive highest priority.

[Package Decomposition](package_decomposition.md)
[Minimal Public API](../../api_design/reference/minimal_api.md)

### Step 2: Identify Principle Violations

Map each failed metric to the principle it violates. A package with 150 exported symbols violates both decomposition (>100 symbols signals decomposition) and minimal API (>100 fails threshold). Group violations by principle to reveal whether the package needs structural reorganization, API surface reduction, or both.

### Step 3: Prioritize Refactoring Targets

Order changes by two criteria: backward compatibility first, impact second. Apply this priority sequence:

| Priority | Change Type | Compatibility | Typical Impact |
|----------|------------|---------------|----------------|
| 1 | Unexport unused symbols | Compatible | Reduces symbol count directly |
| 2 | Move helpers to `internal/` | Compatible | Increases internal percentage |
| 3 | Add missing documentation | Compatible | Fixes documentation validation |
| 4 | Consolidate entry points | Breaking | Reduces entry point count |
| 5 | Group options into config structs | Breaking | Reduces options count |
| 6 | Extract sub-packages | Breaking | Resolves decomposition violations |
| 7 | Relocate interfaces to consumers | Breaking | Fixes interface ownership |

### Step 4: Execute Compatible Changes

Complete all backward-compatible refactoring before introducing any breaking changes.

**Unexport unused symbols**: For each exported symbol with zero external consumers, lowercase the identifier. Run `go build ./...` (and `go vet ./...`, which also compiles test files) to verify no compilation errors. Symbols used only in tests within the same package can also be unexported.

**Extract to `internal/`**: Move implementation-detail types and helper functions into `internal/` sub-packages. Group related internals by concern (e.g., `internal/parse`, `internal/resolve`). This reduces API surface without affecting any consumer.

**Add documentation**: Fill gaps identified by documentation quality validation. Add package comments, export comments, and `Example*` test functions for primary types.

[Error Type Design](../../errors/reference/error_type_design.md)

### Step 5: Extract Sub-Packages

When the package serves multiple conceptual domains (description requires "and"), extract each domain into a sub-package. Follow these extraction steps:

1. Identify type clusters with narrow inter-group dependencies
2. Create the sub-package directory
3. Move types, keeping the original package as a re-export facade during migration
4. Use type aliases (`type OldName = subpkg.NewName`) for backward compatibility
5. Update internal references
6. Deprecate the aliases with a target removal version

[Package Decomposition](package_decomposition.md)

### Step 6: Reduce API Surface

Consolidate multiple entry points to a single canonical constructor. Remove variant constructors (e.g., `LoadWithProfiles`, `NewConfigManager`) and fold their behavior into the primary constructor via options.

[Minimal Public API](../../api_design/reference/minimal_api.md)

### Step 7: Refactor Interfaces

Transition producer-defined interfaces to consumer-defined ones. For each interface defined in the producer package and consumed by a single external package, move the interface definition to the consumer. The concrete type already satisfies the interface implicitly.

[Interface Design Discipline](../../api_design/reference/interface_design.md)

### Step 8: Simplify Options

When a constructor exceeds 10 options, group related options into config structs. Move advanced options (watching, credential resolution, plugin configuration) to sub-package types passed directly to the constructor.

[Options Pattern Discipline](../../api_design/reference/options_pattern.md)

### Step 9: Improve Error Types

Ensure error types follow the sentinel/structured decision matrix. Replace string-matched errors with sentinel variables or structured types. Add `errors.Is`/`errors.As` support where missing.

[Error Type Design](../../errors/reference/error_type_design.md)

### Step 10: Validate Changes

Re-run all four validation procedures. Compare metrics against the step 1 baseline. All warn and fail metrics should show improvement. Any regression indicates an extraction that increased complexity without reducing surface area -- reconsider the decomposition boundary.

## Refactoring Strategies

### Incremental Decomposition

Extract one sub-package per iteration. After each extraction: run tests, update consumers, validate metrics. Avoid extracting multiple sub-packages simultaneously, as overlapping moves create merge conflicts and obscure regressions.

### API Deprecation

Mark symbols for removal using `// Deprecated:` comments that specify the replacement and the version when removal occurs. Provide a minimum two-release window between deprecation and removal. Automated tools (`staticcheck` SA1019) flag deprecated usage in consumer code.

### Internal Extraction

Move code to `internal/` aggressively. The threshold is whether an external consumer needs direct access. If three or more consumers import a symbol, it stays exported. Otherwise, it moves to `internal/`.

### Interface Migration

When relocating interfaces from producer to consumer: create the interface in the consumer package, update consumer imports, then remove the producer-side interface. The concrete type satisfies both definitions through structural typing, so the migration requires no changes to the implementation.

## Breaking Change Management

**When breaking changes are acceptable**: A breaking change proceeds when the refactoring resolves a critical-severity principle violation (e.g., >100 exported symbols) and no backward-compatible alternative exists. Always pair breaking changes with migration guidance.

**Deprecation timeline**: Deprecate in version N, remove in version N+2. Document the replacement in the deprecation comment. Publish a migration guide alongside the deprecation.

**Migration guides**: For each breaking change, document: what changed, why it changed, old usage, new usage, and automated migration steps (if available). `golangci-lint run --fix` applies simple mechanical fixes, and `gofmt -r` handles simple expression rewrites. `gopls` rename handles identifier changes across a module.

**Version considerations**: Breaking changes require a major version bump under semver. For packages not yet at v1, breaking changes are expected and do not require special versioning.

## Common Refactoring Patterns

| Before | After | Principle |
|--------|-------|-----------|
| Large monolithic package | Parent package + focused sub-packages | Decomposition |
| 150+ exported symbols | <50 exports, rest in `internal/` | Minimal API |
| Constructor returns interface | Constructor returns `*ConcreteType` | Interface design |
| 25 functional options | <10 options + config structs for groups | Options discipline |
| String-matched errors | Sentinel variables + structured types | Error design |
| Multiple entry points | Single canonical `New` + options | Minimal API |
| Test mocks in main package | `subpkg/mock` or `subpkg/test` | Decomposition |

## Case Study: Config Package

The glean `config` package (247 exported symbols, 25+ options, multiple entry points) demonstrates a package exceeding thresholds across all five principles. A phased refactoring plan:

**Phase 1 (compatible)**: Unexport symbols used only internally. Move test helpers (`MockKeyring`) to `config/keyring/mock`. Add missing documentation. Expected result: symbol count drops to ~150.

**Phase 2 (compatible)**: Extract `config/internal/` for profile management, credential resolution, and source loading logic. Expected result: internal code percentage rises above 50%.

**Phase 3 (breaking)**: Extract `config/keyring/` as a public sub-package for keyring implementations (`SystemKeyring`, `EnvKeyring`). Deprecate keyring types in root package with type aliases. Expected result: root package drops to ~80 symbols.

**Phase 4 (breaking)**: Consolidate entry points to `NewLoader` + `Load[T]`. Remove `LoadWithProfiles`, `NewConfigManager`. Group related options into config structs, reducing main constructor to <10 options. Expected result: root package reaches ~30-50 symbols, meeting all thresholds.

Progressive disclosure after refactoring:

```go
// Beginner: one-line loading
cfg, err := config.Load[MyConfig](ctx, "myapp")

// Intermediate: explicit configuration
loader, err := config.NewLoader("myapp",
    config.WithProfiles("production"),
)

// Expert: sub-package control
kr := keyring.NewSystem()
loader, err := config.NewLoader("myapp",
    config.WithKeyring(kr),
)
```

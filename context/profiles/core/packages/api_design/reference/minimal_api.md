# Minimal Public API

Export only symbols that consumers must access. Hide implementation details behind `internal/` packages and unexported identifiers.

The language-neutral KISS and DRY principles (in the engineering module's design principles) underpin this discipline.

## Rationale

API usability studies consistently show that larger APIs increase cognitive load and learning time. A systematic review consolidating 883 recommendations across 45 years of security API usability research identifies "minimize conceptual load" and "provide safe defaults" as primary design imperatives. Analysis of well-designed Go packages confirms this: most expose one or a few central types with on the order of dozens of exported identifiers. The Go team recommends keeping module functionality "focused and discrete" and placing implementation details under `internal/`.

Every exported symbol is a commitment. It enters the public API surface, appears in `go doc` output, and constrains future evolution. Unexported symbols can change freely across releases. The gap between "useful internally" and "required externally" determines how aggressively to hide.

## Thresholds

| Metric | Pass | Warn | Fail |
|--------|------|------|------|
| Exported symbols per package | <50 | 50-100 | >100 |
| Entry points (New* functions) | 1-2 | 3 | >3 |
| Internal code percentage | >50% | 30-50% | <30% |

These thresholds converge independently across consumer research, standard library analysis, and infrastructure tooling. Modern backend services place 70-90% of code under `internal/`.

## Progressive Disclosure

Structure the API across expertise levels so beginners see a small surface and experts access full control.

- **Beginner**: 1-3 concepts, a canonical entry point, strong defaults. Consumers write one or two lines and get working results.
- **Intermediate**: Configuration options, explicit types. Consumers customize behavior through the options pattern or config structs.
- **Expert**: Sub-packages, advanced interfaces, full control. Consumers import additional packages only when they need specialized functionality.

The standard library validates this layering: `net/http` provides `http.Get` for beginners, `http.Client` for intermediate use, and `Transport` configuration for experts.

## What to Export

Export a symbol when external packages require direct access to it:

- **Types in public function signatures**: Parameters, return types, and error types that callers must handle.
- **Constructors**: One or two `New*` functions serving as canonical entry points.
- **Interfaces for extensibility**: Small interfaces that consumers implement to extend behavior.
- **Sentinel errors and structured error types**: Values callers check with `errors.Is` or `errors.As`.

## What to Keep Internal

Hide everything else:

- **Implementation helpers**: Parsing utilities, internal state management, format conversions.
- **Test infrastructure**: Mock types, test fixtures, assertion helpers belong in test files or `internal/testutil`.
- **Intermediate types**: Structs used only to shuttle data between internal functions.
- **Configuration plumbing**: Internal option application, default resolution, validation helpers.

Move shared-but-private code to `internal/` sub-packages. The `internal/` directory prevents external import while allowing code reuse across sub-packages within the same module.

## Trade-offs

Aggressive hiding can force consumers to copy internal code when they need access the package author did not anticipate. The deciding test: does the consumer need direct access? If three or more external packages import a type, it belongs in the public API. Below that threshold, consumers can define their own narrow interface that the concrete type implicitly satisfies.

Discoverability suffers when too many symbols compete for attention in `go doc` output. A 30-symbol package lets consumers scan the full API in seconds. A 200-symbol package buries primary types among implementation details, increasing time-to-first-success.

## Deviations

**Extension point packages** (plugin registries, driver interfaces) legitimately export more symbols to support implementors. These packages exist specifically to expose contracts.

**Utility packages** containing pure functions with no state (like `strings` or `strconv`) tolerate higher symbol counts because per-symbol cognitive load is lower. Each function is independent and self-explanatory.

**Hub packages** serving as central coordination points (like `database/sql`) may reach 25-40 exports while maintaining a single conceptual focus.

## Remediation

When a package exceeds export thresholds:

1. **Audit exports**: Identify symbols used only within the package or its tests. Unexport them.
2. **Extract to internal**: Move helper types and implementation details to `internal/` sub-packages.
3. **Consolidate entry points**: Replace multiple undifferentiated constructors with a single canonical constructor accepting options.
4. **Relocate test helpers**: Move mock types and test utilities out of the main package into `internal/testutil` or dedicated test sub-packages.

These changes are backward-compatible for external consumers who should not depend on implementation details.

## Internal Visibility (rule context)

Extracted verbatim from the `go-internal-visibility-context` rule so that rule can stay a pointer.

### Rationale

Go's `internal/` directory enforces encapsulation at the compiler level -- importers outside the parent tree cannot access internal packages. This compiler guarantee eliminates accidental coupling to implementation details without relying on naming conventions or documentation. Hiding implementation behind `internal/` reduces the public API surface, keeping the package approachable for new consumers while preserving flexibility to refactor internals.

### Examples

**Correct**:
```
filesync/
  filesync.go       # Public API
  internal/
    index/          # Internal implementation
    render/         # Internal implementation
```

**Incorrect**:
```
filesync/
  filesync.go       # Public API
  index/            # Exposed to external importers
  render/           # Exposed to external importers
```

### Exceptions

Sub-packages intended for direct consumer import. A `config` package providing advanced configuration or a `types` package shared across multiple consumers may remain outside `internal/`.

### Tooling

Automated `go/analysis` analyzer to check internal code percentage per package tree. Flag packages below the target thresholds during CI.

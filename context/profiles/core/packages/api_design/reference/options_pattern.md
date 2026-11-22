# Options Pattern Discipline

Use functional options for public library APIs. Limit options per constructor. Switch to config structs for internal services and performance-critical paths.

## Rationale

Functional options (popularized by Dave Cheney in 2014) provide backward-compatible API extension, sensible defaults, and self-documenting configuration. The pattern remains the preferred approach for public Go libraries in 2024-2026 sources. However, unbounded option growth signals deeper package design problems. Option count serves as a reliable proxy for package complexity and cohesion.

Performance analysis shows functional options at 37 instructions versus 15 for config structs (2.5x overhead from closure allocations). This overhead is negligible for most constructors but measurable in hot paths. UX research shows discoverability degrades beyond 10 options due to sequential error hunting and loss of IDE-assisted exploration.

## Quantitative Thresholds

| Metric | Optimal | Warning | Decomposition Signal |
|--------|---------|---------|----------------------|
| Options per constructor | 3-8 | 10-20 | >20 |
| Options with conflicting effects | 0 | 1-2 | >2 |

These thresholds converge independently across pattern analysis, consumer research, architecture review, and infrastructure tooling dimensions.

**Smell detection**: A constructor with >20 options indicates the package handles too many concerns. The remedy is package decomposition, not option reduction alone. A constructor with 10-20 options warrants review for grouping related options into config structs or moving advanced options to sub-packages.

## Pattern Selection

Choose the configuration pattern based on context:

| Pattern | Best For | Performance | Boilerplate |
|---------|----------|-------------|-------------|
| Functional options | Public libraries, SDKs | Slower (closures) | Higher |
| Config structs | Internal services, hot paths | Faster (zero overhead) | Lower |
| Builder pattern | Ordered construction (niche) | Variable | Moderate |

Functional options provide the strongest backward-compatibility guarantees: adding a new `WithX` function never changes existing function signatures. Config structs expose structure directly, creating backward-compatibility constraints when fields change, but offer better IDE autocompletion, direct serialization from configuration files, and lower runtime cost.

## Validation and Defaults

Modern implementations return errors from option functions and accumulate validation results:

```go
type Option func(*Server) error

func NewServer(opts ...Option) (*Server, error) {
    s := &Server{Port: 8080} // Document defaults
    var errs []error
    for _, opt := range opts {
        if err := opt(s); err != nil {
            errs = append(errs, err)
        }
    }
    if len(errs) > 0 {
        return nil, errors.Join(errs...) // Go 1.20+
    }
    return s, nil
}
```

Every option should have a documented default value. Constructors with no options applied must produce a valid, working instance. This aligns with Go's "useful zero values" convention and reduces time-to-first-success for new consumers.

## When to Use Config Structs Instead

Switch from functional options to config structs when:

- The constructor is called in tight loops where closure allocation matters
- Configuration maps directly from files (JSON, YAML, TOML)
- The API is internal to a service and backward-compatibility pressure is low
- Option count exceeds 10 and most options are always set together

The `crypto/tls` package demonstrates this pattern: `tls.Config` is a large struct with many fields, validated at use time, with sensible zero-value defaults. This works because TLS configuration is typically loaded from files and the struct fields change infrequently.

## When to Deviate

**Generic functional options** (Go 1.18+) reduce boilerplate for packages with multiple types sharing option patterns. A single generic `Option[T]` type can serve several constructors without duplicating the option function signatures.

**Performance-critical constructors** called in tight loops should use config structs regardless of API surface. Profile first; switch only when allocation overhead appears in benchmarks.

## Evidence Base

Standard library analysis, Dave Cheney's continued advocacy (2024-2025), performance benchmarking of closure versus struct overhead, and API usability studies on decision fatigue at high option counts. Five independent research dimensions (architecture, patterns, consumers, dependencies, infrastructure) converge on the 10-option warning and 20-option decomposition thresholds.

[Package Decomposition](../../package_design/reference/package_decomposition.md)
[Minimal Public API](minimal_api.md)

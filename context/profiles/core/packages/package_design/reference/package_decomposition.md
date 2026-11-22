# Package Decomposition

Structure packages around single cohesive concepts. Split when conceptual boundaries emerge, not when arbitrary size limits are reached.

## Evidence

Standard library packages demonstrate cohesion-driven organization. `io` (~1,500 LOC) serves one purpose. `database/sql` separates the user facade from the driver SPI. `net/http` layers subsystems within a coherent mental model. Academic research on package cohesion metrics (CohP) confirms maintainability drops when inter-type dependencies grow faster than internal cohesion. Modern Go practice (2024-2026) places 70-90% of code under `internal/` for backend services.

## Thresholds

| Metric | Target | Warning | Decompose |
|--------|--------|---------|-----------|
| Exported symbols | <50 | 50-100 | >100 |
| Lines of code (package total) | <3,000 | 3,000-10,000 | >10,000 |
| Lines per file | 300-800 | 800-2,000 | >2,000 |
| Functional options per constructor | <10 | 10-20 | >20 |
| Conceptual domains | 1 | 2-3 | >3 |
| Entry points (constructors) | 1-2 | 3 | >3 |

These thresholds converge independently across architecture, pattern, consumer, and infrastructure research dimensions, providing high confidence in the ranges. Standard library examples validate them.

## Decision Criteria

Stay in one package when:

- All code serves one cohesive domain concept
- Consumers typically need all functionality together
- Internal helpers are not independently reusable

Move to sub-packages when:

- Package purpose requires "and" to describe (e.g., "loading AND credential management AND profile resolution")
- Type clusters form groups with narrow inter-group dependencies
- Test suites naturally separate into distinct fixtures
- Optional features drag in heavy dependencies consumers may not need

## Decomposition Patterns

Three patterns from the standard library address different decomposition needs.

**Facade/SPI**: `database/sql` + `database/sql/driver`. The user facade contains high-level types (`DB`, `Tx`, `Query`). The SPI sub-package contains low-level interfaces (`Conn`, `Stmt`, `Rows`). Driver implementations import only the SPI, keeping the facade independent of backends.

**Internal packages**: `net/http/internal` hides complexity (chunked encoding, ASCII validation) from the public API. Use `internal/` aggressively for non-API code. This approach keeps the public surface clean while maintaining modular implementation.

**Layered hierarchy**: The standard library enforces a strict acyclic dependency graph (`io` -> `net` -> `net/http`). Sibling packages never depend on each other. This prevents distributed monolith patterns where everything imports everything.

## Examples

**Cohesive package** -- a `resolve` package for a configuration tool (42 exported symbols, 9.4 KLOC): Approaches upper thresholds but maintains cohesion through single-purpose focus on tag parsing and resolution. All symbols serve the same conceptual domain. No decomposition needed.

**Decomposition candidate** -- an application `config` package (247 exported symbols, 8 KLOC): Exceeds thresholds across every metric. Mixes loading, profiles, credentials, keyring, and watching into one package. 25+ functional options confirm the decomposition signal. Recommended split:

```
config/                    # Core: interfaces, NewLoader, Load[T] (~30 symbols)
config/keyring/            # Keyring implementations (SystemKeyring, EnvKeyring)
config/keyring/mock/       # MockKeyring for testing
config/internal/           # Profile management, credential resolution, sources
```

## Trade-offs

Aggressive decomposition increases package count and import complexity. Follow the standard library pattern: coarse for tightly coupled concerns (`net/http` keeps core types together), fine-grained for optional or separable concerns (`net/http/httptest` separated for testing).

## When to Deviate

**Utility packages** (pure functions, no state) tolerate higher symbol counts because cognitive load per symbol is lower. `strings` and `strconv` exceed 50 exports without maintainability problems.

**Hub packages** serving as central coordination points (like `database/sql`) may legitimately reach 25-40 exports while maintaining single conceptual focus.

**Generated code** should be excluded from symbol count thresholds. Generated code follows generator conventions rather than hand-authoring principles.

# Dependency Analysis

Validate interface ownership patterns, coupling strength, constructor return types, error wrapping discipline, and dependency isolation. This procedure enforces dependency inversion through consumer-defined interfaces and measures coupling to detect architectural erosion.

## Checks

| Check | Method | Tool | Criteria |
|-------|--------|------|----------|
| Circular dependencies | Automated | Go compiler, `go-cyclic` | Zero cycles |
| Constructor return types | Automated | Custom `go/analysis` analyzer | Concrete types, not interfaces |
| Error wrapping at boundaries | Automated | `wrapcheck` (via `golangci-lint run`) | All cross-package errors wrapped |
| Import bloat | Semi-automated | `goda`, `depgraph` | No unnecessary transitive dependencies |
| Interface ownership | Semi-automated | Custom analyzer (future) | Interfaces defined in consumer packages |
| Test substitutability | Manual | Code review | Key dependencies injectable via constructor |

Automated checks run in CI without human judgment. Semi-automated checks use tooling to surface candidates but require human interpretation. Manual checks rely on code review criteria.

## Circular Dependencies

The Go compiler catches import cycles at compile time. `go-cyclic` provides granular diagnostics, outputting exact import chains that identify which types or functions create the coupling path.

```sh
go-cyclic ./...
```

**Pass**: Zero cycles detected.

**Remediation**: Extract shared types to a neutral third package (`domain`, `types`, or a parent package) containing only data structures and interfaces with no logic. If a child package needs a parent type, invert the dependency using consumer-defined interfaces rather than moving types upward.

## Constructor Return Types

A custom `go/analysis` analyzer scans `New*` and `Must*` functions and flags any that declare an interface return type. Constructors should return `*ConcreteType`; consumer-defined interfaces achieve testability and decoupling without constraining the producer's API evolution.

**Pass**: All constructors return concrete pointer types.

**Exception**: Abstract factory functions where runtime polymorphism selects among multiple implementations (e.g., `NewStorage(cfg) (Storage, error)` choosing between postgres and s3 backends).

## Error Wrapping

Errors crossing package boundaries must be wrapped with `fmt.Errorf("context: %w", err)`. The `wrapcheck` linter (via `golangci-lint run`) enforces this automatically.

**Pass**: All errors originating outside the current package are wrapped before returning.

**Remediation**: Add `fmt.Errorf` wrapping at every return of an external error. Use `%w` to preserve `errors.Is` and `errors.As` chains. Use `errors.Join` (Go 1.20+) when combining multiple errors.

## Interface Ownership

No existing tool validates consumer-defined interface placement. Until a custom analyzer is available, validate through code review:

1. Identify interfaces exported from producer packages.
2. Count how many consumer packages use each interface.
3. Flag interfaces used by only one consumer -- these are candidates for relocation to the consumer package.
4. Shared contracts used by three or more consumers belong in a neutral package.

**Pass**: Interfaces reside in consumer packages unless shared by three or more consumers.

**Remediation**: Move single-consumer interfaces to the consuming package. The producer continues to return its concrete type; the consumer defines the narrowest interface it needs.

## Coupling Metrics

Measure afferent coupling (Ca) and efferent coupling (Ce) to compute instability.

```sh
spm-go analyze ./...
```

| Metric | Formula | Interpretation |
|--------|---------|----------------|
| Afferent coupling (Ca) | Incoming dependencies | High Ca = stable, many dependents |
| Efferent coupling (Ce) | Outgoing dependencies | High Ce = unstable, many dependencies |
| Instability (I) | Ce / (Ca + Ce) | 0 = maximally stable, 1 = maximally unstable |

Stable packages (low instability) should depend on abstractions; unstable packages should be concrete. When a stable package depends on an unstable one, extract a shared interface or restructure.

`go-arch-lint` enforces directional dependency rules through YAML configuration:

```yaml
# go-arch-lint configuration
allow:
  service -> domain
  handler -> service
deny:
  domain -> service
  domain -> handler
```

## Import Bloat

`goda` traces why package A pulls in package B. `depgraph` visualizes dependency trees to identify clusters.

```sh
goda graph ./internal/... | dot -Tpng -o deps.png
```

**Review criteria**: Each direct dependency should serve the package's core purpose. Dependencies pulled in by a single optional feature are candidates for extraction into a sub-package, so consumers who do not need that feature avoid the import.

## Test Substitutability

Key external dependencies (databases, HTTP clients, file systems) should be injectable through constructor parameters, typically as interfaces. Review constructors to verify:

1. External dependencies appear as parameters, not hard-coded internally.
2. Parameters use interface types, allowing test doubles.
3. Fakes (stateful in-memory implementations) are preferred over mocks for behavioral verification.

**Pass**: All external dependencies injectable. Fakes available for integration-critical dependencies.

**Remediation**: Introduce an interface parameter in the constructor for each hard-coded dependency. Create fakes in `internal/testutil` or a dedicated test sub-package. Run shared test suites against both fake and real implementations to verify behavioral parity.

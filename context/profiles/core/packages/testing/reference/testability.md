# Testability as Emergent Property

Testable packages result from applying design principles -- interface segregation, dependency inversion, minimal API surface -- rather than from testing-specific techniques. Packages that require extensive scaffolding or deep mocking typically violate one or more of these principles. The remedy is design correction, not better test tooling.

Standard library packages confirm this relationship. They were designed for clarity and composability, not explicitly for testability, yet they remain among the easiest packages to test. Testability arrived as a side effect of sound design.

## Interface Granularity

Interface size correlates inversely with test isolation. Single-method interfaces allow trivial test double creation: one function to implement, one behavior to verify. Interfaces exceeding five methods require proportionally more maintenance in test doubles and create coupling between test code and interface evolution. When an interface gains a method, every test double must be updated regardless of whether the new method is relevant to the test.

Consumer-defined interfaces produce naturally small interfaces because each consumer declares only the methods it calls. This yields the narrowest possible contract per consumption site, making test double creation a minimal-effort operation.

[Interface Design Discipline](../../api_design/reference/interface_design.md)

**Consolidation threshold**: When three or more consumers define identical interfaces, consolidation into a shared package is warranted. Below that threshold, per-consumer interfaces maximize test isolation with acceptable duplication.

## Explicit Dependencies

Testable packages accept behavioral dependencies through constructor parameters rather than creating them internally. Constructor injection makes substitution explicit at the call site, visible in function signatures, and enforceable at compile time. Packages that instantiate their own database connections, HTTP clients, or file system handles inside constructors cannot be tested without those resources present.

This principle applies to packages with behavioral dependencies on external services or state. Pure computation packages -- those that accept data and return data without side effects -- gain nothing from constructor injection. The distinction matters: applying injection to pure computation adds parameter noise without testability benefit.

[Options Pattern Discipline](../../api_design/reference/options_pattern.md)

When constructor parameter counts grow, functional options provide a pressure relief valve. Optional dependencies wrapped in option functions maintain clean signatures while preserving substitutability. Required dependencies remain positional parameters.

## Test Forms and Placement

Go supports three test forms, and each reaches the package differently. An in-package test (`package foo` in a `_test.go` file) sees unexported symbols. An external test package (`package foo_test`) sees only the exported API, as a consumer does. A black-box test of a compiled binary invokes it as a subprocess and never imports the production code it exercises. Which forms a project commits, where implementation-coupled tests live, and how tests are selected and run is project testing policy; the design principles here hold under any of them.

## Visibility Boundaries and Internal Access

The `internal/` directory restricts import visibility for committed packages -- it does not restrict the questions a test may pose. The common misconception that internal packages are harder to exercise has no basis: a test that lives inside the parent tree of an `internal/` package may import it, and an in-package test of the internal package itself sees its unexported symbols too.

This removes a potential objection to aggressive use of `internal/` for hiding implementation. Packages can be simultaneously hidden from external consumers and thoroughly exercised by tests that live inside the module.

[Minimal Public API](../../api_design/reference/minimal_api.md)

## White-Box Reach as Design Signal

Tests that require access to unexported symbols indicate the package's public API lacks sufficient observability or control points. Occasional white-box reach for complex internal state transitions is acceptable -- an implementation-coupled test is exactly where an implementation-coupled question belongs. When white-box reach predominates, the package should be split so that internal components become the public API of smaller sub-packages.

This signal complements the decomposition thresholds defined in the package-decomposition principle. Where those thresholds measure structural metrics (symbol count, line count, conceptual domains), predominant white-box testing measures behavioral coupling between tests and implementation.

[Package Decomposition](../../package_design/reference/package_decomposition.md)

## Testability Barriers

Three design patterns consistently prevent test isolation:

| Barrier | Mechanism | Resolution |
|---------|-----------|------------|
| Hidden dependencies | Globals, singletons, package-level state | Accept dependencies through constructors |
| Wide interfaces | 10+ method contracts coupling test doubles to full surface | Split into composed single-purpose interfaces |
| Tight resource coupling | Direct instantiation of external resources inside constructors | Inject resource handles as interface parameters |

Each barrier has a design-level resolution. Testing workarounds (monkey-patching, build-tag test overrides, global variable replacement) treat symptoms while preserving the underlying coupling.

## Evidence

Academic research on testability refactoring (2024) confirms that design-level interventions produce more lasting testability improvements than test-level fixes. Community consensus across the Uber Go Style Guide, Google Go Style Guide, and Go standard library analysis converges on consumer-defined interfaces and constructor injection as the primary testability enablers. The structural typing system eliminates interface registration overhead, making small consumer-defined interfaces zero-cost at the production code level.

## Trade-offs

Strict adherence to constructor injection increases parameter counts and constructor complexity. Packages with many dependencies develop long constructor signatures that obscure the primary purpose. The options pattern mitigates this for optional dependencies, but packages with numerous required dependencies signal a cohesion problem -- the package likely handles too many concerns.

Aggressive interface granularity can scatter interface definitions across many consumer files. The consolidation threshold (three or more identical interfaces) provides a practical limit. Below that threshold, per-consumer interfaces provide superior test isolation with acceptable redundancy.

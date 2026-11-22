---
description: >-
  The Go development workflow library: package design principles plus
  the planning, refactoring, and testing procedures and the validation
  suite that measures them; consult on demand during Go package work.
binds_to_depth: main...
---

# Go Development Workflow

The Go workflow library defines how to design, implement, and validate Go packages; consult the relevant principle or procedure on demand during package work.

Design principles behind the convention rules:

- the language, standard-library, and tooling baseline the code is written against — the `go` directive, iterators, generics additions, `encoding/json/v2`, `os.Root`, and the `go fix` modernizers — [Modern Go Baseline](../baseline/reference/modern_go.md)
- functional options for public library APIs, config structs for internal and performance-critical paths — [Options Pattern](../api_design/reference/options_pattern.md)
- structuring packages around single cohesive concepts and when to split — [Package Decomposition](reference/package_decomposition.md)
- testability as a side effect of interface segregation, dependency inversion, and minimal API surface — [Testability](../testing/reference/testability.md)
- the out-of-repo consumer census: when other checkouts depend on this module's live working tree (for example through a `replace` directive pointing at a sibling directory), any public-API change needs its consumer set counted first — the in-repo `go build ./...` does not see those consumers

Implementation procedures:

- designing a package's structure, API surface, and internal organization before writing code — [Planning](reference/planning.md)
- refactoring existing packages toward design-principle compliance without breaking consumers — [Refactoring](reference/refactoring.md)
- writing and organizing tests, and where defect-focused effort goes — [Testing Procedure](../testing/reference/testing_procedure.md)

Validation procedures:

- exported-symbol, entry-point, options-count, and internal-percentage thresholds — [API Surface](../validation/reference/api_surface.md)
- interface ownership, coupling strength, constructor returns, and error-wrapping checks — [Dependency Analysis](../validation/reference/dependency_analysis.md)
- doc completeness, accuracy, and discoverability standards for `go doc` consumers — [Documentation Quality](../validation/reference/documentation_quality.md)
- dependency direction, `internal/` usage, cycle detection, and per-package cohesion — [Package Structure](../validation/reference/package_structure.md)

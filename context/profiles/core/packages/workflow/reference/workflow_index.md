# Go Development Workflow Index

Which principle, procedure or validation reference of the Go workflow library answers which question during package work.

Design principles behind the convention rules:

- the language, standard-library, and tooling baseline the code is written against — the `go` directive, iterators, generics additions, `encoding/json/v2`, `os.Root`, and the `go fix` modernizers — [Modern Go Baseline](../../baseline/reference/modern_go.md)
- functional options for public library APIs, config structs for internal and performance-critical paths — [Options Pattern](../../api_design/reference/options_pattern.md)
- structuring packages around single cohesive concepts and when to split — [Package Decomposition](../../package_design/reference/package_decomposition.md)
- testability as a side effect of interface segregation, dependency inversion, and minimal API surface — [Testability](../../testing/reference/testability.md)

Implementation procedures:

- designing a package's structure, API surface, and internal organization before writing code — [Planning](../../package_design/reference/planning.md)
- refactoring existing packages toward design-principle compliance without breaking consumers — [Refactoring](../../package_design/reference/refactoring.md)
- writing and organizing tests, and where defect-focused effort goes — [Testing Procedure](../../testing/reference/testing_procedure.md)

Validation procedures:

- exported-symbol, entry-point, options-count, and internal-percentage thresholds — [API Surface](../../validation/reference/api_surface.md)
- interface ownership, coupling strength, constructor returns, and error-wrapping checks — [Dependency Analysis](../../validation/reference/dependency_analysis.md)
- doc completeness, accuracy, and discoverability standards for `go doc` consumers — [Documentation Quality](../../validation/reference/documentation_quality.md)
- dependency direction, `internal/` usage, cycle detection, and per-package cohesion — [Package Structure](../../validation/reference/package_structure.md)

Runtime operation:

- setting `GOMEMLIMIT` and `GOGC` for a deployed binary, container-aware `GOMAXPROCS`, and why `ulimit -v` kills Go programs — [Memory Management](../../runtime/reference/memory_management.md)

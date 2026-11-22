# Package Testing

Procedure for writing and organizing Go tests. Tests exercise the package's exported interface from the consumer's perspective, concentrate effort where defect impact is highest, and use test doubles that survive internal refactoring. Claims about a compiled program's working behavior need a black-box test that runs the binary; package-level tests alone never establish them. Where a project keeps a stricter testing policy (where implementation-coupled tests live, which forms are committed, how tests are selected and run), that policy decides placement and invocation; the scope, double, and coverage guidance here holds under any policy.

## Testing Scope

### What to Test

Author tests against the package's exported interface from the consumer's perspective (`package foo_test` black-box style). This validates the contract, remains stable through refactoring, and produces tests that double as usage documentation. A working-behavior claim about the compiled program additionally requires a black-box test that invokes the compiled binary as a subprocess; static checks and package-level tests alone never establish it.

Concentrate testing effort on logic where defects carry real cost:

| Category | Examples |
|----------|----------|
| Conditional business logic | Branching paths, state machines, rule engines |
| Data transformation | Parsing, serialization, format conversion |
| Validation and error paths | Input validation, error propagation, sentinel checks |
| Concurrency patterns | Goroutine coordination, channel protocols, shared state |
| Algorithms with edge cases | Boundary conditions, overflow, empty inputs |

### What to Skip

Simple passthrough code, trivial getters, one-line wrappers, and configuration wiring offer low return on testing investment. If a function delegates to a single dependency call without branching, the dependency's own tests provide coverage.

### White-Box Reach as Signal

When tests require access to unexported symbols (`package foo` instead of `package foo_test`), the package's public API likely lacks sufficient observability or control points. Occasional white-box reach for complex internal state transitions is acceptable. Predominant white-box reach indicates the package should be split.

[Package Decomposition](../../package_design/reference/package_decomposition.md)

## Test Organization

### Test Placement

The Go convention places `_test.go` files alongside production code in the same directory; `go test` compiles them only for tests, and `go build` ignores them.

| Convention | Purpose |
|------------|---------|
| `foo_test.go` with `package foo_test` | Black-box test of the exported API (default) |
| `foo_internal_test.go` with `package foo` | White-box test needing unexported access (the exception) |
| `testdata/` | Fixture files (ignored by the build system) |

### Test Helpers

Mark helpers with `t.Helper()` so failure messages report the caller's location. Helpers shared across several test files of one package live in a shared `helpers_test.go`-style file; helpers confined to a single test belong in its own file.

### Table-Driven Tests

Use table-driven tests when multiple inputs exercise the same code path with varying expectations. Name each case descriptively for `t.Run` output. For cases that share no setup, separate test functions are clearer than a forced table.

```go
tests := []struct {
    name     string
    input    Input
    expected Output
    wantErr  bool
}{
    {"valid input", validInput, expectedOutput, false},
    {"empty input", emptyInput, zeroOutput, true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := pkg.Process(tt.input)
        // assertions
    })
}
```

Since Go 1.22 each loop iteration declares a fresh `tt` (the main module's `go` directive must be 1.22 or later), so the `tt := tt` copy before `t.Run` with `t.Parallel()` is dead code; `go fix` removes it.

### The Modern `testing` Surface

Prefer the standard-library mechanisms over hand-rolled equivalents; each removes a class of flaky or leaky test:

| Need | Mechanism (version) | Replaces |
|------|---------------------|----------|
| Context bound to the test's lifetime | `t.Context()` (1.24) — canceled just before `t.Cleanup` functions run | `context.WithCancel` plus a deferred cancel per test |
| Working directory change | `t.Chdir(dir)` (1.24) — restored after the test; incompatible with `t.Parallel` | manual `os.Chdir` with a deferred restore |
| Benchmark loop | `for b.Loop() { ... }` (1.24) — setup outside the loop is excluded from timing, and the body is kept from being optimized away (no longer blocks inlining since 1.26) | `for i := 0; i < b.N; i++` plus `b.ResetTimer` and sink variables |
| Time- and concurrency-dependent code | `synctest.Test(t, func(t *testing.T) {...})` (1.25) with `synctest.Wait()` and `synctest.Sleep(d)` (1.27) — a bubble with a fake clock that advances only when every goroutine in it is durably blocked; `httptest.NewTestServer` (1.27) serves inside a bubble | real `time.Sleep`, polling loops, injected clock interfaces added only for tests |
| Files a test produces for later inspection | `t.ArtifactDir()` (1.26), kept with `go test -artifacts` | ad-hoc writes into `testdata/` or `/tmp` |
| Structured test metadata and unprefixed log output | `t.Attr(k, v)` and `t.Output()` (1.25) | parsing free-form `t.Log` lines |
| Deterministic crypto randomness | `testing/cryptotest.SetGlobalRandom` (1.26) — the `crypto/...` random parameters are ignored since 1.26 | passing a seeded reader to crypto APIs |

`synctest` limits: mutex locking, network I/O, and system calls are not "durably blocked", so code blocked on them stalls the fake clock; channels, timers, and `WaitGroup`s created in a bubble panic when used from outside it; and `t.Run`, `t.Parallel`, and `t.Deadline` must not be called on the bubble's `*testing.T`.

Run concurrency-bearing packages with the race detector (`go test -race`) when a question touches shared state; a clean race run covers only the interleavings that executed.

### Integration-Style Tests

A test that needs an external resource (database, network service) must not run by default. Two mechanisms bound its cost: `//go:build integration` gates entire files at compilation (selected with `go test -tags integration ./...`), and `if testing.Short() { t.Skip("requires database") }` skips tests during short runs (`go test -short ./...`). Select them only when the question or acceptance criterion needs them.

## Test Double Patterns

### Selection by Verification Need

Choose test doubles based on what the test verifies, not on tool availability.

| Type | When to Use | Verification |
|------|-------------|-------------|
| Fake | Core domain logic, state matters | State: check end result |
| Stub | Predictable dependency responses | State: check caller behavior |
| Mock | External service boundaries | Interaction: check call sequences |
| Spy | Observation concerns (logging, events) | Interaction: check what was recorded |
| Dummy | Parameter exists but is irrelevant | None |

Prefer state verification ("the record was saved correctly") over interaction verification ("Save was called once with these arguments"). State verification survives refactoring; interaction verification breaks when implementation details change without affecting behavior.

### Hand-Rolled Fakes

For core domain testing, hand-rolled fakes (simplified working implementations using in-memory data structures) outperform generated mocks. Fakes decouple tests from call sequences, require no code generation, and read as usage documentation.

The function-field pattern provides per-test customization without external dependencies:

```go
type fakeStore struct {
    saveFn func(id string, data []byte) error
}

func (f *fakeStore) Save(id string, data []byte) error {
    return f.saveFn(id, data)
}
```

Reserve generated mocks for external service boundaries where interfaces are large, change frequently, or require call-order assertions.

[Interface Design Discipline](../../api_design/reference/interface_design.md)

### Test Double Placement

| Scope | Location |
|-------|----------|
| Single test | Inline in its file |
| Shared within one package | Shared `_test.go` file in the package directory |
| Shared across packages | An `internal/testutil`-style package (or a `<pkg>test` package such as `httptest` when consumers need it too) |

## Coverage Philosophy

Coverage measures execution paths, not correctness. High coverage with weak assertions provides false confidence. Tests exist to verify meaningful behavior and edge cases that prove the robustness of the software.

Write tests that assert specific behavioral expectations and contract guarantees. Focus on contract-level assertions, not implementation details. Prefer fewer high-value tests that catch real bugs over many low-value tests that duplicate coverage. Do not write redundant tests solely for coverage metrics or artificial confidence.

### Risk-Based Effort Allocation

Concentrate testing effort where defects carry real cost, and accept minimal testing on low-impact code:

| Risk Level | Testing Expectation | Examples |
|------------|---------------------|----------|
| Critical | Thorough behavioral coverage of all meaningful paths and edge cases | Payment processing, auth, data integrity |
| Standard | Solid coverage of primary behaviors and error paths | Business logic, API handlers |
| Low | Minimal — simple delegation needs few or no dedicated tests | Configuration wiring, simple passthrough |

### Coverage as a Discovery Tool

Use coverage reports to identify untested code paths, not as a target to maximize. Uncovered critical paths warrant investigation. Uncovered low-risk paths are acceptable. A coverage ratchet (coverage must not decrease) combined with risk-based judgment balances enforcement with pragmatism.

[API Surface Analysis](../../validation/reference/api_surface.md)

## Testing Workflow

1. **Identify scope**: List exported functions and methods. Classify each by risk category.
2. **Author tests**: Use `package foo_test` style. Start with the primary use case, then add error paths and edge cases.
3. **Create test doubles**: Build fakes for domain dependencies, stubs for simple responses.
4. **Organize fixtures**: Place static test data in `testdata/`. Generate dynamic data in test helpers.
5. **Run deliberately**: Run the package's tests (`go test ./<pkg>/...`) when a question or acceptance criterion needs them, and exercise the compiled binary through a black-box test when a working-behavior claim is needed.
6. **Review coverage**: Run `go test -coverprofile=cover.out ./<pkg>/...` and `go tool cover -func=cover.out`, and investigate uncovered critical paths. Accept uncovered low-risk paths.
7. **Validate independence**: Confirm each test passes in isolation (`go test -run '^TestName$'`, and `-shuffle=on` for order dependence) without importing siblings of the exercised package.

[Testability as Emergent Property](testability.md)

## Sources

- https://go.dev/doc/go1.24 — `T.Context`, `T.Chdir`, `B.Loop` (Go 1.24, released 2025-02-11; accessed 2026-11-22)
- https://go.dev/doc/go1.25 — `testing/synctest` generally available, `T.Attr`, `T.Output` (Go 1.25, released 2025-08-12; accessed 2026-11-22)
- https://go.dev/doc/go1.26 — `T.ArtifactDir` and `go test -artifacts`, `cryptotest.SetGlobalRandom`, `B.Loop` no longer prevents inlining (Go 1.26, released 2026-02-10; accessed 2026-11-22)
- https://go.dev/doc/go1.27 — `synctest.Sleep`, `httptest.NewTestServer` (Go 1.27, released 2026-08-19; accessed 2026-11-22)
- https://pkg.go.dev/testing/synctest — bubble semantics, durable blocking, restrictions (go1.27.4; accessed 2026-11-22)
- https://go.dev/blog/loopvar-preview — per-iteration loop variables keyed by the `go` directive (Go 1.22; accessed 2026-11-22)

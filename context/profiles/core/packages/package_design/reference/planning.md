# Package Planning

Procedure for designing a Go package's structure, API surface, and internal organization before writing code. Planning upfront prevents costly restructuring by establishing boundaries, consumer contracts, and validation criteria early.

## Prerequisites

Gather before starting:

- **Purpose statement**: One sentence describing what the package does. If "and" connects unrelated concepts, scope is too broad.
- **Consumer inventory**: Which packages import this one and what operations they need.
- **Dependency map**: Standard library and third-party imports.
- **Lifecycle stage**: New package, refactoring, or decomposition of an oversized package.

## Planning Workflow

### 1. Define Purpose and Scope

Write the one-sentence purpose statement. Test it against cohesion criteria: the package serves one conceptual domain.

[Package Decomposition](package_decomposition.md)

Multiple domains in the purpose statement means multiple packages. Apply thresholds (exported symbols <50, LOC <3,000, single domain) as design constraints from the start.

### 2. Identify Consumers

List consumers with required operations. Group by frequency:

| Frequency | Design Implication |
|-----------|--------------------|
| Common (>80%) | Primary API -- core types and constructors |
| Occasional (20-80%) | Configuration options or secondary constructors |
| Rare (<20%) | Sub-packages or expert interfaces |

This drives progressive disclosure: beginners see 1-3 concepts, intermediate users configure, experts access sub-packages.

### 3. Design API Surface

Apply minimal API constraints to the consumer list from step 2.

[Minimal Public API](../../api_design/reference/minimal_api.md)

Plan these outputs:

- **Entry points**: 1-2 constructors. More than 2 signals scope creep.
- **Core types**: Primary struct(s) consumers interact with.
- **Methods**: Only those consumers call directly.
- **Sentinel errors**: Only those consumers check with `errors.Is`.

### 4. Plan Internal Structure

| Content | Placement |
|---------|-----------|
| Core types and constructors | Package root (exported) |
| Implementation helpers | Package root (unexported) |
| Shared internal types across files | `internal/` sub-package |
| Optional features with heavy deps | Separate sub-package |
| Test fixtures and mocks | `internal/testutil/` or `_test.go` |

[Package Decomposition](package_decomposition.md)

Verify each sub-package is independently useful. A sub-package with only one consumer belongs as unexported code in the parent.

### 5. Design Interfaces

[Interface Design Discipline](../../api_design/reference/interface_design.md)

For each interface, determine:

- **Parameters**: Narrowest interface the function needs. One method ideal; 2-5 acceptable.
- **Return types**: Concrete types from constructors. Document abstract factory exceptions.
- **Ownership**: Interfaces this package consumes belong here. Interfaces it implements belong in the consumer's package.

### 6. Plan Configuration

[Options Pattern Discipline](../../api_design/reference/options_pattern.md)

- Public library: functional options
- Internal service or hot path: config struct
- <3 parameters: direct constructor arguments
- 3-8 parameters: functional options
- Signs of >10 options: reassess package scope (return to step 1)

### 7. Design Error Handling

[Error Type Design](../../errors/reference/error_type_design.md)

Map each failure mode:

- Binary condition: sentinel error (`Err` prefix)
- Caller needs context fields: structured error type (`Error` suffix)
- Multiple simultaneous failures: `errors.Join`
- Cross-package boundary: `fmt.Errorf("context: %w", err)`

### 8. Validate the Plan

[API Surface Analysis](../../validation/reference/api_surface.md)
[Package Structure Validation](../../validation/reference/package_structure.md)

Check planned exports, entry points, interface sizes, option counts, and dependency direction against thresholds. Violations caught here cost a fraction of post-implementation fixes.

## Decision Support

### When to Split Packages

**Split** when: purpose requires "and" between unrelated concepts, planned exports exceed 50, consumer groups need disjoint API subsets, optional features drag in heavy dependencies, or test fixtures share no setup code.

**Stay together** when: all types serve one concept, consumers use all exports together, and helpers are not independently reusable.

### Export Decisions

| Question | If Yes | If No |
|----------|--------|-------|
| Does a consumer call this directly? | Export | Unexport |
| Do 3+ external packages need this type? | Export | Keep internal |
| Is this a test helper? | `internal/testutil/` | `_test.go` |
| Implementation detail that may change? | Unexport | Export if stable |

### Interface Placement

- One consumer: define in that consumer's package
- 3+ consumers: define in neutral package (`types`, parent)
- Function parameter: define in the package containing the function

## Planning Artifacts

**API sketch**: Go file with exported signatures and `panic("not implemented")` bodies. Must compile. Serves as the contract.

**Package diagram**: Text diagram of packages, dependency directions, and interface boundaries. Zero circular dependencies.

**Consumer examples**: One usage example per frequency tier (common, occasional, rare). These become `Example*` test functions.

## Validation Checkpoints

**After step 4**: verify structure against decomposition thresholds.

[Package Structure Validation](../../validation/reference/package_structure.md)

**After step 8**: verify full API surface against all principle thresholds.

[API Surface Analysis](../../validation/reference/api_surface.md)

Return to the relevant planning step on any violation. The planning loop is cheaper than the implementation loop.

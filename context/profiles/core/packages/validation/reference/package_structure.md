# Package Structure Validation

Verify that package organization follows decomposition principles: correct dependency direction, appropriate `internal/` usage, zero circular dependencies, and single-domain cohesion per package.

## Validation Checks

Six checks cover package structure. Four are fully automated; two require manual review.

| Check | Method | Tool | Criteria |
|-------|--------|------|----------|
| Circular dependencies | Automated | Go compiler + `go-cyclic` | Zero cycles |
| Dependency direction | Automated | `go-arch-lint` (YAML rules) | Parent imports child only |
| Import rule enforcement | Automated | `depguard` (via `golangci-lint run`) | No violations |
| Coupling metrics (Ca/Ce) | Automated | `spm-go` or `effrit` | Instability = Ce/(Ca+Ce) |
| Conceptual cohesion | Manual | Code review | Single domain per package |
| Sub-package justification | Manual | Code review | Each sub-package independently useful |

## Automated Checks

### Circular Dependency Detection

The Go compiler rejects import cycles at build time. This catches direct cycles but misses indirect chains through intermediate packages. Run `go-cyclic` for granular cycle detection with full import chain output. Any cycle is a hard failure.

**Pass criteria**: Zero cycles across the entire module.

### Dependency Direction Enforcement

`go-arch-lint` enforces layering rules through YAML configuration. Define allowed import relationships between package groups. The tool verifies that dependencies flow in one direction: parent packages import child packages, never the reverse. Sibling packages at the same level should not depend on each other.

```yaml
# .go-arch-lint.yml example
allow:
  - from: "transport/**"
    to: "service/**"
  - from: "service/**"
    to: "domain/**"
deny:
  - from: "domain/**"
    to: "transport/**"
```

**Pass criteria**: All imports conform to declared layering rules.

### Import Rule Enforcement

`depguard` (a golangci-lint linter, run by `golangci-lint run`) enforces package-level dependency rules. Use it to block imports from specific packages, restrict access to `internal/` packages from outside their parent tree, and enforce architectural boundaries.

**Pass criteria**: Zero depguard violations.

### Coupling Metrics

`spm-go` or `effrit` measure afferent coupling (Ca: incoming dependencies) and efferent coupling (Ce: outgoing dependencies) per package. The instability metric I = Ce/(Ca+Ce) ranges from 0 (maximally stable, many dependents) to 1 (maximally unstable, many dependencies).

Stable packages (low I) should contain abstractions and interfaces. Unstable packages (high I) should contain concrete implementations. A package that is both stable and concrete becomes a pain point -- difficult to change but full of implementation detail.

**Warning criteria**: Concrete packages with I < 0.3 (stable but not abstract). Abstract packages with I > 0.7 (unstable despite being depended upon).

## Manual Checks

### Conceptual Cohesion

Review whether each package serves a single domain concept. A package that requires "and" to describe its purpose ("loading AND credential management AND profile resolution") violates cohesion. Look for type clusters with narrow inter-group dependencies. These clusters indicate natural sub-package boundaries.

**Review criteria**:

- Package purpose describable in one phrase without conjunctions
- Types within the package share data or collaborate directly
- Test fixtures are cohesive rather than segmented into independent groups

### Sub-package Justification

Each sub-package should be independently useful to at least some consumers. A sub-package that exists only to satisfy an import restriction, or one that every consumer must import alongside the parent, provides no real separation.

**Review criteria**:

- Sub-package has consumers independent of the parent package
- Sub-package encapsulates a distinct capability (keyring, driver, codec)
- Removing the sub-package would force consumers to import unrelated dependencies

## Internal Package Usage

Validate that non-public implementation details reside under `internal/` directories.

| Metric | Pass | Warn | Fail |
|-------|------|------|------|
| Code in `internal/` (library packages) | >50% | 30-50% | <30% |
| Code in `internal/` (service packages) | >70% | 50-70% | <50% |

Measure by lines of code. Compare total LOC under `internal/` against total package LOC. A custom script using `wc -l` or `go/analysis` provides this metric -- no standard linter exists for it.

**Exception**: Packages designed as pure utility collections (like `strings` or `strconv`) may have little internal code because most functions are independently useful to consumers.

## Tools

| Tool | Purpose | Integration |
|------|---------|-------------|
| `go-cyclic` | Granular cycle detection with import chain output | Standalone CLI |
| `go-arch-lint` | Layering rule enforcement via YAML configuration | Standalone CLI, CI |
| `depguard` | Package dependency rules | `golangci-lint run` (built-in linter) |
| `spm-go` | Afferent/efferent coupling measurement | Standalone CLI |
| `effrit` | Coupling analysis alternative | Standalone CLI |
| `gopkgview` | Web-based interactive dependency visualization | Development aid |

## Remediation

**Circular dependency**: Extract shared types into a neutral third package (`domain`, `types`, or a parent package). Both packages then import the neutral package instead of each other.

**Wrong dependency direction**: Invert the dependency using consumer-defined interfaces. The lower-level package defines no interface; the higher-level package defines the interface it needs and the lower-level package's concrete type implicitly satisfies it.

**Low cohesion**: Split along conceptual boundaries identified through type cluster analysis. Group types that share data and collaborate into one package; move types serving a different concept into a new sub-package.

**Missing internal usage**: Move non-public implementation details behind `internal/`. Start with helpers used only within the package, then move types that no external consumer imports. This is backward-compatible for external consumers.

**Unjustified sub-packages**: Merge sub-packages back into the parent when they have no independent consumers. Unnecessary decomposition adds import complexity without reducing cognitive load.

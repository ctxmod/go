# Documentation Quality Validation

Verify that Go package documentation meets completeness, accuracy, and discoverability standards. Documentation quality directly affects time-to-first-success: consumers encounter `go doc` output and pkg.go.dev before reading source code, making documentation the primary API surface for adoption decisions.

## What to Validate

This procedure covers four areas: symbol documentation presence, package-level documentation quality, testable example coverage, and documentation structure for progressive disclosure.

## Validation Checks

### Automated Checks

| Check | Tool | Pass | Warn | Fail |
|-------|------|------|------|------|
| Exported symbol comments | `revive` (`exported` rule) | 100% coverage | -- | <100% coverage |
| Package-level comment | `revive` (`package-comments`) | Present with purpose statement | -- | Missing |
| Comment grammar (trailing period) | `godot` | No violations | -- | Violations present |
| Spelling | `misspell` | No errors | -- | Errors present |
| Example function validity | `go test -run Example ./...` | All examples compile and pass | -- | Any example fails |
| Example function signatures | `go vet ./...` (the `tests` analyzer) | Valid signatures | -- | Malformed examples |

### Semi-Automated Checks

| Check | Method | Criteria |
|-------|--------|----------|
| Testable example coverage | `go test -run Example ./...` + manual audit | `Example*` functions exist for primary types and constructors |

Run `go doc <package>` and verify that major types show associated examples. `go test` executes every `Example*` function that has an output comment, validating that output comments match actual output. This guarantees documentation cannot drift from implementation.

### Manual Review Checks

| Check | Review Criteria |
|-------|----------------|
| Progressive disclosure | Package docs present a clear beginner path before advanced features |
| Error semantics documentation | Package docs list sentinel errors and typed errors with handling guidance |

## Tool Commands

**Run all automated checks**:
```sh
golangci-lint run ./...
```

**Validate testable examples compile and produce correct output**:
```sh
go test -run Example ./...
```

**Inspect package documentation as consumers see it**:
```sh
go doc <package>
go doc <package> <Symbol>
```

**Generate readable test descriptions** (optional):
```sh
gotestdox ./...
```

## Symbol Documentation Standards

Every exported symbol requires a comment starting with the symbol name, following godoc conventions:

```go
// Resolver parses and resolves template tags within markdown files.
type Resolver struct { ... }

// NewResolver creates a Resolver with the given options.
func NewResolver(opts ...Option) *Resolver { ... }

// ErrNotFound indicates the referenced file does not exist.
var ErrNotFound = errors.New("not found")
```

Package comments state purpose in the first sentence, then describe usage patterns. The first sentence appears as the package synopsis on pkg.go.dev and in `go doc` listings.

## Testable Example Requirements

Testable examples serve three purposes simultaneously: live documentation (rendered in godoc), regression testing (executed by `go test`), and accuracy validation (build fails when output diverges from implementation).

**Coverage targets**:
- Package-level example (`Example`) showing the canonical usage path
- Examples for each primary constructor (`ExampleNewResolver`)
- Examples for each primary type method when behavior is non-obvious

**Example structure**:
```go
func ExampleNewResolver() {
    r := resolve.NewResolver(resolve.WithBaseDir("/docs"))
    result, _ := r.Resolve(input)
    fmt.Println(result.Name)
    // Output:
    // context-efficiency
}
```

The `// Output:` comment is required for `go test` to validate the example. Examples without output comments serve as documentation but lack regression protection.

## Progressive Disclosure in Documentation

Package documentation should guide consumers from simple to advanced usage. Review for these structural elements:

1. **Opening synopsis**: One sentence stating what the package does (appears in `go doc` listings).
2. **Getting started section**: A complete code snippet using the canonical entry point with defaults. Consumers should copy-paste this and get working results.
3. **Configuration section**: Options, customization, and intermediate usage patterns.
4. **Advanced section**: Sub-package usage, extension interfaces, and expert-level control.

Packages with 50+ exported symbols require stronger progressive disclosure because consumers cannot scan the full API quickly.

## Error Documentation

Package documentation should include an error handling section when the package exports sentinel errors or structured error types. Document:

- Which functions return which error types
- How to check errors with `errors.Is` and `errors.As`
- Whether errors are retryable, permanent, or transient

## Remediation

**Missing symbol comments**: Add a comment starting with the symbol name. Focus on what the symbol represents or does, not implementation details.

**Missing package comment**: Add a `// Package <name> ...` comment in `doc.go` or the file containing the primary type. State purpose first, then usage patterns.

**Missing examples**: Create `Example*` functions in `*_test.go` files for primary types and constructors. Include `// Output:` comments to enable regression testing. Start with the package-level example showing the canonical path.

**Poor progressive disclosure**: Restructure the package comment to lead with the simplest usage pattern. Move advanced documentation to dedicated sections or sub-package documentation. Add a "Getting Started" code block as the first example.

**Undocumented error semantics**: Add an "Errors" section to the package comment listing sentinel errors and structured error types. Include short code snippets showing `errors.Is` and `errors.As` usage.

## Pass/Fail Summary

| Category | Pass Condition |
|----------|----------------|
| Automated checks | Zero violations from `golangci-lint run` with revive, godot, and misspell enabled |
| Example coverage | Primary types and constructors have `Example*` functions with output validation |
| Progressive disclosure | Package docs show a clear path from basic to advanced usage |
| Error documentation | Exported errors and error types documented with handling patterns |

A package passes documentation quality validation when all automated checks report zero violations, primary API elements have testable examples, and manual review confirms progressive disclosure and error documentation.

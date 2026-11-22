# Error Type Design

Use sentinel errors for simple binary signals. Use structured error types when callers need programmatic access to failure details. Support error chain inspection via `errors.Is` and `errors.As`.

## Error Categories

Go provides three mechanisms for communicating failure information. Each serves a distinct purpose, and selecting the wrong one forces callers into string parsing or type guessing.

**Sentinel errors** are package-level variables created with `errors.New`. They signal binary conditions (found/not found, open/closed) without carrying dynamic context. The standard library names these with an `Err` prefix: `io.EOF`, `sql.ErrNoRows`, `context.Canceled`.

**Structured error types** are exported structs implementing the `error` interface. They carry contextual fields (identifiers, codes, retry delays) that callers extract programmatically. The standard library names these with an `Error` suffix: `os.PathError`, `json.SyntaxError`, `net.DNSError`.

**Opaque error interfaces** define behavioral contracts without coupling callers to specific types. A consumer defines a local interface (e.g., `Temporary() bool`) and uses `errors.As` to check whether any error in the chain satisfies that behavior. This pattern decouples retry logic from error implementation.

## Decision Matrix

| Scenario | Mechanism | Example |
|----------|-----------|---------|
| Simple binary signal | Sentinel error (`var ErrNotFound = errors.New(...)`) | `io.EOF` |
| Caller needs context fields | Structured error type (`type QueryError struct{...}`) | `os.PathError` |
| Multiple errors from validation or cleanup | `errors.Join(err1, err2)` (Go 1.20+) | Accumulated option validation |
| Cross-library behavior checking | Opaque interface + `errors.As` | `Temporary() bool` |
| Library public API boundary | Sentinel errors + opaque interfaces | Minimize type coupling |

## Chain Inspection

Go 1.13 introduced `errors.Is` and `errors.As` for traversing error chains created by `fmt.Errorf("context: %w", err)`. Go 1.20 extended both to support error trees from `errors.Join`.

**`errors.Is(err, target)`** performs value equality across the chain. Use it to check sentinels:

```go
if errors.Is(err, ErrNotFound) {
    // Handle missing resource
}
```

**`errors.As(err, &target)`** performs type matching and extraction. Use it to read fields from structured errors:

```go
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    log.Printf("operation %s failed on %s", pathErr.Op, pathErr.Path)
}
```

Both functions unwrap recursively. Callers never need manual unwrap loops or type switch chains.

## Structured Error Pattern

Export error types when callers need programmatic access to failure details. Each type carries fields relevant to its failure mode:

```go
// QueryError provides programmatic access to query failure details.
type QueryError struct {
    Query   string
    Code    int
    Message string
}

func (e *QueryError) Error() string {
    return fmt.Sprintf("query %q failed (code %d): %s", e.Query, e.Code, e.Message)
}

// Wrap the original cause for chain inspection.
func (e *QueryError) Unwrap() error { return e.Cause }
```

Implement `Unwrap() error` when the structured error wraps an underlying cause. This preserves the chain so callers can inspect both the typed error and its root cause with a single `errors.As` call.

## Opaque Error Handling

For advanced decoupling across library boundaries, assert behavior rather than type:

```go
// Consumer defines the contract locally.
type temporary interface {
    Temporary() bool
}

var t temporary
if errors.As(err, &t) && t.Temporary() {
    // Retry without depending on any specific error type
}
```

This follows the interface segregation principle: consumers depend only on the behavior they need.

## Trade-offs

Sentinel errors are simpler but carry no context. Structured types provide context but increase API surface. The decision follows consumer needs: if callers only branch on error presence, use sentinels; if callers need details for logging, retries, or user-facing messages, use structured types.

Opaque interfaces add indirection but eliminate cross-package type coupling. Reserve them for library public APIs where stability across versions matters more than direct field access.

## When to Deviate

Packages with few failure modes may use only sentinels. Utility packages performing pure computation (string formatting, math operations) often need no exported error types at all. Domain packages with rich failure semantics (network operations, file systems, query engines) benefit from structured types and behavioral interfaces.

Error type hierarchies (embedding one error type in another) add complexity. Prefer flat, independent error types unless a clear is-a relationship exists between failure modes.

## Error Wrapping (rule context)

Extracted verbatim from the `go-error-wrapping-context` rule so that rule can stay a pointer.

### Rationale

Cross-package boundaries lose caller context unless errors are wrapped at each boundary. Unwrapped errors propagate without indicating where they originated, forcing callers to guess failure location. The `%w` verb preserves the original error in the chain so `errors.Is` and `errors.As` continue to match wrapped values.

### Examples

**Correct**:
```go
func (r *Resolver) Resolve(path string) (*Entry, error) {
    entry, err := r.index.Get(path)
    if err != nil {
        return nil, fmt.Errorf("resolve %s: %w", path, err)
    }
    return entry, nil
}
```

**Incorrect**:
```go
func (r *Resolver) Resolve(path string) (*Entry, error) {
    entry, err := r.index.Get(path)
    if err != nil {
        return nil, err // Lost context, breaks chain
    }
    return entry, nil
}
```

### Exceptions

Errors returned to external callers (CLI main functions, HTTP handlers) where wrapping adds no actionable context. At these terminal boundaries, logging or formatting the error for display replaces wrapping.

### Tooling

The `wrapcheck` linter enforces cross-package wrapping automatically.

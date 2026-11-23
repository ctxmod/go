---
paths:
  - "*.go"
---

# Error Wrapping Context

Wrap all cross-package errors with context using `fmt.Errorf("context: %w", err)`. The `%w` verb preserves the original error in the chain so `errors.Is` and `errors.As` continue to match wrapped values; returning a bare `err` loses caller context and breaks the chain.

Terminal boundaries are exempt: errors returned to external callers (CLI main functions, HTTP handlers) where wrapping adds no actionable context are logged or formatted for display instead of wrapped. The `wrapcheck` linter enforces cross-package wrapping automatically.

- sentinel vs structured vs opaque error selection, chain inspection with `errors.Is`/`errors.As`/`errors.AsType` (Go 1.26+), `errors.Join`, and the worked correct/incorrect wrapping examples — [Error Type Design](../reference/error_type_design.md)
- not-found checks on wrapped errors: `os.IsNotExist` does not traverse `%w` chains — use `errors.Is(err, os.ErrNotExist)` — [Not-found checks on wrapped errors](../../pitfalls/reference/errors_is_not_exist.md)

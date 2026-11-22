# Service Error Semantics

Errors crossing a service boundary are translated to structured RPC status
codes — once, at the boundary, by the shared mapping — so clients can
classify retryability instead of guessing from wrapped strings. Raw Go
errors, their stack-bound context, and internal messages never cross the
wire. Retryable codes (transient unavailability, conflicts, quota) are the
only ones retry policies act on, and only for idempotent or
idempotency-keyed operations.

- the full code table with client postures, machine-readable error details, and the retryability classification — [Error Semantics](../reference/error_semantics.md)

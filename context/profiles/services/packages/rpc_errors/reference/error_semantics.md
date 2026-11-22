# Error Semantics

Return structured RPC status codes with machine-readable error details instead of passing raw errors through, and keep the retryability classification clients depend on.

Passing raw Go errors across a service boundary leaves clients unable to
answer the one question every caller must ask: *may I retry this?* A wrapped
string carries no classification, leaks implementation detail, and makes
clients guess. Service boundaries exchange structured status codes.

## Structured Status Codes at the Boundary

Internal errors are translated to canonical RPC status codes once, at the
server boundary — by the status-mapping interceptor, not ad hoc in handlers.
The mapping is a small, reviewed table:

| Internal condition | Code | Client posture |
|---|---|---|
| Malformed input, validation failure | Invalid argument | Terminal — fix the request |
| Unknown referenced entity | Not found | Terminal |
| Authenticated, not allowed | Permission denied | Terminal |
| State conflict with concurrent work | Aborted | Retryable (with backoff) |
| Transient dependency/network failure | Unavailable | Retryable (with backoff) |
| Quota/rate exhausted | Resource exhausted | Retryable (honoring the stated delay) |
| Timeout budget spent | Deadline exceeded | Usually terminal within this request |
| Unmapped catch-all | Internal | Terminal — a service defect |

Two unmapped-error rules keep the table honest: anything that reaches the
boundary unmapped becomes `Internal` (loud in telemetry, never silently
recast as a client's fault), and internal error strings never cross the wire —
the code and its details are the contract.

## Machine-Readable Error Details

Rich, typed error details accompany the code where callers need more than a
classification: which field failed validation, what the retry delay is, which
quota was hit. Structured detail payloads (the standard error-details proto
family, or a schema'd equivalent) let clients branch programmatically — a
client can honor `retry_info.delay` or highlight `bad_request.field` without
parsing prose. Details are schema'd and versioned like everything else on the
wire; free-form strings in a details slot are no better than the raw error
they replaced.

## Retryability Is the Client's Branch Point

The classification above is what the [resilience
policies](../../resilience/reference/resilience.md) consume: retry with backoff on `Unavailable`,
`Aborted`, and `Resource exhausted` (idempotent operations or idempotency-keyed
writes only); never retry terminal codes; treat `Deadline exceeded` as spent
budget for this request. When a service leaks raw errors instead of codes,
clients either retry everything (amplifying outages) or nothing (converting
transient blips into user-visible failures) — the ambiguity itself is the
defect.

Telemetry gets the same benefit: aggregated status codes are a service health
signal; aggregated error strings are noise.

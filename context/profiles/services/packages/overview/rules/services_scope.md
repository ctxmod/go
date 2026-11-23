---
paths:
  - "*.go"
  - "*/*.go"
  - "*/*/*.go"
  - "*/*/*/*.go"
  - "*/*/*/*/*.go"
  - "*/*/*/*/*/*.go"
  - "*/*/*/*/*/*/*.go"
  - "*/*/*/*/*/*/*/*.go"
  - "*/*/*/*/*/*/*/*/*.go"
  - "*/*/*/*/*/*/*/*/*/*.go"
---

# Go Services Scope

Scope: long-lived networked Go services — RPC and HTTP servers behind load
balancers, clients of other services, and the shared in-process libraries
those services compose. Single-binary CLI tools are the `cli` profile's
concern, not this one.

The profile adds service-specific guidance on top of the go core
references; it never restates them. The go core references it inherits
unchanged ([error type design](../../../../core/packages/errors/reference/error_type_design.md),
[interface design](../../../../core/packages/api_design/reference/interface_design.md),
[package decomposition](../../../../core/packages/package_design/reference/package_decomposition.md),
[testability](../../../../core/packages/testing/reference/testability.md)) apply to service code
exactly as to any other Go code.

## Reference surface

- [Resilience](../../resilience/reference/resilience.md) — context deadlines at the edge, bounded concurrency, circuit breaking, retries, client-side throttling, L7 load balancing.
- [Interceptors and tracing](../../telemetry/reference/interceptors_and_tracing.md) — uniform client/server interceptor layers, W3C TraceContext and baggage propagation, OTLP wiring.
- [Error semantics](../../rpc_errors/reference/error_semantics.md) — structured status codes with machine-readable details, retryability classification.
- [Contract governance](../../contracts/reference/contract_governance.md) — schema-first boundaries, compatibility gates, reserved tags, presence semantics, generated-code discipline.

# Interceptors and Tracing

Uniform client and server interceptor layers carry logging, distributed tracing, and status mapping; W3C TraceContext and baggage propagate into RPC metadata and queue attributes; OTLP wiring stays generic.

When one user request cascades through many services, latency spikes, silent
errors, and dropped context are unexplainable without structured telemetry —
and per-service ad-hoc instrumentation guarantees gaps. The interceptor layer
is where logging, tracing, recovery, and error mapping are configured once, in
a shared module, and inherited by every service.

## Uniform Interceptor Layers

Client and server interceptor chains are assembled in one shared library and
carry a fixed, ordered concern set:

| Concern | Side | Responsibility |
|---|---|---|
| Recovery | server | Convert panics into structured internal errors, not connection resets |
| Logging | both | Request/response logging with identifiers, duration, outcome |
| Tracing | both | Extract (server) / inject (client) trace context |
| Status mapping | server | Translate internal error types into canonical RPC status codes once, at the boundary |
| Retry and backoff | client | The resilience policies, attached at one choke point |

Services compose the shared chain; they do not re-implement concerns. A
concern added per-service drifts — the fifth service logs a different shape
than the first, and cross-service correlation tools cannot consume the
result.

## Distributed Tracing Propagation

- **W3C TraceContext** (`traceparent`, `tracestate`) is the propagation
  format. The server-side interceptor extracts context from inbound RPC
  metadata (or HTTP headers at an HTTP edge); the client-side interceptor
  injects it into outgoing metadata.
- **Baggage** carries cross-cutting request attributes (tenant, feature flag,
  originating region) alongside the trace. Inject baggage into RPC metadata
  deliberately — it propagates on every hop, so only bounded, low-cardinality
  values ride in it.
- **Asynchronous hops must carry context too.** A message published to a queue
  injects trace context into message attributes at produce time; the consumer
  extracts it and links its spans to the producer's trace. A queue that drops
  the context breaks the trace exactly where debugging is hardest — the
  HTTP-to-RPC-to-queue transitions are the gaps that motivated the telemetry
  in the first place.

## OTLP Wiring

- One shared setup module configures the tracer and meter providers, the OTLP
  exporters, and the resource attributes (service name, version, environment)
  derived from the build, not hand-maintained per service.
- Sampling is decided at the edge; downstream services honor the parent's
  sampling decision so a trace is complete or absent, never fragmented.
- Span names are stable, low-cardinality identifiers (route or method shape),
  never raw URLs or parameterized values — high-cardinality names destroy
  aggregation.

## Boundary with Domain-Specific Span Semantics

This reference covers generic transport mechanics: context propagation,
interceptor wiring, exporter configuration. Domain-specific span attribute
vocabularies — AI model-call attributes, retrieval attributes, and similar
semantic-convention layers — are a separate discipline owned by the AI
guidance when a repository carries it; they are layered on top of this
wiring, never inlined into service interceptors. Keeping the two apart lets
the transport wiring stay stable while semantic conventions evolve.

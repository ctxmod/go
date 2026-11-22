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

## Go Wiring Notes

- **OpenTelemetry Go signal status**: traces and metrics are stable; logs are
  a release candidate on the project's status page (the `sdk/log` module is
  tagged v1). Version-pin the API, SDK, and contrib instrumentation modules
  together — contrib releases track a specific core release.
- **gRPC tracing rides stats handlers, not interceptors.** The `otelgrpc`
  contrib package instruments through `grpc.StatsHandler(otelgrpc.NewServerHandler())`
  on servers and `grpc.WithStatsHandler(otelgrpc.NewClientHandler())` on
  clients; its interceptor filter API is deprecated in favor of stats
  handlers. Logging, recovery, and status mapping stay in the shared
  interceptor chain; tracing does not need a slot in it.
- **`grpc.NewClient`, not `grpc.Dial`.** `Dial` and `DialContext` are
  deprecated (supported through 1.x). `NewClient` creates the channel
  without connecting and defaults to the `dns` resolver, where `Dial` used
  `passthrough` — which is what makes client-side per-RPC balancing across
  resolved backends (see the L7 load-balancing section of the resilience
  reference) work without a custom resolver.
- **Logs correlate through the context.** Bridge `log/slog` into the OTel
  logs pipeline with the contrib `otelslog` handler (combine it with a local
  handler through `slog.NewMultiHandler`, Go 1.26+). The log SDK reads the
  span context from the `context.Context` passed to `Emit`, so log through
  slog's `InfoContext`/`ErrorContext` family with the request context — a
  plain `Info` call drops the trace and span IDs.

## Boundary with Domain-Specific Span Semantics

This reference covers generic transport mechanics: context propagation,
interceptor wiring, exporter configuration. Domain-specific span attribute
vocabularies — AI model-call attributes, retrieval attributes, and similar
semantic-convention layers — are a separate discipline owned by the AI
guidance when a repository carries it; they are layered on top of this
wiring, never inlined into service interceptors. Keeping the two apart lets
the transport wiring stay stable while semantic conventions evolve.

## Sources

- https://opentelemetry.io/docs/languages/go/ — OpenTelemetry Go signal status: traces and metrics stable, logs release candidate (accessed 2026-11-22)
- https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc — `NewServerHandler`/`NewClientHandler` stats handlers, deprecated interceptor filters (v0.72.0, published 2026-10-02; accessed 2026-11-22)
- https://pkg.go.dev/google.golang.org/grpc#NewClient — `Dial`/`DialContext` deprecated in favor of `NewClient`, `dns` versus `passthrough` default resolver (v1.85.0; accessed 2026-11-22)
- https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog — the slog-to-OTel-logs bridge (v0.21.0, published 2026-10-02; accessed 2026-11-22)
- https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/log/logger.go — `Emit` takes trace and span IDs from `trace.SpanContextFromContext(ctx)` (source read 2026-11-22)
- https://www.w3.org/TR/trace-context/ — the W3C TraceContext propagation format (accessed 2026-11-22)

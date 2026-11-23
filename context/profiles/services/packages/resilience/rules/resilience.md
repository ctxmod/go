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

# Go Service Resilience

Concurrency in service code is bounded: work enters through fixed-size worker
pools, every goroutine has an owner and an exit path, and saturation sheds
load instead of queueing without limit. `context.Context` propagates through
every layer with deadlines set at the edge and cancellation honored
everywhere. Calls to other services carry circuit breakers, exponential
backoff with jitter, and per-dependency in-flight caps; long-lived multiplexed
connections are balanced per-request at L7, never spread once at L4.

- the full resilience discipline — context propagation and edge deadlines, bounded pools versus unbounded spawns, breakers and retry budgets, client-side throttling, and L7 load balancing — [Resilience](../reference/resilience.md)
- uniform interceptor layers and trace-context propagation — [Interceptors and Tracing](../../telemetry/reference/interceptors_and_tracing.md)

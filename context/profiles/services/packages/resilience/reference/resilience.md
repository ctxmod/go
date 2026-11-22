# Resilience

Resilience discipline for Go services: edge deadlines, bounded concurrency, circuit breaking, retries with jitter, client-side throttling, and L7 load balancing for multiplexed connections.

A service behind other services fails in patterns, not at random: unbounded
goroutine spawns exhaust memory, a degraded downstream turns its callers into
holders of exhausted connection pools, and one deep dependency graph turns a
single slow hop into a cascading outage. The mitigations are structural, not
per-call heroics — they belong in shared libraries every service composes.

## Strict Context Propagation

`context.Context` flows through every layer of every call chain — first
parameter, never stored in structs, never nil. Deadlines are set once, at the
edge (the request ingress), with `context.WithTimeout`/`WithDeadline`, and
every downstream hop derives its child context from the parent so cancellation
and remaining budget propagate for free. A handler that starts work without a
deadline borrows the client's patience indefinitely; a goroutine that ignores
`ctx.Done()` outlives the request that spawned it.

Edge-owned budget means each hop may further bound its portion
(`context.WithTimeout(ctx, remaining/2)` for a two-hop budget), but never
extends it. When the edge cancels, everything under the tree stops — that is
the contract.

## Bounded Concurrency and Goroutine Discipline

Go makes spawning trivial; production requires that every goroutine has an
owner and an exit path:

- **Worker pools over unbounded spawns.** Work entering a service is processed
  by a pool of fixed size (bounded channel semaphore, `errgroup` with
  `SetLimit`, or an explicit worker pool) sized to the resource the work
  consumes — connections, file descriptors, CPU. An unbounded `go func()` per
  request converts a traffic spike into memory exhaustion with no valve.
- **Every spawn is waited on or cancels.** A goroutine blocked forever on a
  channel nobody closes is a leak; the owner either collects the result or
  guarantees termination via context cancellation. Leak detectors in tests
  catch the residue; discipline prevents it.
- **Backpressure is explicit.** When the pool is saturated, the service sheds
  (bounded queue with a maximum, reject at ingress) rather than queues without
  limit. Queuing is a memory decision, not a default.

## Circuit Breaking and Retries

- **Circuit breakers** (closed / open / half-open) wrap calls to dependencies
  that can degrade. Sustained failure rates trip the breaker; while open, calls
  fail fast without touching the network, letting the dependency recover and
  freeing the caller's resources. A caller that retries through an open
  breaker is part of the outage.
- **Retries use exponential backoff with jitter**, and only on
  [retryable classifications](../../rpc_errors/reference/error_semantics.md) — a tight retry loop on a
  degraded dependency multiplies load exactly when the dependency can least
  afford it. Cap retries (attempt count and total time budget derived from the
  parent deadline), add jitter so retry storms do not synchronize, and never
  retry when the parent deadline is nearly spent.
- **Retry only idempotent operations**, or attach idempotency keys so a
  replayed write collapses into the first attempt. Non-idempotent calls retry
  at the application layer, where the dedup key lives.

## Client-Side Throttling

Each client of a dependency caps its in-flight calls (a concurrency limiter or
bounded connection pool per dependency). When the cap is hit, the caller fails
fast or queues briefly with a bound — it does not keep feeding a struggling
dependency. Throttling pairs with the breaker: the breaker decides *whether*
to call; the limiter decides *how much at once*.

## L7 Load Balancing

HTTP/2 multiplexes many RPCs over few long-lived TCP connections. An L4
(round-robin TCP) balancer spreads connections, not requests — after the first
connection lands, that one backend receives the connection's entire RPC
stream while its siblings idle. Balance at L7 instead: per-request/per-RPC
routing through an L7 proxy, or client-side balancing that resolves backends
and picks per call. Any autoscaling or rollout story that assumes L4 even
spreading is broken by multiplexing before the first deploy.

## Failure Postures

Degrade rather than block: a service that cannot reach a non-critical
dependency serves its degraded path (cached value, partial response, feature
off) with the degradation observable; it does not hang its callers waiting for
a dependency that may not return. Propagate deadline exhaustion upward as the
structured error it is — callers classify and adapt (see
[error semantics](../../rpc_errors/reference/error_semantics.md)); stringly passthrough gives them
nothing to branch on.

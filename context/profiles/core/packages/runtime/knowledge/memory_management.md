# Memory Management

Go provides runtime-level knobs for controlling memory. Use them
instead of OS-level virtual-memory limits, which are incompatible with
Go's runtime.

## GOMEMLIMIT (Go 1.19+)

`GOMEMLIMIT` sets a soft memory ceiling. The garbage collector becomes
more aggressive as live heap approaches the limit, trading CPU for
memory. Unlike `GOGC` (which is relative), `GOMEMLIMIT` is an absolute
byte count.

```bash
# Force aggressive GC when heap approaches 1 GiB
GOMEMLIMIT=1GiB ./myapp
```

The limit is soft: the runtime may exceed it briefly during
allocation bursts. Set it 5-10% below the hard container limit to
leave headroom for non-heap memory (goroutine stacks, GC metadata).

## GOGC

`GOGC` controls the GC trigger as a percentage of live heap since the
last collection. The default `100` triggers a GC when the heap doubles.
Lower values collect more frequently (more CPU, less memory).

```bash
# Trigger GC when heap grows by 50% (more frequent, lower peak memory)
GOGC=50 ./myapp

# Disable GC entirely (only for short-lived batch tools)
GOGC=off ./myapp
```

### Combined tuning for constrained environments

```bash
GOMEMLIMIT=1GiB GOGC=50 ./myapp
```

`GOGC=50` makes the GC trigger earlier in each cycle, while
`GOMEMLIMIT=1GiB` makes it collect harder as total Go memory nears 1 GiB.
The two compose — whichever triggers first wins — but neither is a hard
ceiling: the memory limit stays soft, and only the container's cgroup
limit (or the kernel OOM killer) enforces a hard one.

## CRITICAL: ulimit -v is incompatible with Go

Go's runtime pre-reserves a large virtual address space for heap
arenas on 64-bit systems (up to 128 TiB). This is reservation, not
committed memory -- the OS only backs pages that are actually written.
But `ulimit -v` caps **virtual** memory, not resident set size (RSS).

```bash
# This KILLS the Go process immediately:
ulimit -v 2097152  # 2 GiB virtual limit
./myapp            # fatal: runtime: out of memory (arena allocation)
```

The runtime cannot allocate its arena address space and fails before
`main()` runs. Use `GOMEMLIMIT` instead -- it is a Go-native soft limit
that works with the GC, not against it.

For container environments, set a cgroup memory limit (Docker
`--memory`, Kubernetes `resources.limits.memory`) and pair it with
`GOMEMLIMIT` at ~90% of the container limit. The runtime reads the cgroup
CPU limit on its own (next section) but documents no memory-limit
equivalent through Go 1.27, so `GOMEMLIMIT` stays an explicit setting.

## GOMAXPROCS is container-aware (Go 1.25+)

Since Go 1.25 the default `GOMAXPROCS` on Linux is the lower of the
logical CPU count and the cgroup CPU *bandwidth limit* (CPU requests are
ignored), and on every OS the runtime re-reads it periodically as the CPU
count or the limit changes. This replaces the old practice of importing
`go.uber.org/automaxprocs` for containerized services: on a `go 1.25`+
module the default is already right, so drop the dependency rather than
stacking two mechanisms.

- Setting `GOMAXPROCS` explicitly (environment or `runtime.GOMAXPROCS`)
  disables both behaviors; `runtime.SetDefaultGOMAXPROCS()` restores the
  runtime default after an override.
- `GODEBUG=containermaxprocs=0` and `updatemaxprocs=0` opt out of the
  cgroup read and the periodic update respectively. The behavior follows
  the main module's `go` directive, so an older directive keeps the old
  default.

## Garbage collector generation (Go 1.26+)

Go 1.26 made the Green Tea collector the default (an experiment in 1.25).
The release notes expect 10–40% less GC overhead in GC-heavy programs.
The `GOEXPERIMENT=nogreenteagc` opt-out was announced for removal in Go
1.27, so treat it as a bisection aid, never a deployment setting. Re-measure
`GOGC`/`GOMEMLIMIT` tuning carried over from older toolchains instead of
assuming it still holds.

## Goroutine leaks are profileable (Go 1.27+)

The `goroutineleak` profile (experimental in 1.26, generally available in
1.27) reports goroutines blocked on channels, mutexes, or condition
variables that the GC can prove will never wake; it is served at
`/debug/pprof/goroutineleak` and through `runtime/pprof`. It uses GC
reachability, so it misses leaks whose primitive is reachable from a global
or from a runnable goroutine's locals — a clean profile is evidence, not
proof, of no leaks.

## Periodic runtime.GC() for batch processing

When processing many files (encoding, hashing, buffering), per-file
allocations accumulate as live heap until the next GC cycle. In
long-running batch loops with large per-iteration allocations, call
`runtime.GC()` periodically to reclaim encoder/buffer state promptly:

```go
import "runtime"

for i, file := range files {
    processFile(file)
    if i%100 == 0 {
        runtime.GC() // reclaim accumulated encoder/buffer state
    }
}
```

This is a hint, not a command -- the runtime may collect sooner or
later regardless. Reserve it for loops where per-iteration heap growth
is large (multi-MiB buffers) and the total file count is high (thousands).
For most code, the default GC triggers are sufficient.

## Sources

- https://go.dev/doc/go1.25 — container-aware and periodically updated `GOMAXPROCS`, `SetDefaultGOMAXPROCS`, `containermaxprocs`/`updatemaxprocs` (Go 1.25, released 2025-08-12; accessed 2026-11-22)
- https://go.dev/doc/go1.26 — Green Tea GC on by default, `nogreenteagc` opt-out slated for removal in 1.27 (Go 1.26, released 2026-02-10; accessed 2026-11-22)
- https://go.dev/doc/go1.27 — `goroutineleak` profile generally available, its reachability limits (Go 1.27, released 2026-08-19; accessed 2026-11-22)
- https://go.dev/doc/godebug — `containermaxprocs`/`updatemaxprocs` and how GODEBUG defaults follow the main module's `go` line (accessed 2026-11-22)
- https://go.dev/doc/gc-guide — `GOGC` and the soft memory limit semantics (accessed 2026-11-22)

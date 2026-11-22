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
`GOMEMLIMIT=1GiB` enforces a hard ceiling. The two compose: whichever
triggers first wins.

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
`GOMEMLIMIT` at ~90% of the container limit.

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

# Go test pitfalls — `t.TempDir` paths vs the unix-socket `sun_path` budget

`t.TempDir()` paths can exceed the kernel's unix-socket address budget,
and the bind/dial then fails in a place far from the real cause. Keep
any test socket path — or any path spelling a client will dial — in a
short `os.MkdirTemp` root, and guard the byte count when dialability is
load-bearing.

## The budget and how tests exceed it

The budget is 108 bytes of `sun_path` on Linux, 104 on macOS.
`t.TempDir()` roots are long: the go-test temp root is
`$GOTMPDIR` (or `$TMPDIR`) followed by `Test<Name><10-digit-rand>`, and
with a non-default temp directory such as a cache under a mounted
volume (for example `/mnt/<volume>/.cache/go-tmp/`) that is about 86
bytes for a 50-character test name — before the per-call `001`/`002`/…
nested dirs are appended. In one observed remediation fixture, an alias
socket spelling published at `<root>/004/alias/<daemon>.sock` came to
110 bytes, and every `connect(2)` failed with `EINVAL`.

## The misleading failure shape

The failure surfaced four hops from its cause. With the client
transport's retries set to 0, the transport reported "daemon gone",
which routed the client's ensure-the-daemon-is-answering step into its
spawn path, failing the test with a misleading "daemon binary
unresolvable" spawn error — not a dial error. The mechanism making long
spellings fatal rather than merely lossy: the client adopted the socket
path the daemon published verbatim (no length guard, no hash-slot
fallback), so any fixture that publishes a long spelling is undialable,
and the client's spawn-on-gone reflex converts the undialable socket
into a spawn failure.

## The rule

- For any test fixture that creates a unix socket — or any path
  spelling a client will dial — use
  `os.MkdirTemp("", "short-prefix-")` (a stub-server regime with
  ~50-byte roots), not `t.TempDir()`.
- Assert `len(path) < sunPathLimit()` when the path's dialability is
  load-bearing; a fail-loud length guard catches this with the real
  byte count printed.
- Diagnosis first, not last: when a daemon-client stub test fails with
  a spawn / "binary unresolvable" error, count the bytes of every path
  the client adopted before attributing anything to binary resolution.
  In the observed case the unelided offending path came from the test
  gate's structured failure details — un-elided failure detail is what
  makes this diagnosable at all.

A daemon that publishes its socket path should treat the same budget
on its own side: lay fixtures out so the published path stays short,
watch for path canonicalization (for example resolving a mount alias
to a longer canonical prefix) re-inflating a short root, and refuse an
over-budget path with a typed error instead of publishing an
undialable one. Scratch-environment fencing and per-process tool
resolution interact with temp-root choice too: a fenced temp
environment that relocates `TMPDIR` changes every `t.TempDir()` length
at once.

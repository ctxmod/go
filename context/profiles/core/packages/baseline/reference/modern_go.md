# Modern Go Baseline

The language, standard-library, and tooling baseline Go code is written against, current through Go 1.27 (2026-11). Each feature is tagged with the release that introduced it; a module may use it only when its `go` directive is at least that version.

## Release Cadence and Support

Go ships a major release every six months (February and August) and supports each until two newer majors exist. As of 2026-11-22 the current release is Go 1.27 (1.27.0 on 2026-08-19, patch 1.27.4 on 2026-11-10) and the supported set is 1.27 and 1.26; Go 1.25 and older receive no security fixes. Build and ship with the latest patch of a supported major.

## The `go` and `toolchain` Lines

- The `go` line in `go.mod` is the minimum Go version the module requires. It gates language features (a file cannot use a feature newer than the line), sets loop-variable and timer semantics, and amends the GODEBUG defaults the main module's binaries run with — the same toolchain builds different runtime behavior under different `go` lines.
- When the local toolchain is older than the `go` line, `GOTOOLCHAIN=auto` (the default) downloads the required toolchain; the optional `toolchain` line names a preferred newer toolchain. Since Go 1.25 the go command no longer adds a `toolchain` line when it updates the `go` line.
- `go mod init` (Go 1.26+) writes the previous major (`go 1.(N-1).0`) so new modules stay buildable by the older supported toolchain. A library raises its `go` line only when it needs a newer feature; an application may track the latest supported release.
- Raise the line with `go get go@1.27.0` (or `go mod edit -go=`), then run `go fix ./...` and the tests. For `go 1.27`+ modules, `go mod tidy` consolidates `require` blocks into at most two (direct, indirect).
- Since Go 1.27 `go test` runs the `stdversion` vet check by default: it reports standard-library symbols newer than the file's effective Go version.

## Feature Baseline by Version

| Version | Feature | Use it for |
|---|---|---|
| 1.21 | `min`/`max`/`clear` builtins; `slices`, `maps`, `cmp` packages; `log/slog` | sorting, searching, and structured logging without third-party helpers |
| 1.22 | per-iteration loop variables; `range` over integers; method- and wildcard-pattern `net/http.ServeMux` (`"GET /items/{id}"`, `r.PathValue`) | dropping `v := v` copies and most router dependencies |
| 1.23 | range-over-func iterators (`func(yield func(K, V) bool)`), the `iter` package (`iter.Seq`, `iter.Seq2`, `iter.Pull`), `slices.All/Values/Backward/Collect/Sorted/Chunk`, `maps.All/Keys/Values/Insert/Collect`; `unique` interning; unbuffered, collectable timers | exposing collections as `iter.Seq` instead of returning slices or callbacks |
| 1.24 | generic type aliases; `omitzero` JSON tag option (zero value or `IsZero()`); `os.Root`/`os.OpenRoot`; `strings`/`bytes` `Lines`, `SplitSeq`, `FieldsSeq`; `weak` pointers and `runtime.AddCleanup`; Swiss-table maps; `tool` directives in `go.mod` | `omitzero` for `time.Time` and struct fields, where `omitempty` never omitted them |
| 1.25 | `sync.WaitGroup.Go`; `testing/synctest`; container-aware `GOMAXPROCS`; `runtime/trace.FlightRecorder`; `net/http.CrossOriginProtection`; `os.Root` gains `MkdirAll`, `ReadFile`, `WriteFile`, `RemoveAll`, `Rename`, and more; `ignore` directive in `go.mod` | `wg.Go(f)` instead of `wg.Add(1)`/`go`/`defer wg.Done()` |
| 1.26 | `new(expr)` (pointer to an expression's value); self-referential generic constraints; `errors.AsType`; `slog.NewMultiHandler`; `reflect` iterator methods; Green Tea GC by default; rewritten `go fix` | `new(computeAge())` for optional pointer fields; typed error extraction |
| 1.27 | generic methods (methods may declare their own type parameters; interface methods may not, and generic methods never satisfy interface methods); `encoding/json/v2` and `encoding/json/jsontext` (with `encoding/json` now backed by v2); `uuid` package (`uuid.NewV4`, `uuid.NewV7`, `uuid.Parse`); `strings.CutLast`/`bytes.CutLast`; `goroutineleak` profile GA | namespacing generic helpers on their receiver type; standard-library UUIDs |

### Iterators

Expose sequences as `iter.Seq[V]` / `iter.Seq2[K, V]` when consumers stream or filter them; materialize with `slices.Collect` only where a slice is required. An iterator honors `yield`'s `false` return (stop immediately), documents whether it is single-use, and uses `iter.Pull` only when two sequences must be advanced in lockstep — a pulled iterator must have its `stop` function called.

```go
func (s *Store) Items() iter.Seq[Item] {
    return func(yield func(Item) bool) {
        for _, it := range s.items {
            if !yield(it) {
                return
            }
        }
    }
}

for it := range store.Items() { /* ... */ }
sorted := slices.SortedFunc(store.Items(), compareItems)
```

### `encoding/json/v2`

Since Go 1.27 `encoding/json/v2` is a standard package and `encoding/json` runs on its implementation with v1 behavior preserved (error text may differ; `GOEXPERIMENT=nojsonv2` restores the old implementation as a temporary escape hatch). v2 defaults are stricter: case-sensitive member matching, invalid UTF-8 and duplicate member names rejected, nil slices and maps encoded as `[]`/`{}`, and `omitempty` judged by JSON emptiness while `omitzero` judges the Go zero value. New code that controls both ends of its JSON may adopt v2; existing v1 callers need not migrate — the v1 package gains `Options` that let v2 calls reproduce v1 semantics where a gradual migration needs them.

### `os.Root` for Untrusted Paths

When a path comes from input (archive entries, request parameters, config), open the directory once with `os.OpenRoot` and operate through the `*os.Root`: its methods refuse paths that escape the root, including through `..` and symlinks. `filepath.Join` plus a prefix check is not equivalent — it misses symlink escapes and check-then-use races.

## Keeping Code Current: `go fix`

Since Go 1.26 `go fix` hosts the modernizers: analyzers whose fixes rewrite code to newer idioms without changing behavior, gated by each file's Go version. Run `go fix -diff ./...` to review and `go fix ./...` to apply after raising the `go` line. The Go 1.27 suite includes, among others: `errorsastype` (`errors.As` → `errors.AsType`), `forvar` (drop redundant loop-variable copies), `rangeint`, `minmax`, `slicescontains`, `slicessort`, `slicesbackward`, `mapsloop`, `stringsseq`, `stringscut`, `stringscutprefix`, `stringsbuilder`, `testingcontext` (`context.WithCancel` in tests → `t.Context()`), `waitgroupgo` (`wg.Add(1)`/`go`/`wg.Done()` → `wg.Go`), `omitzero`, `newexpr`, `any`, and `inline` (applies `//go:fix inline` directives, the way a library deprecates a function in favor of its replacement). `go tool fix help` lists the analyzers the installed toolchain carries.

## Static Analysis and Vulnerability Scanning

- `go vet` runs on every `go test`; recent additions worth knowing are `waitgroup` and `hostport` (1.25: misplaced `WaitGroup.Add`, `Sprintf("%s:%d")` addresses that break on IPv6), the printf check for non-constant format strings (1.24) and for `%w` applied to `*E` where `E` implements `error` (1.27), and `stdversion` in the default `go test` set (1.27).
- golangci-lint is on its v2 line (v2.0.0 on 2025-03-24; v2.15.0 on 2026-10-16): v2 configurations declare `version: "2"`, select linters through `linters.default` plus enable/disable lists, and move formatters (gofmt, goimports, gofumpt) to a separate `formatters` section run by `golangci-lint fmt`; `golangci-lint migrate` converts a v1 file.
- `govulncheck ./...` (module `golang.org/x/vuln`) reports known vulnerabilities from the Go vulnerability database filtered to the ones the code actually reaches through its call graph — a lower-noise signal than version-only scanners. Run it in CI and before releases. `govulncheck -mode=binary` checks a built artifact from its symbol table — useful for shipped binaries, but less precise: no call stacks, and code linked in yet unreachable can be reported.

## `go.mod` Directives Beyond `require`

- `tool` (1.24): `go get -tool golang.org/x/vuln/cmd/govulncheck` records a tool dependency, versioned with the module, and `go tool govulncheck` runs it — this replaces the `tools.go` blank-import convention.
- `ignore` (1.25): excludes directories (for example a `node_modules/` or generated tree) from `./...` and `all` package patterns while keeping them in module zips.
- `godebug` (1.23): pins individual GODEBUG settings for the main module, for example to stage a behavior change separately from a `go` line bump.

## Sources

- https://go.dev/doc/devel/release — release history and the two-release support policy; go1.27.4 and go1.26.11 on 2026-11-10 (accessed 2026-11-22)
- https://go.dev/doc/go1.27 — generic methods, `encoding/json/v2` and `jsontext`, `uuid`, `CutLast`, `stdversion` in `go test`, printf `%w` check, new modernizers, `go mod tidy` consolidation (Go 1.27, released 2026-08-19; accessed 2026-11-22)
- https://go.dev/doc/go1.26 — `new(expr)`, self-referential constraints, `errors.AsType`, `slog.NewMultiHandler`, rewritten `go fix`, `go mod init` default version (Go 1.26, released 2026-02-10; accessed 2026-11-22)
- https://go.dev/doc/go1.25 — `sync.WaitGroup.Go`, `testing/synctest`, container-aware `GOMAXPROCS`, `os.Root` additions, `ignore` directive, `waitgroup`/`hostport` vet analyzers, no implicit `toolchain` line (Go 1.25, released 2025-08-12; accessed 2026-11-22)
- https://go.dev/doc/go1.24 — generic type aliases, `omitzero`, `os.Root`, string iterators, `tool` directives, `weak`, `runtime.AddCleanup`, Swiss-table maps (Go 1.24, released 2025-02-11; accessed 2026-11-22)
- https://go.dev/doc/go1.22 — per-iteration loop variables, range over integers, method and wildcard `ServeMux` patterns, `Request.PathValue` (Go 1.22; accessed 2026-11-22)
- https://go.dev/doc/go1.21 — `min`/`max`/`clear`, `slices`/`maps`/`cmp`/`log/slog`, the `go` line as a strict minimum and the `toolchain` directive (Go 1.21; accessed 2026-11-22)
- https://golangci-lint.run/docs/product/migration-guide/ — v2 `version` property, `linters.default` (`none`/`all`/`fast`), `formatters` section, `migrate` (accessed 2026-11-22)
- https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck — source versus binary mode precision (v1.10.0, published 2026-11-17; accessed 2026-11-22)
- https://go.dev/doc/go1.23 — range-over-func, `iter`, `slices`/`maps` iterator functions, `unique`, timer changes (Go 1.23; accessed 2026-11-22)
- https://pkg.go.dev/encoding/json/v2 — v2 API, options, and behavioral differences from v1 (go1.27.4; accessed 2026-11-22)
- https://pkg.go.dev/uuid — `New`, `NewV4`, `NewV7`, `Parse` (go1.27.4; accessed 2026-11-22)
- https://go.dev/doc/toolchain — the `go` and `toolchain` lines and `GOTOOLCHAIN` (accessed 2026-11-22)
- https://go.dev/doc/godebug — GODEBUG defaults amended by the `go` line and the `godebug` block (accessed 2026-11-22)
- https://golangci-lint.run/docs/product/changelog/ — v2.0.0 (2025-03-24) through v2.15.0 (2026-10-16), `fmt` and `migrate` commands (accessed 2026-11-22)
- https://go.dev/doc/security/vuln/ — govulncheck and the Go vulnerability database (accessed 2026-11-22)
- `go tool fix help` under go1.27.4 — the registered modernizer list (run 2026-11-22)

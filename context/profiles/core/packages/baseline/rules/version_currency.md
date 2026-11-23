---
paths:
  - "go.mod"
  - "*/go.mod"
  - "*/*/go.mod"
  - "*/*/*/go.mod"
  - "*/*/*/*/go.mod"
  - "*/*/*/*/*/go.mod"
  - "go.work"
  - "*/go.work"
  - "*/*/go.work"
---

# Go Version Currency

The `go` directive is a language and behavior switch, not a comment: it selects the language features a module may use and the GODEBUG defaults its binaries run with, so set it to a released, still-supported version (one of the two newest major releases) and move it deliberately. Write code to the idioms that version provides — iterators and the `slices`/`maps` helpers, `errors.AsType`, `sync.WaitGroup.Go`, `os.Root` for untrusted paths — and let `go fix` apply the mechanical modernizations after each bump. Track executable dependencies with `tool` directives and run `govulncheck` against the module rather than relying on dependency-version scanners alone.

- the release and support cadence, what the `go` and `toolchain` lines do, the feature-by-version table, the `go fix` modernizers, the vet and govulncheck posture, and the `tool` and `ignore` directives — [Modern Go Baseline](../reference/modern_go.md)

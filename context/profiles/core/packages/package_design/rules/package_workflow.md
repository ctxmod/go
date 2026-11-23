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
  - "go.mod"
  - "*/go.mod"
  - "*/*/go.mod"
  - "*/*/*/go.mod"
  - "*/*/*/*/go.mod"
  - "*/*/*/*/*/go.mod"
  - "*/*/*/*/*/*/go.mod"
  - "*/*/*/*/*/*/*/go.mod"
  - "*/*/*/*/*/*/*/*/go.mod"
  - "*/*/*/*/*/*/*/*/*/go.mod"
  - "go.work"
  - "*/go.work"
  - "*/*/go.work"
  - "*/*/*/go.work"
  - "*/*/*/*/go.work"
  - "*/*/*/*/*/go.work"
  - "*/*/*/*/*/*/go.work"
  - "*/*/*/*/*/*/*/go.work"
  - "*/*/*/*/*/*/*/*/go.work"
  - "*/*/*/*/*/*/*/*/*/go.work"
---

# Go Package Workflow

The Go workflow library defines how to design, implement, and validate Go packages. Consult it on demand during package work. [workflow_index.md](../reference/workflow_index.md) says which reference covers the language baseline, API design, package decomposition, testability, planning, refactoring, testing, and the validation thresholds.

Before a public-API change, count its out-of-repo consumers. Other checkouts that depend on this module's live working tree (for example through a `replace` directive pointing at a sibling directory) are invisible to the in-repo `go build ./...`.

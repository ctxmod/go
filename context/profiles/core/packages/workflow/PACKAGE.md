---
description: >-
  The index of the Go module: the tier-1 pointer to the workflow
  library and the reference index that names each principle, procedure
  and validation reference and when to read it.
binds_to_depth: main...
---

# Go

This module's rules load once the session reads or writes a `.go`, `go.mod` or `go.work` file. Even the first Go file wraps cross-package errors with `%w`, returns concrete types from constructors, and keeps non-public code under `internal/`.
The on-demand references, including the planning procedure to follow before the first Go file exists, are indexed in `context/profiles/core/packages/workflow/reference/workflow_index.md` in the github.com/ctxmod/go repository.

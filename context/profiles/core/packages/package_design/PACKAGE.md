---
description: >-
  The Go development workflow library: package design principles plus
  the planning, refactoring, and testing procedures and the validation
  suite that measures them; consult on demand during Go package work.
binds_to_depth: main...
---

# Go

This module's rules load once the session reads or writes a `.go`, `go.mod` or `go.work` file. Even the first Go file wraps cross-package errors with `%w`, returns concrete types from constructors, and keeps non-public code under `internal/`.
The on-demand references are indexed in `context/profiles/core/packages/package_design/reference/workflow_index.md` in the github.com/ctxmod/go repository.

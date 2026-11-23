---
paths:
  - "*.go"
---

# Constructor Return Types

Return concrete types (`*ConcreteType`) from constructors. Do not return interfaces from `New*` functions — concrete return types allow adding methods without breaking API compatibility, and consumers define their own narrow interfaces that concrete types satisfy implicitly through structural typing.

Abstract factories that select between implementations at runtime may return interfaces. For example, `NewStorage(cfg StorageConfig) Storage` returns the `Storage` interface when the constructor chooses between postgres and s3 backends based on configuration.

- why the standard library returns concrete types, accepting narrow interfaces at parameters, and the worked correct/incorrect constructor examples — [Interface Design Discipline](../reference/interface_design.md)

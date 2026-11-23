# Internal Visibility

Place non-public implementation under `internal/` packages. Target >50% internal code for libraries, >70% for services. Go's `internal/` directory enforces encapsulation at the compiler level — importers outside the parent tree cannot access it — so the guarantee does not rely on naming conventions or documentation.

Sub-packages intended for direct consumer import are exempt: a `config` package providing advanced configuration or a `types` package shared across multiple consumers may remain outside `internal/`.

- export thresholds, progressive disclosure, what to export vs hide, the remediation procedure, the directory-layout examples, and the internal-percentage tooling note — [Minimal Public API](../reference/minimal_api.md)

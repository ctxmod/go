# Contract Governance

The schema module is the single source for every consumer's generated
bindings; copies are never forked. Compatibility runs mechanically in CI
against the main branch — wire breaks (tag renumbering, scalar changes) and
source breaks (field renames) both fail before anything builds. Removed
fields reserve their tag numbers and names; packages version by namespace
(`<domain>.v1`); fields where zero is meaningful carry explicit presence
instead of sentinel values; generated code is never edited.

- the full governance discipline — schema-first boundaries, the two compatibility surfaces, the deprecation and reserved-tag path, presence semantics, and generated-code posture — [Contract Governance](../reference/contract_governance.md)

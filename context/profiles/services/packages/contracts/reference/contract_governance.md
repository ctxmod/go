# Contract Governance

Schema-first contract governance: a central schema module, CI compatibility gates against the main branch, reserved tags on deprecation, semver package namespacing, explicit presence semantics, and generated code that is never edited.

In an estate of many services, clients and servers rarely deploy at the same
moment. An uncoordinated schema change breaks downstream consumers silently —
or corrupts deserialization loudly. Contracts are therefore reviewed source
code with mechanical compatibility gates, not artifacts regenerated on a whim.

## Schema-First Boundaries

The wire contract exists before either side of it: a single central schema
module (a monorepo schema directory or a schema registry module) is the one
source from which every consumer's code is generated. Copy-pasting schema
definitions across services or repositories creates divergent contracts —
stale copies that parse today and desync on the next change. If two services
import the same message shape, they import it from the same module.

## Automated Compatibility Gates in CI

The schema module carries lint rules (naming, style, structure) and a
breaking-change detector that runs in CI against the main branch. A pull
request that would break wire or source compatibility fails before anything
builds against it. The two compatibility surfaces are distinct and both
gated:

- **Wire compatibility** — renumbering a tag, changing a scalar type, or
  repacking a message corrupts deserialization for every deployed peer.
- **Source compatibility** — some changes preserve the wire bytes yet break
  generated accessors in a consumer's language (a field rename is the classic
  case). Renames are additive: add the new field, migrate, then deprecate and
  remove the old one through the reserved-tag path below.

## Reserved Tags on Deprecation

A deprecated field is marked deprecated first and removed only after a
documented window. When removal lands, its tag number — and its name — are
entered as reserved. Reserved numbers and names can never be accidentally
reused by a later edit; reusing a removed field's number feeds old-in-flight
data into a new field's type, the most damaging corruption class there is.

## Semver Package Namespacing

Schema packages are version-namespaced (`<domain>.v1`): a breaking evolution
lands as a new package, not as a mutation of the existing one. Consumers
migrate from `v1` to `v2` on their own schedule; the two versions coexist on
the wire during the migration window. Within a version, every change is
backward-compatible by the CI gate.

## Presence Semantics and Zero-Value Ambiguity

Scalar fields without explicit presence cannot distinguish *unset* from
*intentionally zero* — a reader cannot tell "no timeout given" from "timeout
of zero." Where that distinction matters, make presence explicit: the
language's optional-presence field form, or a well-known wrapper type. Do not
invent sentinel values (`-1` means unset) — sentinels are undocumented wire
semantics that every consumer must rediscover. Fields where zero is a valid,
meaningful value are the ones that need this; fields where zero and unset are
equivalent do not.

## Generated Code Is Never Edited

Generated bindings are build artifacts with a committed or deterministically
regenerated posture — owned by the repository's generated-code discipline.
They are never hand-edited, and a schema change is reviewed through the
schema diff, with regeneration producing the code change mechanically. Hand
edits survive exactly one regeneration.

## Boundary with Protocol-Specific Schema Work

This reference owns the governance mechanics for service wire contracts.
Vendoring and tracking *protocol* schemas from external specifications
(agent protocols and similar) is a distinct concern with its own upstream
sync discipline, owned by the AI guidance where a repository carries it; the
frontends' consumption of generated bindings is owned by the frontend
profiles. Neither repeats what is here: the gates, reserved-tag rules, and
presence semantics apply wherever a schema module exists, whoever vendors it.

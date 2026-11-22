# Kong grammar footguns

Kong derives flag names from Go field names via camelCase
splitting at letter/digit class boundaries (`P256dh` →
`--p-256-dh`, `WtExtensionOrigins` → `--wt-extension-origins`),
treats `arg:""` fields as POSITIONAL arguments, and applies
`envprefix` only to the struct it tags — a nested subcommand
struct's `env:"X"` binds verbatim without the prefix. A name
written in the name-position of a `kong:"..."` tag (e.g.
`kong:"--foo,help='...'"`) is silently ignored: the tag parser
hydrates only known keys, so the dashed token is stored and
never consulted. Each of these has produced a live incident in
practice: a push handler invoked with the wrong argv, a
crash-looping service deploy, and a silently ignored
environment variable.

Test the derived grammar, not the struct you meant to write. A
small pure helper that walks the parsed Kong model (the
`*kong.Application` node tree: each node's flags with their
derived names and env bindings, and its positionals) can
enumerate the DERIVED surface so a test asserts the exact argv
shape a tool really accepts; a sibling audit helper reports the
specific defect shapes above (digit-split names, unprefixed
nested env bindings, dashed name-position tokens). Keep both
pure — no logging, no global state — so they are safe to call
from tests or tools.

## Shared global flags: short aliases and enum sync

When a shared global-flags struct is embedded by many tools,
leave short aliases off flags whose letter is conventionally
claimed by commands — `-o` is the conventional short flag for a
command's own output path. Kong rejects a duplicate short flag
at parser-construction time, so binding `-o` on the shared
struct would panic every binary whose command already claims
it, before it parses a single argument. A tool that wants `-o`
for the output format declares it on its own root struct.

An `enum:"..."` tag duplicates a list the code owns elsewhere
(for example the output package's valid modes), so the two MUST
stay in sync: when the output package declares, accepts,
documents, and renders a mode but the Kong enum omits it, the
mode is unparseable for every tool embedding the flags while
`--help` advertises it. Derive the enum from the source list or
pin the two together with a test.

## Kong defaults erase "unset"

Kong's precedence is flag > env > default, and a `default:"..."`
tag fills the field unconditionally. When downstream code needs
to tell "the user pinned nothing" from "the user pinned the
default value" — for example so an explicit flag or exported
environment variable wins over config-file settings, while an
absent one lets the config file decide — leave the Kong default
off, let absent materialize as the empty string, and resolve the
empty sentinel to the effective default where the renderer is
built. The empty value is a sentinel, not a gap.

## Parent-command flags and parent Run on subcommand paths

A flag declared on a parent command struct is also in scope for
every subcommand beneath it (`tool wait list --repo X` parses
against the parent's `--repo`). Redeclaring the same long or
short flag on a child struct is a duplicate: Kong panics in
`kong.Must` at parser construction (`waitListCmd.Repo: duplicate
flag --repo`), before any argument is parsed. Because every
entrypoint of a binary — including hook entrypoints that other
tools invoke — shares its single grammar, one duplicate in ANY
subcommand kills every entrypoint: a hook that exits non-zero on
the panic can then block every caller that depends on it.
Declare the flag once on the parent and read it from the child's
`Run` through the bound root (`root.Wait.Repo`).

When a command struct has both a `Run` method and child
`cmd:""` fields, Kong calls `Run` on every node of the selected
path, not only the leaf: `tool wait list` runs
`waitListCmd.Run` and then `waitCmd.Run`. A parent whose `Run`
assumes its own flags were supplied fails after the child
succeeded. Guard the parent by taking `kctx *kong.Context` and
returning early when `kctx.Selected()` is not the parent node.
The example project's `deploy` group shows both fixes.

After adding or moving any flag, smoke the built binary
(`<tool> --help`, the bare parent, and each subcommand) — the
duplicate-flag panic surfaces only at runtime, never at compile.

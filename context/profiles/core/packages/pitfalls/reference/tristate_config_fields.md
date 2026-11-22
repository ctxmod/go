# Tri-State Config Field Evolution

Three traps await when evolving a `bool` config field to a durable
tri-state (unset/true/false) in YAML: `omitempty` fixes the wrong
collapse, `*bool` silently breaks struct `==`, and gating emission on
declaredness alone loses a true value.

Making an on-disk `bool` express three states (key absent = unset,
`true`, `false`) has three traps. A worked shape: an instance config
whose `IncludeAPICompletions bool` is paired with a `yaml:"-"` presence
flag (`APICompletionsDeclared bool`). The tri-state is load-bearing
there — an explicit `false` is authoritative and blocks a filter from
implying the feature, while "unset" leaves the implication in force;
collapsing either direction is a silent behavior change. Apply this
whenever "absent" and "explicitly false" must differ.

## Trap 1 — `omitempty` fixes the wrong collapse

`omitempty` drops the zero value, so *explicitly false* becomes
indistinguishable from *unset*. It trades an unset→false collapse for
a false→unset one. Never the answer for a tri-state bool.

## Trap 2 — `*bool` silently changes `==` on the containing struct

A common convention (nil-means-unset pointers, e.g. an optional
`secret_baseline` setting) makes `*bool` look idiomatic. But promoting a field
on an existing struct to `*bool` keeps the struct **compilably**
comparable while turning `==` into pointer identity: every `a == b`
and every `configsEqual`-style helper keeps compiling and starts being
wrong. Check for `==` usage on the struct before reaching for the
pointer; a `yaml:"-"` presence flag plus a custom marshaller has a far
smaller blast radius when the struct is compared.

## Trap 3 — gating emission on declaredness alone loses a true value

The intuitive predicate `if !declared { drop the key }` is wrong for
in-memory literals built with `value: true` and the presence flag left
at its zero value: the key is dropped and the **value** is lost, not
just its declaredness — persisting an enable as an off. Correct
predicate: `declared || value`. A `true` is self-declaring; only
`undeclared && false` is genuinely unset.

## Suppressing one key without a hand-written marshaller

A hand-written `MarshalYAML` that enumerates fields silently drops any
field added later. Instead, shed the method with a local type and
delete one pair from an encoded node:

```go
type marshalPlain InstanceConfig // sheds MarshalYAML, prevents recursion

func (c InstanceConfig) MarshalYAML() (any, error) {
	if c.emitsGate() {
		return marshalPlain(c), nil // byte-identical to the pre-change output
	}
	var node yaml.Node
	if err := node.Encode(marshalPlain(c)); err != nil { /* ... */ }
	dropMappingKey(&node, "the_key") // walk Content in steps of 2
	return &node, nil
}
```

`yaml.v3` accepts a `*yaml.Node` returned from `MarshalYAML` — its
marshal switch has `case *Node` *before* `case Marshaler`, and
`Node.Encode` round-trips through the emitter and parser, so scalar
styles are preserved and the gated body differs from the ungated one
by exactly the removed line. Pin that with a test that saves the same
config both ways and diffs.

## Test signal

If a change like this makes existing round-trip tests fail
(`TestAPISplit_ThreeTogglesRoundTrip`,
`TestMarshalRoundTrip/openai`, `TestMarshalRoundTrip/api-free` caught
trap 3 immediately), read the failures before adjusting the tests —
they are encoding the tri-state contract.

# Interface Size

Prefer interfaces with 1-5 methods. Decompose interfaces exceeding 10 methods into composed smaller interfaces. "The bigger the interface, the weaker the abstraction" — each additional method reduces the set of types that can satisfy the interface, and large interfaces resist decomposition once consumers depend on them.

Standard library interfaces (`http.Handler`, `database/sql.Driver`) and framework contracts where the definition is externally imposed are exempt. The `interfacebloat` analyzer detects interfaces exceeding method count thresholds.

- the per-method-count assessment table, composition guidance, consumer-defined interfaces, interface-pollution prevention, and the worked correct/incorrect examples — [Interface Design Discipline](../reference/interface_design.md)

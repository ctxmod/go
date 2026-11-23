# Go CLI Reference Map

Where each part of the Kong-structured CLI stack is documented, for the [Framework Usage](../rules/framework_usage.md) rule.

- tag grammar, flag-derivation traps, bind system, lifecycle hooks, the Kong conventions list, and the worked correct/incorrect command examples — [Kong Command Parsing](../reference/kong_parsing.md)
- Elm Architecture, program options, Bubbles catalog, and the scenario table for choosing Bubble Tea over the output interface — [Bubble Tea and Bubbles](../../tui/reference/bubbletea_bubbles.md)
- the output abstraction Kong commands render through — [Output Interface](../../output/reference/output_modes.md)
- the koanf precedence stack that sits behind Kong's grammar — [Configuration Layering with koanf](../../config/reference/config_layering.md)
- the `Runtime` injected into every `Run()` method — [Runtime](../../output/reference/runtime_injection.md)
- camelCase flag derivation, positional-arg and name-position footguns, testing the derived grammar, shared global-flag short-alias and enum-sync constraints, Kong defaults erasing "unset", parent-flag redeclaration panics, and parent `Run` firing on subcommand paths — [Kong grammar footguns](../reference/kong_footguns.md)
- scaffolding a new CLI project with the Kong grammar, shared config/output/runtime packages, and context-based dependency injection — [CLI Project Scaffolding](../../scaffolding/reference/project_scaffolding.md); the complete buildable project is the profile's example, `examples/cli_project`

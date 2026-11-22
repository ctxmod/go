# Interface Design Discipline

Accept interfaces in function parameters. Return concrete types from constructors. Keep interfaces small and let consumers define them.

## Constructor Return Types

Research corrects a common misconception. The initial hypothesis -- that constructors should return interface types to hide implementation details and enable mocking -- is wrong for the general case. The dominant Go pattern, validated across the standard library and 2024-2026 community sources, is to return concrete types.

Returning `*ConcreteType` from constructors allows adding methods without breaking API compatibility. Consumers can define their own narrow interfaces that the concrete type implicitly satisfies, achieving the same goals (testability, abstraction) without locking the API to a fixed contract. Returning an interface prevents consumers from accessing new methods added to the concrete type unless the interface definition is updated.

Standard library examples confirm this pattern: `http.NewRequest` returns `*Request`, `bufio.NewReader` returns `*Reader`, `bytes.NewBuffer` returns `*Buffer`. None return interfaces.

**Exception**: Abstract factory constructors where runtime polymorphism selects among implementations. When `NewStorage(cfg)` chooses between postgres and s3 backends, returning a `Storage` interface is appropriate because the caller cannot know the concrete type at compile time. This exception applies only when multiple implementations exist behind a single constructor.

## Accepting Interfaces

Function parameters should accept the narrowest interface the function needs. Prefer `io.Reader` over `*os.File` when only `Read` is called. This widens the set of types callers can pass and decouples the function from specific implementations.

Narrowing parameters to minimal interfaces enables callers to substitute test doubles, alternative implementations, or wrapper types without modification to the function signature.

## Consumer-Defined Interfaces

Interfaces belong in the package that consumes them, not the package that implements them. A consumer defines only the methods it calls, producing naturally small interfaces.

```go
// In consumer package -- defines only what it needs
type UserSaver interface {
    SaveUser(u *User) error
}

func Register(s UserSaver, u *User) error { ... }
```

The producer package exports a concrete `*Store` type with many methods. Each consumer defines its own one- or two-method interface. The concrete type satisfies all of them implicitly.

**When to break this rule**: Shared contracts used by three or more consumer packages belong in a neutral package (`types`, `api`, or a parent package) to avoid duplication. Below that threshold, each consumer should own its interface.

## Interface Size

Rob Pike's proverb -- "The bigger the interface, the weaker the abstraction" -- provides the governing heuristic. Each additional method constrains the number of types that can satisfy the interface, reducing its utility.

| Methods | Assessment | Standard Library Examples |
|---------|------------|--------------------------|
| 1 | Ideal | `io.Reader`, `io.Writer`, `io.Closer`, `fmt.Stringer` |
| 2-5 | Acceptable | `io.ReadCloser`, `sort.Interface` |
| 6-10 | Excessive | Signals potential Interface Segregation Principle violation |
| >10 | Decompose | Split into composed smaller interfaces |

Compose small interfaces into larger ones only when consumers genuinely need all methods together. `io.ReadWriteCloser` composes three single-method interfaces; it exists because real consumers (like network connections) use all three operations.

## Interface Pollution Prevention

Creating an interface alongside its only implementation ("Service interface + DefaultService struct") for mocking purposes is an anti-pattern. Instead:

1. Use the concrete type directly.
2. Create an interface only when two real implementations exist or a consumer needs to narrow scope for testing.
3. Let consumers define test-scoped interfaces with the subset of methods they call.

Pre-emptive interfaces add abstraction cost without abstraction benefit. Wait for actual polymorphism needs before introducing them.

## Trade-offs

Returning concrete types means consumers see all exported methods and fields. This increases the visible API surface per type. However, consumer-defined interfaces mitigate this: each consumer narrows its view to exactly the methods it uses, restoring the abstraction boundary at the consumption site rather than the production site.

The concrete-return pattern also makes `go doc` output more informative -- consumers see the full type capability rather than an interface's limited view. For package documentation and discoverability, this is an advantage.

## Interface Size (rule context)

Extracted verbatim from the `go-interface-size-context` rule so that rule can stay a pointer.

### Rationale

Rob Pike's proverb applies directly: "The bigger the interface, the weaker the abstraction." Each additional method reduces the set of types that can satisfy the interface, limiting reuse and testability. Small interfaces compose naturally through embedding, while large interfaces resist decomposition after consumers depend on them.

### Examples

**Correct**:
```go
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
type ReadWriter interface {
    Reader
    Writer
}
```

**Incorrect**:
```go
type FileSystem interface {
    Open(path string) (File, error)
    Create(path string) (File, error)
    Remove(path string) error
    Rename(old, new string) error
    Stat(path string) (FileInfo, error)
    ReadDir(path string) ([]DirEntry, error)
    MkdirAll(path string, perm os.FileMode) error
    WriteFile(path string, data []byte, perm os.FileMode) error
    ReadFile(path string) ([]byte, error)
    Chmod(path string, mode os.FileMode) error
    Chown(path string, uid, gid int) error
    Walk(root string, walkFn WalkFunc) error
}
```

### Exceptions

Standard library interfaces (e.g., `http.Handler`, `database/sql.Driver`) and framework contracts where the interface definition is externally imposed.

### Tooling

`interfacebloat` analyzer detects interfaces exceeding method count thresholds.

## Constructor Return Types (rule context)

Extracted verbatim from the `go-constructor-return-types-context` rule so that rule can stay a pointer.

### Rationale

Concrete return types allow adding methods without breaking API compatibility. Consumers define their own narrow interfaces and concrete types satisfy them implicitly through structural typing. Returning interfaces from constructors locks the API to a fixed method set and prevents non-breaking evolution.

### Examples

**Correct**:
```go
func NewResolver(options ...ResolverOption) *Resolver {
    return &Resolver{}
}
```

**Incorrect**:
```go
func NewResolver(options ...ResolverOption) Resolver {
    return &resolver{}
}
```

### Exceptions

Abstract factories that select between implementations at runtime may return interfaces. For example, `NewStorage(cfg StorageConfig) Storage` returns the `Storage` interface when the constructor chooses between postgres and s3 backends based on configuration.

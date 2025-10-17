# Domain Layer

The domain layer contains Streamy's pure business logic. Code placed here must **never** import from `internal/application`, `internal/infrastructure`, or any `internal/ports` packages. The goal is to keep this layer independent and testable without external dependencies.

## Rules

1. **No Infrastructure Awareness**: Domain entities work with Go primitives, value objects, and domain-specific types only.
2. **Zero Side Effects**: Avoid file I/O, network calls, logging, or metric emission. Those concerns live in adapters.
3. **Deterministic Behavior**: Methods should behave identically given the same inputs; avoid hidden state.
4. **Validation at the Edges**: Constructors or factory functions verify invariants (unique step IDs, valid step types, etc.).
5. **Context-Friendly**: When a domain method requires cancellation or deadlines, accept `context.Context` as the first parameter.
6. **Tests Co-Located**: Keep `*_test.go` files next to the implementation. Tests must run without mocks or external setup.

## Structure

```
internal/domain/
├── pipeline/   # Pipeline aggregate root, steps, execution plan, results, errors
└── plugin/     # Plugin metadata and domain-level contracts
```

## Adding Domain Code

1. Define entities and value objects with clear invariants.
2. Write table-driven tests before implementation when possible.
3. Use constructor functions (`NewType`) if initialization requires validation.
4. Export only what other layers require; keep helpers unexported.
5. Document domain rules inline so maintainers understand the intent.

## Domain Errors

Use structured errors with well-known codes (`ErrCodeValidation`, `ErrCodeExecution`, etc.) to allow higher layers to react appropriately. Domain errors should include contextual metadata but not wrap infrastructure details.

## Testing

```
go test ./internal/domain/... -cover
```

Domain tests must complete in under 100ms to satisfy success criterion SC-001.


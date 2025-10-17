# Application Layer

The application layer orchestrates use cases by coordinating domain entities and port interfaces. It contains services such as pipeline preparation, execution, verification, and validation. This layer is responsible for:

- Translating user intent (CLI/TUI commands) into domain operations
- Orchestrating workflows across multiple ports
- Aggregating results and errors for presentation

## Rules

1. **Dependency Direction**: Imports are allowed from `internal/domain/...` and `internal/ports`, but not from `internal/infrastructure`.
2. **Constructor Injection**: All services receive dependencies through constructors using port interfaces. No global state.
3. **Context Propagation**: Every method must accept `context.Context` as the first argument and pass it to dependencies.
4. **Error Wrapping**: Application services wrap domain errors with user-oriented context while preserving the error chain (`fmt.Errorf("...: %w", err)`).
5. **Testing**: Unit tests live alongside the code and use mocks from `internal/application/<feature>/testutil` to simulate port behavior.

## Layout

```
internal/application/
├── pipeline/
│   ├── prepare_usecase.go
│   ├── apply_usecase.go
│   ├── verify_usecase.go
│   └── testutil/ (mocks for ConfigLoader, PluginExecutor, etc.)
└── validation/
    ├── service.go
    └── service_test.go
```

## Adding a Use Case

1. Define a struct that encapsulates the required ports.
2. Provide a constructor that validates dependencies and returns the use case.
3. Implement behavior using domain entities and ports only.
4. Write table-driven tests using mocks to verify success and failure paths.
5. Update wiring in `cmd/streamy/main.go` to inject real infrastructure adapters.

## Testing Commands

```
go test ./internal/application/... -cover
```

Coverage for this layer must remain above 90% (SC-002). Use mocks to isolate tests from infrastructure.


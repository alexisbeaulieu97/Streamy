# Research: Create Streamy MVP

**Date**: 2025-10-03  
**Feature**: Streamy MVP - Declarative Environment Setup Tool

## Technology Stack Decisions

### 1. Programming Language: Go 1.25+

**Decision**: Use Go for the core implementation

**Rationale**:
- Single binary distribution with zero runtime dependencies (static linking)
- Excellent cross-platform support (Linux, macOS, Windows native)
- Strong concurrency primitives (goroutines, channels) for DAG parallel execution
- Rich standard library for file operations, shell execution, networking
- Fast compilation and startup times (<100ms cold start achievable)
- Growing ecosystem of CLI and TUI libraries

**Alternatives Considered**:
- Rust: Excellent performance and safety, but steeper learning curve, longer compile times
- Python: Easier prototyping, but requires Python runtime (violates onboarding-first principle)
- Node.js: Good ecosystem, but large binary size and runtime dependency

### 2. CLI Framework: Cobra + Viper

**Decision**: Use `spf13/cobra` for CLI commands and `spf13/viper` for application configuration

**Rationale**:
- Cobra is the de facto standard for Go CLI applications (used by kubectl, Hugo, GitHub CLI)
- Automatic help generation, flag parsing, subcommand support
- Viper integrates seamlessly with Cobra for config file + env var + flag precedence
- Well-documented with extensive community support

**Alternatives Considered**:
- cli/cli: Simpler but less feature-rich, no nested commands
- urfave/cli: Popular but less idiomatic for complex CLIs

**Important Note**: Viper is used for **application-level** configuration (global settings, CLI flags), NOT for parsing user YAML configs. User pipeline configs are parsed with go-yaml to avoid coupling.

### 3. YAML Parsing: gopkg.in/yaml.v3

**Decision**: Use `gopkg.in/yaml.v3` for YAML parsing

**Rationale**:
- Pure Go implementation (no C dependencies)
- Excellent error messages with line/column numbers
- Supports custom unmarshalers for complex types
- Mature and battle-tested

**Alternatives Considered**:
- Viper: Inappropriate for user configs (adds unnecessary abstraction)
- goccy/go-yaml: Good performance but less mature

### 4. Schema Validation: go-playground/validator

**Decision**: Use `go-playground/validator/v10` for struct validation

**Rationale**:
- Declarative validation via struct tags (`validate:"required,min=1"`)
- Rich built-in validators (required, min, max, regex, url, etc.)
- Custom validator support for domain-specific rules
- Clear error messages with field paths

**Alternatives Considered**:
- Manual validation: Error-prone and verbose
- JSON Schema: Requires separate schema files, less Go-idiomatic

### 5. TUI Framework: Bubbletea + Lipgloss + Bubbles

**Decision**: Use Charm's TUI stack

**Rationale**:
- Bubbletea: Elm-inspired architecture (model-update-view) for predictable state management
- Lipgloss: Declarative styling with excellent terminal compatibility
- Bubbles: Pre-built components (spinners, progress bars, lists) accelerate development
- Excellent CI/non-interactive mode support (fallback to plain text)
- Active development and strong community

**Alternatives Considered**:
- tview: Feature-rich but more imperative, steeper learning curve
- termui: Good for dashboards but less suited for interactive workflows
- Plain fmt.Printf: Too basic, poor UX

### 6. Logging: zerolog

**Decision**: Use `rs/zerolog` for structured logging

**Rationale**:
- Zero-allocation design for high performance
- Supports both JSON (machine-readable) and console (human-readable) output
- Leveled logging (debug, info, warn, error)
- Contextual logging with fields for step IDs, durations, etc.

**Alternatives Considered**:
- logrus: Heavier, slower, more allocations
- zap: Excellent performance but more complex API
- Standard log: Too basic, no structured logging

### 7. Git Operations: go-git

**Decision**: Use `go-git/go-git/v5` for repository cloning

**Rationale**:
- Pure Go implementation (no external git binary dependency)
- Satisfies "onboarding first" principle (zero dependencies)
- Supports all common git operations (clone, checkout, pull)
- Good documentation and examples

**Alternatives Considered**:
- Exec git: Requires git binary installed (violates onboarding principle)
- libgit2 bindings: Requires CGo, complicates cross-platform builds

### 8. Concurrency Model: Go Standard Library

**Decision**: Use goroutines + channels + sync package for DAG execution

**Rationale**:
- Native Go concurrency primitives are well-suited for DAG execution
- Worker pool pattern limits resource usage (configurable parallelism)
- sync.WaitGroup for waiting on parallel steps
- context.Context for timeouts and cancellation

**Implementation Pattern**:
```
1. Build DAG from config (detect cycles)
2. Topological sort to find execution order
3. For each level in DAG:
   - Dispatch independent steps to worker pool
   - Wait for all workers to complete
   - Check for errors (fail-fast)
4. Proceed to next level or halt on error
```

### 9. Testing Strategy: Go Testing + Testify

**Decision**: Use standard `testing` package with `testify/assert`

**Rationale**:
- Go's built-in testing is sufficient for unit tests
- Testify adds better assertion syntax without heavy framework
- Table-driven tests for comprehensive coverage
- Integration tests with real YAML configs in testdata/

**Test Coverage Goals**:
- Unit tests: 80%+ coverage for core packages (engine, config, plugins)
- Integration tests: All 5 step types, happy path, error cases
- Contract tests: Plugin interface compliance

### 10. Build and Distribution: GitHub Actions + Goreleaser

**Decision**: Use GitHub Actions for CI/CD with Goreleaser for multi-platform builds

**Rationale**:
- GitHub Actions is free for public repos, well-integrated
- Goreleaser automates cross-compilation for multiple OS/arch
- Generates checksums, creates GitHub releases, supports Homebrew tap
- Reproducible builds for security

**Release Targets**:
- Linux: amd64, arm64 (Debian/Ubuntu, Fedora, Arch)
- macOS: amd64, arm64 (Intel, Apple Silicon)
- Windows: amd64 (native, no WSL)

## Performance Considerations

### DAG Execution Optimization

**Approach**: Topological sort with level-based parallelism

**Strategy**:
- Precompute execution levels (all steps in a level have no dependencies on each other)
- Execute each level fully before proceeding to next
- Within a level, use bounded worker pool to limit concurrency
- Default pool size: 4 workers (configurable via --parallel flag)

**Expected Performance**:
- DAG construction: O(V + E) where V=steps, E=dependencies
- Typical config (50 steps): <10ms to build DAG
- Dry-run (no actual execution): <100ms including validation

### Memory Optimization

**Approach**: Streaming execution with minimal buffering

**Strategy**:
- Parse YAML once, hold config in memory (typically <1MB)
- Execute steps lazily (no pre-loading of all outputs)
- Stream TUI updates incrementally (no full buffer)
- Target: <50MB RSS for 100-step config execution

### Binary Size Optimization

**Approach**: Static linking with minimal dependencies

**Strategy**:
- Use `go build -ldflags="-s -w"` to strip debug symbols
- Statically link all dependencies (no dynamic libs)
- Compress with UPX if needed (but test compatibility)
- Target: <20MB uncompressed binary

## Cross-Platform Considerations

### Path Handling

**Decision**: Always use `filepath` package, never hardcode separators

**Strategy**:
- `filepath.Join()` for path construction
- `filepath.Clean()` for normalization
- `filepath.Abs()` for absolute paths
- `filepath.FromSlash()` for converting YAML paths

### Shell Command Execution

**Decision**: Use platform-specific shell detection

**Strategy**:
- Linux/macOS: `/bin/sh -c` (POSIX-compliant)
- Windows: `cmd /c` (native Windows shell)
- Detect shell from SHELL env var if available
- Document shell requirements in YAML schema

### Package Manager Detection (MVP)

**Decision**: MVP supports apt only, plugin architecture allows future expansion

**Strategy**:
- Check for `apt-get` binary in PATH
- Fail gracefully with clear message on non-Debian systems
- Document apt-only limitation in README
- Post-MVP: Add brew (macOS), choco/winget (Windows), pacman (Arch), dnf (Fedora) plugins

### File Permissions

**Decision**: Preserve permissions during copy, respect umask for new files

**Strategy**:
- Copy: `io.Copy()` + `os.Chmod()` to preserve mode
- Symlink: Use `os.Symlink()` (cross-platform)
- Create: Respect system umask (default 0644 for files, 0755 for dirs)

## Security Considerations

### Command Injection

**Risk**: Malicious YAML could inject shell commands

**Mitigation**:
- Never concatenate user input into shell strings
- Use `exec.Command()` with separate args array
- Validate command paths (no relative paths like `../`)
- Warn users about running untrusted configs

### Path Traversal

**Risk**: Symlink/copy operations could escape intended directories

**Mitigation**:
- Validate paths don't contain `..` or symlink tricks
- Use `filepath.Clean()` and `filepath.Abs()` before operations
- Check target paths are within expected boundaries

### Privilege Escalation

**Risk**: Package installation requires elevated privileges

**Mitigation**:
- Detect if running as root (warn if not needed)
- Recommend using sudo only for package steps
- Document security implications in README

## Open Questions (Deferred to Implementation)

1. **Logging to file**: Should --verbose log to file in addition to TUI?
   - Deferred: MVP logs to stdout/stderr only
   - Post-MVP: Add --log-file flag

2. **Config includes/imports**: Should configs support including other YAML files?
   - Deferred: MVP is single-file only
   - Post-MVP: Add `imports: [base.yaml]` field

3. **Secrets management**: How should users provide secrets (API keys, passwords)?
   - Deferred: MVP uses env vars only
   - Post-MVP: Consider secret manager integration

4. **Rollback on failure**: Should Streamy attempt to undo completed steps on failure?
   - Deferred: MVP is fail-fast with no automatic rollback
   - Post-MVP: Optional rollback with plugin-defined undo operations

5. **Progress persistence**: Should Streamy remember which steps completed across runs?
   - Deferred: MVP is stateless (relies on plugin Check() for idempotency)
   - Post-MVP: Optional state file for resume capability

## Conclusion

The technology stack selected prioritizes the constitutional principles:
- **Onboarding First**: Single Go binary, no dependencies (go-git, static linking)
- **Schema Clarity**: YAML + validator for clear, validated configs
- **Plugin-Centric**: Clean plugin interface, statically linked for MVP
- **Safety**: Dry-run support, idempotency checks, fail-fast errors
- **Performance**: Goroutines for parallelism, <1s dry-run, <100ms startup
- **Extensibility**: Plugin architecture supports future connectors
- **Consistency**: Cobra/Viper patterns, zerolog structured logging

All technical decisions are documented, with alternatives considered and rationale provided. No critical unknowns remain that would block implementation.

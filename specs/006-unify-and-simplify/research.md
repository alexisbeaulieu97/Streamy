# Research: Unify and Simplify the Plugin System

**Date**: October 7, 2025  
**Feature**: Plugin Interface Refactoring

## Research Questions

### 1. Go Interface Design Best Practices for Plugin Systems

**Decision**: Single-responsibility methods with rich return types

**Rationale**: 
- Go interfaces should be small and focused (Rob Pike's "accept interfaces, return structs")
- Rich return types (structs) allow evolution without breaking the interface
- Separation of read (Evaluate) from write (Apply) aligns with Command-Query Separation (CQS) principle
- Single evaluation method eliminates need for mode parameters, which are error-prone

**Alternatives Considered**:
- **Mode parameter approach**: `Execute(ctx, step, mode ExecutionMode)` - Rejected because it pushes mode logic into plugins, defeating the purpose
- **Interface composition**: Separate `Evaluator` and `Applier` interfaces - Rejected as over-engineered for this use case
- **Keep existing 4-method interface**: Rejected as it's the problem we're solving

**References**:
- Go Proverbs: "The bigger the interface, the weaker the abstraction"
- Effective Go: Interface design patterns
- Command-Query Separation principle (Bertrand Meyer)

---

### 2. Read-Only Enforcement in Go

**Decision**: Convention + documentation + testing (no language-level enforcement)

**Rationale**:
- Go lacks const/immutable annotations for method receivers
- Best practice: Document contract clearly, enforce through comprehensive testing
- Integration tests will verify Evaluate() doesn't mutate state by checking filesystem/system state before/after
- Code review and linting can catch violations during development

**Alternatives Considered**:
- **Clone and compare**: Clone entire system state, call Evaluate(), verify no changes - Rejected as impractical and platform-specific
- **Read-only filesystem views**: Use OS-level read-only mounts - Rejected as complex and platform-dependent
- **Static analysis tool**: Custom linter to detect state mutations - Considered for future enhancement but not blocking

**Implementation**:
- Document "MUST be read-only" prominently in interface comments
- Add integration tests that verify state immutability
- Include read-only verification in plugin contract test suite

---

### 3. Error Type Hierarchy Design

**Decision**: Interface-based error types with Unwrap() support

**Rationale**:
- Go 1.13+ error wrapping with `errors.Unwrap()` and `errors.Is()`
- Interface approach allows type assertions and categorical handling
- Preserves full error chain for debugging while enabling structured handling in engine
- Aligns with existing `pkg/errors/errors.go` patterns in codebase

**Structure**:
```go
type PluginError interface {
    error
    StepID() string      // Which step failed
    Unwrap() error       // Underlying error
}

type ValidationError struct {
    ID  string
    Err error
}

type ExecutionError struct {
    ID  string
    Err error
}

type StateError struct {
    ID  string
    Err error
}
```

**Alternatives Considered**:
- **Simple error strings**: Rejected - insufficient structure for engine decision-making
- **Error codes only**: Rejected - loses rich error context and chain
- **Full exception hierarchy**: Rejected - not idiomatic Go

**Usage Pattern**:
```go
if err != nil {
    var execErr *ExecutionError
    if errors.As(err, &execErr) {
        // Handle execution failures (external commands, etc.)
    }
}
```

---

### 4. InternalData Field Design Pattern

**Decision**: `interface{}` (empty interface) with type assertion in Apply()

**Rationale**:
- Maximum flexibility for plugins to pass domain-specific data
- Type safety maintained through plugin implementation (same plugin creates and consumes)
- Avoids generic type complexity (Go generics not needed for this use case)
- Pattern used successfully in line_in_file plugin's existing evaluate() function

**Alternatives Considered**:
- **Generic type parameter**: `EvaluationResult[T any]` - Rejected as over-engineered; adds complexity to interface
- **JSON serialization**: `InternalData []byte` - Rejected as inefficient for in-memory data transfer
- **Predefined union type**: Multiple typed fields (InternalString, InternalBytes, etc.) - Rejected as rigid

**Safety Considerations**:
- Document that Apply() must handle nil InternalData gracefully
- Type assertion failures in Apply() should return clear errors
- Example pattern:
```go
func (p *myPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
    if evalResult.InternalData == nil {
        return nil, fmt.Errorf("missing evaluation data")
    }
    data, ok := evalResult.InternalData.(*myPluginData)
    if !ok {
        return nil, fmt.Errorf("invalid internal data type")
    }
    // Use data...
}
```

---

### 5. Migration Strategy for Existing Tests

**Decision**: Update tests incrementally per plugin, maintain test coverage throughout

**Rationale**:
- Each plugin has existing unit and integration tests that must continue passing
- Tests refactored in lock-step with plugin implementation
- Ensures no regression during migration
- Test-driven approach: update tests first, then make them pass

**Migration Pattern Per Plugin**:
1. **Identify existing test coverage**: Review `*_test.go` files for plugin
2. **Update test structure**: Replace calls to Check()/Apply()/DryRun()/Verify() with Evaluate()/Apply()
3. **Add read-only verification**: New test cases to verify Evaluate() doesn't mutate state
4. **Verify test failures**: Tests should fail initially (old implementation still present)
5. **Refactor plugin implementation**: Make tests pass with new interface
6. **Add diff verification**: Test that Diff field is populated correctly

**Alternatives Considered**:
- **Rewrite tests from scratch**: Rejected - loses valuable test coverage and edge case knowledge
- **Run both old and new tests in parallel**: Rejected - unnecessary complexity, confusing which tests are source of truth
- **Skip tests during migration**: Rejected - violates safety principles

**Test Coverage Requirements**:
- All existing test scenarios must pass
- New test: Evaluate() is read-only (no filesystem/state changes)
- New test: Evaluate() → Apply() flow works correctly
- New test: Diff field contains meaningful preview information
- New test: Structured errors are returned correctly

---

### 6. Performance Measurement Approach

**Decision**: Benchmark suite comparing old Check() vs new Evaluate()

**Rationale**:
- Need quantitative data to validate 20% overhead budget
- Go's built-in benchmarking framework (`testing.B`) is sufficient
- Measure per-plugin to identify any outliers
- Establish baseline before refactoring, compare after

**Benchmark Structure**:
```go
func BenchmarkSymlinkCheck(b *testing.B) {
    // Benchmark old Check() method
}

func BenchmarkSymlinkEvaluate(b *testing.B) {
    // Benchmark new Evaluate() method
}
```

**Metrics to Track**:
- Time per operation (ns/op)
- Memory allocations (allocs/op)
- Bytes allocated (B/op)

**Acceptance Criteria**:
- Evaluate() may be up to 20% slower than Check()
- Memory overhead should be proportional (for Diff/Message strings)
- No unexpected performance cliffs

**Detailed Measurement Methodology**:

**Baseline Establishment**:
- Run benchmarks on current `main` branch before any refactoring
- Capture `Check()` timing for all 8 plugins using representative test cases
- Record results in `research.md` for reference:
  ```
  Baseline (Check method on main branch, commit <hash>):
  - symlink: 120 ns/op, 0 allocs/op
  - copy: 2500 ns/op, 1 allocs/op
  - lineinfile: 15000 ns/op, 5 allocs/op
  - template: 25000 ns/op, 10 allocs/op
  - package: 50000 ns/op, 15 allocs/op
  - repo: 100000 ns/op, 20 allocs/op
  - command: 80000 ns/op, 12 allocs/op
  - internalexec: 60000 ns/op, 10 allocs/op
  ```

**Test Scenarios** (use existing testdata):
- Symlink: Check existing symlink pointing to correct target
- Copy: Compare identical 10KB text file
- Lineinfile: Search for existing line in 100-line file
- Template: Render template with 5 variables, compare to existing output
- Package: Query for installed package (vim or equivalent)
- Repo: Check git repo status (clean, on correct branch)
- Command: Execute simple check command (`echo "test"`)
- Internalexec: Execute internal command equivalent

**Measurement Conditions**:
- Run benchmarks on same hardware as baseline
- Use `go test -bench=. -benchmem -count=10` for statistical significance
- Discard first run (warmup), average remaining 9 runs
- Run during low system load (no concurrent builds/tests)

**Comparison Method**:
- Calculate per-plugin overhead: `((Evaluate_time - Check_time) / Check_time) * 100`
- Report both per-plugin and aggregate overhead
- Aggregate = weighted average based on typical usage frequency

**Budget Enforcement**:

| Severity | Overhead Range | Action |
|----------|----------------|--------|
| ✅ Pass | 0-20% | Merge approved |
| ⚠️ Warning | 20-30% | Requires justification + optimization plan |
| ❌ Fail | >30% | Must optimize before merge |

**Per-Plugin Exception**:
- Individual plugins may exceed 20% if aggregate stays within budget
- Example: If symlink is 25% slower but all others are <15%, aggregate may still pass

**Documentation Requirements**:
- Benchmark results committed to `specs/006-unify-and-simplify/benchmarks.md`
- Include comparison table, analysis, and explanation of any outliers
- If >20% overhead: document justification and mitigation strategy

**Regression Detection**:
- Add benchmark baseline to CI (future enhancement, not blocking)
- Manual re-run before major releases

**Alternatives Considered**:
- **Production profiling only**: Rejected - need controlled benchmarks for regression detection
- **External benchmarking tools**: Rejected - Go's built-in tools are sufficient
- **Skip performance measurement**: Rejected - violates stated performance goals

---

## Summary of Decisions

| Area | Decision | Key Rationale |
|------|----------|---------------|
| Interface Design | Single Evaluate/Apply methods | Simplicity, CQS principle |
| Read-Only Enforcement | Convention + testing | Go language limitations, practical approach |
| Error Hierarchy | Interface-based with Unwrap() | Structured handling, Go 1.13+ compatibility |
| InternalData | `interface{}` with type assertions | Flexibility without complexity |
| Test Migration | Incremental per-plugin updates | Maintain coverage, no regression |
| Performance | Benchmark suite, 20% budget | Quantitative validation of trade-offs |

## Next Steps

All research questions resolved. Ready to proceed to Phase 1 (Design & Contracts).

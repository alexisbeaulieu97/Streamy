# Executor-Plugin Interaction Contract

**Package**: `internal/engine`  
**Interacts With**: `internal/plugin`, `internal/model`

## Overview

This contract defines how the Executor (execution engine) interacts with plugins using the unified interface. The Executor is responsible for orchestrating the Evaluate → Apply lifecycle based on execution mode (verify, dry-run, apply).

## Execution Modes

### 1. Verify Mode (`streamy verify`)

**Purpose**: Check if system state matches desired state without making changes.

**Flow**:
```
For each step in DAG:
  1. Resolve plugin for step.Type
  2. Call plugin.Evaluate(ctx, step)
  3. Check EvaluationResult.CurrentState:
     - Satisfied → Log "✓ step satisfied"
     - Missing/Drifted → Log "✗ step drifted" + Message
     - Blocked → Log "⊘ step blocked" + Message
     - Unknown → Log "? step unknown" + error
  4. Continue to next step (never call Apply)
  
Return: Summary of satisfied vs drifted steps
```

**Engine Responsibilities**:
- Call Evaluate() for all steps regardless of result
- Log clear status for each step
- Generate summary report (X satisfied, Y drifted, Z blocked)
- Exit with code 0 if all satisfied, non-zero otherwise

**Never Calls**: `Apply()`

---

### 2. Dry-Run Mode (`streamy apply --dry-run`)

**Purpose**: Preview what changes would be made without applying them.

**Flow**:
```
For each step in DAG:
  1. Resolve plugin for step.Type
  2. Call plugin.Evaluate(ctx, step)
  3. Check EvaluationResult.RequiresAction:
     - false → Log "⊙ would skip: " + Message
     - true → Log "→ would apply: " + Message
              Display Diff if present
  4. Continue to next step (never call Apply)
  
Return: Summary of steps that would be skipped vs applied
```

**Engine Responsibilities**:
- Call Evaluate() for all steps
- Display Diff prominently when RequiresAction is true
- Use Message for summary line
- Count and report: X would skip, Y would apply
- Exit with code 0 (dry-run never fails)

**Never Calls**: `Apply()`

---

### 3. Apply Mode (`streamy apply`)

**Purpose**: Apply changes to bring system to desired state.

**Flow**:
```
For each step in DAG (respecting dependencies):
  1. Resolve plugin for step.Type
  2. Call plugin.Evaluate(ctx, step)
  3. Check EvaluationResult.RequiresAction:
     - false → Log "⊙ skipped: " + Message
              Record StepResult with Status=Skipped
     - true → Call plugin.Apply(ctx, evalResult, step)
              Log result based on StepResult.Status:
                Success → "✓ applied: " + Message
                Failed → "✗ failed: " + Message + Error
  4. If step failed and --continue-on-error not set:
       Stop execution, return error
  5. Continue to next step
  
Return: Summary of skipped, applied, failed steps
```

**Engine Responsibilities**:
- Call Evaluate() first, always
- Only call Apply() when RequiresAction is true
- Pass EvaluationResult to Apply() (includes InternalData)
- Handle errors according to --continue-on-error flag
- Log timing for each step
- Generate detailed execution report

**Critical Contract**: Apply() is ONLY called when Evaluate() reports RequiresAction=true

---

## Error Handling Contract

### Engine Behavior on PluginError

```go
evalResult, err := plugin.Evaluate(ctx, step)
if err != nil {
    var valErr *ValidationError
    var execErr *ExecutionError
    var stateErr *StateError
    
    switch {
    case errors.As(err, &valErr):
        // Configuration error - always fatal
        log.Error("Configuration validation failed for step %s", step.ID)
        log.Error("  %s", valErr.Error())
        log.Suggestion("Review step configuration in YAML")
        return err // Stop execution
        
    case errors.As(err, &execErr):
        // Execution error - depends on --continue-on-error
        log.Error("Execution failed for step %s", step.ID)
        log.Error("  %s", execErr.Error())
        if continueOnError {
            recordFailure(step)
            continue // Next step
        }
        return err // Stop execution
        
    case errors.As(err, &stateErr):
        // State detection error - treat as Unknown status
        log.Warn("Could not determine state for step %s", step.ID)
        log.Warn("  %s", stateErr.Error())
        recordUnknown(step)
        continue // Next step with warning
        
    default:
        // Unexpected error type - always fatal
        log.Error("Unexpected error for step %s", step.ID)
        log.Error("  %s", err.Error())
        return err // Stop execution
    }
}
```

### Error Reporting Requirements

- **ValidationError**: Always fatal, never continue
- **ExecutionError**: Fatal unless `--continue-on-error` flag set
- **StateError**: Warning level, mark step as Unknown, continue
- All errors include step ID and actionable message
- Errors are logged with appropriate severity
- Exit codes reflect highest severity error encountered

---

## Context Handling Contract

### Cancellation Propagation

```go
// Engine creates context with timeout/cancellation
ctx, cancel := context.WithTimeout(context.Background(), executionTimeout)
defer cancel()

// Pass context to all plugin calls
evalResult, err := plugin.Evaluate(ctx, step)
if err != nil {
    if errors.Is(err, context.Canceled) {
        log.Info("Execution canceled by user")
        return ErrCanceled
    }
    if errors.Is(err, context.DeadlineExceeded) {
        log.Error("Execution timeout after %v", executionTimeout)
        return ErrTimeout
    }
    // Handle other errors...
}
```

### Timeout Configuration

- **Default timeout**: 5 minutes per step
- **Configurable via**: `--timeout` flag or step-level `timeout` field
- **Engine responsibility**: Wrap each plugin call with timeout
- **Plugin responsibility**: Check ctx.Done() periodically in long operations

---

## Logging Contract

### Required Log Entries

Each step execution must log:

1. **Start**: `"[step-id] Starting: <step.Name>"`
2. **Evaluation**: `"[step-id] State: <CurrentState> (action: <RequiresAction>)"`
3. **Outcome**: `"[step-id] Result: <Status> (<duration>)"`
4. **Error** (if any): `"[step-id] Error: <error message>"`

### Log Levels

- **Info**: Normal execution flow, status updates
- **Warn**: State errors, unknown status, recoverable issues
- **Error**: Validation errors, execution failures, fatal errors
- **Debug**: Detailed evaluation results, internal data, timing breakdowns

### Structured Logging Format

```go
log.Info("Step completed",
    "step_id", step.ID,
    "step_name", step.Name,
    "status", result.Status,
    "duration_ms", duration.Milliseconds(),
    "requires_action", evalResult.RequiresAction,
)
```

---

## Performance Contract

### Executor Requirements

- **Parallel execution**: Steps with no dependencies run in parallel
- **Sequential for dependencies**: Dependent steps wait for dependencies
- **Evaluation caching**: Do not re-evaluate same step multiple times
- **Timeout enforcement**: Kill long-running plugins after timeout

### Timing Expectations

| Operation | Target | Maximum |
|-----------|--------|---------|
| plugin.Evaluate() | < 100ms | < 1s |
| plugin.Apply() | varies by plugin | configurable timeout |
| Total dry-run (50 steps) | < 1s | < 5s |
| Total apply (50 steps) | varies | configurable |

### Resource Limits

- **Memory**: Executor monitors total memory usage
- **Goroutines**: Limit concurrent plugin execution (default: NumCPU)
- **File descriptors**: Plugins must close resources properly

---

## Testing Contract

### Engine Tests Must Verify

1. **Evaluate-only modes**: Verify and dry-run never call Apply()
2. **Apply gating**: Apply() only called when RequiresAction is true
3. **Error handling**: Each error type handled correctly
4. **Context cancellation**: Graceful shutdown on cancel
5. **Timeout handling**: Steps killed after timeout
6. **Parallel execution**: Independent steps run concurrently
7. **Dependency ordering**: Dependent steps wait correctly

### Mock Plugin for Testing

```go
type MockPlugin struct {
    EvaluateFunc func(ctx context.Context, step *config.Step) (*model.EvaluationResult, error)
    ApplyFunc    func(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error)
}

func (m *MockPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
    if m.EvaluateFunc != nil {
        return m.EvaluateFunc(ctx, step)
    }
    return &model.EvaluationResult{
        StepID:         step.ID,
        CurrentState:   model.StatusSatisfied,
        RequiresAction: false,
        Message:        "mock satisfied",
    }, nil
}

func (m *MockPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
    if m.ApplyFunc != nil {
        return m.ApplyFunc(ctx, evalResult, step)
    }
    return &model.StepResult{
        StepID:  step.ID,
        Status:  model.StatusSuccess,
        Message: "mock applied",
    }, nil
}
```

---

## Summary

The Executor-Plugin contract ensures:

1. **Clear separation of concerns**: Engine handles orchestration, plugins handle domain logic
2. **Mode-specific behavior**: verify/dry-run never mutate state
3. **Evaluation-first**: Always evaluate before deciding to apply
4. **Structured errors**: Engine makes intelligent decisions based on error type
5. **Performance**: Parallel execution, timeouts, resource limits
6. **Observability**: Comprehensive logging of all operations

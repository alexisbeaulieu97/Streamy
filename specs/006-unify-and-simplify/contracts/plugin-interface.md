# Plugin Interface Contract

**Version**: 2.0.0 (Unified Interface)  
**Package**: `internal/plugin`

## Interface Definition

```go
package plugin

import (
    "context"
    
    "github.com/alexisbeaulieu97/streamy/internal/config"
    "github.com/alexisbeaulieu97/streamy/internal/model"
)

// Plugin is the unified interface all Streamy plugins must implement.
// This interface replaces the legacy 4-method interface (Check/Apply/DryRun/Verify)
// with a simpler 2-method approach centered on the Evaluate/Apply lifecycle.
type Plugin interface {
    // Metadata returns the plugin's identity and capability information.
    // This method must return consistent values across calls and should be
    // safe to call multiple times.
    Metadata() PluginMetadata

    // Schema returns a struct that defines the YAML configuration schema
    // for this plugin's steps. The returned struct should have JSON tags
    // for schema generation and validation.
    Schema() interface{}

    // Evaluate performs a STRICTLY READ-ONLY assessment of the system's
    // current state against the desired state defined in the step configuration.
    //
    // CRITICAL CONTRACT: This method MUST NOT mutate any system state.
    // It should only read current state and compute what changes (if any)
    // would be needed to reach the desired state.
    //
    // Returns:
    //   - EvaluationResult: Rich state assessment including current status,
    //     whether action is required, human-readable message, optional diff,
    //     and optional internal data to pass to Apply()
    //   - error: PluginError (ValidationError, ExecutionError, StateError)
    //     if evaluation cannot be completed
    Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error)

    // Apply mutates the system to match the desired state defined in the
    // step configuration. This method is ONLY called by the engine if
    // Evaluate() reported RequiresAction = true.
    //
    // The evalResult parameter contains the result from the previous
    // Evaluate() call, including InternalData that can be used to avoid
    // redundant computation.
    //
    // This method MUST be idempotent: calling it multiple times with the
    // same inputs should produce the same final state.
    //
    // Returns:
    //   - StepResult: Outcome of the apply operation (success, failure, etc.)
    //   - error: PluginError if the operation fails
    Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error)
}
```

## Contract Requirements

### 1. Metadata() Contract
- **Stability**: Must return consistent values across calls
- **Completeness**: All fields in PluginMetadata must be populated
- **Performance**: Should be fast (no I/O operations)

### 2. Schema() Contract
- **Type**: Must return a struct type with JSON tags
- **Validation**: Struct should define all valid configuration fields
- **Documentation**: Use struct tags for field documentation

### 3. Evaluate() Contract

#### CRITICAL: Read-Only Guarantee
- **MUST NOT** write to filesystem
- **MUST NOT** execute state-mutating commands
- **MUST NOT** modify databases or external systems
- **MUST NOT** create temporary files (use in-memory buffers)
- **MAY** read files, check system state, execute read-only queries

#### Read-Only Enforcement Mechanism

**Detection Method**: Contract test suite verifies read-only guarantee by:

1. **Filesystem State Capture**: Before calling Evaluate(), capture checksums of all files in relevant directories
2. **System State Snapshot**: Record relevant system state (symlink targets, file permissions, etc.)
3. **Evaluate() Execution**: Call plugin.Evaluate(ctx, step)
4. **State Verification**: Compare post-execution state to pre-execution state
5. **Assertion**: All checksums and state must match exactly (no modifications)

**Prohibited Operations** (comprehensive list):

- Writing to any file (includes temp files - use `bytes.Buffer` instead)
- Creating/deleting files or directories
- Modifying file permissions, ownership, or attributes
- Creating/modifying/deleting symlinks
- Executing shell commands that mutate state (use `--dry-run` flags where available)
- Modifying databases, caches, or persistent storage
- Network requests that trigger side effects (POST, PUT, DELETE, etc.)

**Permitted Operations** (read-only queries):

- Reading file contents (`os.ReadFile`, `os.Open` with read-only)
- Checking file existence, permissions, ownership (`os.Stat`, `os.Lstat`)
- Executing read-only commands (e.g., `git status`, `dpkg -l`, `brew list`)
- HTTP GET requests to idempotent endpoints
- In-memory computation (string manipulation, diff generation, etc.)

**Test Implementation** (contract_test.go):

```go
func TestEvaluate_is_read_only(t *testing.T, plugin Plugin, step *config.Step) {
    // Setup: Create test environment
    testDir := t.TempDir()
    setupTestFiles(t, testDir)
    
    // Capture state before evaluation
    beforeState := captureFilesystemState(t, testDir)
    beforeSymlinks := captureSymlinkState(t, testDir)
    
    // Execute Evaluate()
    _, err := plugin.Evaluate(context.Background(), step)
    if err != nil {
        t.Logf("Evaluate() returned error (acceptable): %v", err)
    }
    
    // Verify no state changes
    afterState := captureFilesystemState(t, testDir)
    afterSymlinks := captureSymlinkState(t, testDir)
    
    if !reflect.DeepEqual(beforeState, afterState) {
        t.Errorf("Evaluate() modified filesystem state")
        t.Logf("Before: %+v", beforeState)
        t.Logf("After: %+v", afterState)
    }
    
    if !reflect.DeepEqual(beforeSymlinks, afterSymlinks) {
        t.Errorf("Evaluate() modified symlink state")
    }
}
```

**Violation Handling**:
- Contract test failure blocks PR merge
- Error message includes: which files changed, before/after state, remediation guidance
- Plugin must be fixed to use in-memory buffers or read-only queries

**Limitations**:
- Cannot prevent all side effects (e.g., network requests with server-side effects)
- Relies on test environment coverage (test what you can detect)
- Plugin developers must understand and follow convention
- The primary automated enforcement focuses on filesystem mutations, as these are the most common and reliably detectable violations. Adherence to the broader principle (e.g., no state-mutating network calls) relies on code review and developer diligence.

#### Special Case: Command and InternalExec Plugins

**The Problem**: Command-based plugins (`command`, `internalexec`) execute user-provided shell commands or executables. The plugin cannot enforce whether these commands mutate state.

**Contract Relaxation**:

For `command` and `internalexec` plugins ONLY:
- Evaluate() MAY execute user-provided commands that could mutate state
- Plugin MUST document this limitation clearly in its Schema and documentation
- Plugin SHOULD provide separate `check_command` configuration field for read-only state checks

**Recommended Configuration Schema**:

```go
type CommandStep struct {
    // Check command: SHOULD be read-only (e.g., test, check, status)
    CheckCommand string `json:"check_command" yaml:"check_command"`
    
    // Apply command: Mutates state (e.g., install, configure, start)
    Command string `json:"command" yaml:"command"`
    
    // Exit code interpretation
    SuccessExitCodes []int `json:"success_exit_codes" yaml:"success_exit_codes"`
}
```

**Evaluate() Implementation Pattern**:

```go
func (p *commandPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
    cfg := step.Command
    
    // Use check_command if provided (developer's responsibility to make it read-only)
    cmdStr := cfg.CheckCommand
    if cmdStr == "" {
        // Fallback: use main command (may have side effects!)
        cmdStr = cfg.Command
    }
    
    // Execute command
    cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
    output, err := cmd.CombinedOutput()
    exitCode := cmd.ProcessState.ExitCode()
    
    // Interpret exit code
    satisfied := contains(cfg.SuccessExitCodes, exitCode)
    
    return &model.EvaluationResult{
        StepID:         step.ID,
        CurrentState:   statusFromExitCode(exitCode, satisfied),
        RequiresAction: !satisfied,
        Message:        fmt.Sprintf("Command exited with code %d: %s", exitCode, string(output)),
        InternalData:   nil, // Command will re-run in Apply()
    }, nil
}
```

**User Guidance** (to include in docs/plugins.md):

```yaml
# Good: Separate check and apply commands
- id: ensure-nginx-running
  type: command
  command:
    check_command: "systemctl is-active nginx"  # Read-only check
    command: "systemctl start nginx"             # State mutation
    success_exit_codes: [0]

# Acceptable: Check command with unavoidable side effects
- id: ensure-database-initialized
  type: command
  command:
    # Some operations combine check + idempotent apply
    command: "psql -c 'CREATE DATABASE IF NOT EXISTS mydb'"
    success_exit_codes: [0]

# Discouraged: Using apply command for check (re-runs expensive operation)
- id: download-file
  type: command
  command:
    command: "curl -o /tmp/file.txt https://example.com/file.txt"
    success_exit_codes: [0]
  # Problem: Evaluate() will download file, Apply() will download again
```

**Documentation Requirements**:
- Plugin README must have prominent warning about check command side effects
- Contract test for command plugin must acknowledge this limitation
- FR-013 in spec.md should note "except command/internalexec plugins"

**Why Not Enforce Read-Only?**:
- Cannot inspect arbitrary user commands for side effects (Turing halting problem)
- Users need flexibility for legacy scripts and external tools
- Idempotent commands (like `CREATE IF NOT EXISTS`) are inherently safe even if run in Evaluate()

#### Return Value Requirements
- **EvaluationResult.StepID**: Must match input step.ID
- **EvaluationResult.CurrentState**: Must accurately reflect state comparison
- **EvaluationResult.RequiresAction**: 
  - `true` for Missing or Drifted states
  - `false` for Satisfied, Blocked, or Unknown states
- **EvaluationResult.Message**: Must be non-empty, human-readable
- **EvaluationResult.Diff**: Should be populated when RequiresAction is true
- **EvaluationResult.InternalData**: Optional, for passing data to Apply()

#### Idempotency
- Calling Evaluate() multiple times in sequence must return equivalent results
- No observable side effects between calls

#### Context Handling
- Must respect context cancellation
- Should return context error if ctx.Done() channel closes

### 4. Apply() Contract

#### Idempotency Requirement
- Calling Apply() multiple times must be safe
- Final state should be identical regardless of call count
- Must handle "already satisfied" case gracefully

#### EvaluationResult Usage
- Should use evalResult.InternalData to avoid recomputation
- Must validate InternalData type before use (type assertion)
- Should fall back to recomputing if InternalData is nil or invalid

#### Error Handling
- Partial failures should be recoverable
- Must return PluginError with appropriate type
- Error messages must be actionable

#### Context Handling
- Must respect context cancellation
- Should return context error if ctx.Done() channel closes
- Long-running operations should periodically check context

## Error Contract

All errors returned by Evaluate() and Apply() should implement the PluginError interface:

```go
type PluginError interface {
    error
    StepID() string
    Unwrap() error
}
```

### Error Type Selection

Use **ValidationError** when:
- Step configuration is invalid
- Required fields are missing
- Field values violate validation rules

Use **ExecutionError** when:
- External command fails
- File I/O fails (permissions, not found, etc.)
- Network operation fails

Use **StateError** when:
- Cannot determine current state
- System state is inconsistent
- State detection requires unavailable resources

## Example Implementation

```go
type myPlugin struct{}

func (p *myPlugin) Metadata() PluginMetadata {
    return PluginMetadata{
        Name:        "my_plugin",
        Version:     "1.0.0",
        APIVersion:  "2.x",
        Description: "Example plugin",
        Stateful:    false,
    }
}

func (p *myPlugin) Schema() interface{} {
    return config.MyPluginStep{}
}

func (p *myPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
    // Parse configuration
    cfg := step.MyPlugin
    if cfg == nil {
        return nil, &ValidationError{
            ID:  step.ID,
            Err: fmt.Errorf("configuration missing"),
        }
    }

    // Read current state (READ-ONLY!)
    currentState, err := readCurrentState(cfg.Path)
    if err != nil {
        return nil, &ExecutionError{
            ID:  step.ID,
            Err: fmt.Errorf("failed to read state: %w", err),
        }
    }

    // Compare with desired state
    desiredState := computeDesiredState(cfg)
    
    if currentState.Equals(desiredState) {
        return &model.EvaluationResult{
            StepID:         step.ID,
            CurrentState:   model.StatusSatisfied,
            RequiresAction: false,
            Message:        "already in desired state",
        }, nil
    }

    // Generate diff
    diff := generateDiff(currentState, desiredState)

    return &model.EvaluationResult{
        StepID:         step.ID,
        CurrentState:   model.StatusDrifted,
        RequiresAction: true,
        Message:        "state differs from desired",
        Diff:           diff,
        InternalData:   desiredState, // Pass to Apply() to avoid recomputation
    }, nil
}

func (p *myPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
    cfg := step.MyPlugin

    // Use InternalData if available
    var desiredState *State
    if evalResult.InternalData != nil {
        if state, ok := evalResult.InternalData.(*State); ok {
            desiredState = state
        }
    }
    if desiredState == nil {
        // Fall back to recomputing
        desiredState = computeDesiredState(cfg)
    }

    // Mutate system to desired state
    if err := applyState(cfg.Path, desiredState); err != nil {
        return &model.StepResult{
            StepID:  step.ID,
            Status:  model.StatusFailed,
            Message: err.Error(),
            Error:   err,
        }, &ExecutionError{ID: step.ID, Err: err}
    }

    return &model.StepResult{
        StepID:  step.ID,
        Status:  model.StatusSuccess,
        Message: "state applied successfully",
    }, nil
}
```

## Verification Testing

All plugin implementations must pass this contract test suite:

```go
func TestPluginContract(t *testing.T, plugin Plugin) {
    t.Run("Metadata is stable", func(t *testing.T) {
        m1 := plugin.Metadata()
        m2 := plugin.Metadata()
        assert.Equal(t, m1, m2)
    })

    t.Run("Schema returns struct", func(t *testing.T) {
        schema := plugin.Schema()
        assert.NotNil(t, schema)
        // Verify it's a struct type
        rt := reflect.TypeOf(schema)
        assert.Equal(t, reflect.Struct, rt.Kind())
    })

    t.Run("Evaluate is read-only", func(t *testing.T) {
        step := createTestStep()
        snapshot := takeSystemSnapshot()
        
        _, err := plugin.Evaluate(context.Background(), step)
        
        assert.NoError(t, err)
        assert.Equal(t, snapshot, takeSystemSnapshot(), 
            "Evaluate() mutated system state")
    })

    t.Run("Evaluate is idempotent", func(t *testing.T) {
        step := createTestStep()
        
        result1, err1 := plugin.Evaluate(context.Background(), step)
        result2, err2 := plugin.Evaluate(context.Background(), step)
        
        assert.Equal(t, err1, err2)
        assert.Equal(t, result1.CurrentState, result2.CurrentState)
        assert.Equal(t, result1.RequiresAction, result2.RequiresAction)
    })

    t.Run("Apply is idempotent", func(t *testing.T) {
        step := createTestStep()
        evalResult, _ := plugin.Evaluate(context.Background(), step)
        
        result1, _ := plugin.Apply(context.Background(), evalResult, step)
        result2, _ := plugin.Apply(context.Background(), evalResult, step)
        
        assert.Equal(t, result1.Status, result2.Status)
    })

    t.Run("Error types are correct", func(t *testing.T) {
        invalidStep := createInvalidTestStep()
        
        _, err := plugin.Evaluate(context.Background(), invalidStep)
        
        var pluginErr PluginError
        assert.True(t, errors.As(err, &pluginErr))
        assert.Equal(t, invalidStep.ID, pluginErr.StepID())
    })
}
```

## Migration Checklist

When migrating a plugin from the old interface to this unified interface:

- [ ] Remove Check() method
- [ ] Remove DryRun() method  
- [ ] Remove Verify() method
- [ ] Rename or consolidate Metadata() if needed
- [ ] Implement Evaluate() with read-only constraint
- [ ] Implement Apply() using InternalData from Evaluate()
- [ ] Update error handling to use structured error types
- [ ] Update all unit tests
- [ ] Add contract test suite
- [ ] Add read-only verification test
- [ ] Update plugin documentation
- [ ] Verify performance is within 20% budget

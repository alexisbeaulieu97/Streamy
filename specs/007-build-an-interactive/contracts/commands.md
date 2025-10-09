# Contract: Tea.Cmd Patterns for Async Operations

**Phase**: 1 (Design & Contracts)  
**Date**: October 8, 2025  
**Purpose**: Define command constructors and patterns for async operations in the dashboard

## Overview

In Bubble Tea's Elm Architecture, all I/O and async work happens in `tea.Cmd` functions. These are functions that return `tea.Msg` and are executed by the Bubble Tea runtime in goroutines. This contract defines the patterns for creating commands for dashboard operations.

## Command Constructor Pattern

### Standard Command Structure

```go
// Command constructor returns a tea.Cmd
func operationCmd(params) tea.Cmd {
    return func() tea.Msg {
        // 1. Perform I/O or async work
        result, err := doWork(params)
        
        // 2. Convert result to message
        if err != nil {
            return OperationErrorMsg{Error: convertError(err)}
        }
        
        return OperationCompleteMsg{Result: result}
    }
}
```

**Rules**:
1. Command constructors are synchronous (return immediately)
2. The returned function runs in a goroutine (Bubble Tea manages this)
3. MUST return a message (never return nil from the function)
4. Errors are converted to error messages (never panic)
5. No direct model mutation (communicate via messages only)

## Core Command Constructors

### 1. verifyCmd

**Purpose**: Run verification for a single pipeline (FR-007)

**Signature**:
```go
func verifyCmd(pipelineID string, configPath string) tea.Cmd
```

**Implementation**:
```go
func verifyCmd(pipelineID string, configPath string) tea.Cmd {
    return func() tea.Msg {
        // Send started message first (optional optimization)
        // Note: This is synchronous, happens before goroutine spawns
        
        ctx := context.Background()
        result, err := engine.VerifyPipeline(ctx, configPath)
        
        if err != nil {
            return VerifyErrorMsg{
                PipelineID: pipelineID,
                Error: ErrorDetail{
                    Code:       "VERIFY_FAILED",
                    Message:    err.Error(),
                    Context:    fmt.Sprintf("Config: %s", configPath),
                    Suggestion: "Check config file syntax and step definitions",
                },
            }
        }
        
        return VerifyCompleteMsg{
            PipelineID: pipelineID,
            Result: ExecutionResult{
                PipelineID:  pipelineID,
                Operation:   "verify",
                Status:      determineStatus(result),
                Success:     result.AllPassed(),
                StepResults: result.Steps,
                Duration:    result.Duration,
                CompletedAt: time.Now(),
            },
        }
    }
}
```

**Error Handling**:
- Config file not found → `VERIFY_FAILED` with suggestion to check path
- Parse error → `VERIFY_FAILED` with line number context
- Step execution error → `VERIFY_FAILED` with failing step details

**Functional Requirements**: FR-007

---

### 2. applyCmd

**Purpose**: Run apply operation for a single pipeline (FR-008)

**Signature**:
```go
func applyCmd(pipelineID string, configPath string) tea.Cmd
```

**Implementation**:
```go
func applyCmd(pipelineID string, configPath string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        result, err := engine.ApplyPipeline(ctx, configPath)
        
        if err != nil {
            return ApplyErrorMsg{
                PipelineID: pipelineID,
                Error: ErrorDetail{
                    Code:       "APPLY_FAILED",
                    Message:    err.Error(),
                    Context:    fmt.Sprintf("Config: %s", configPath),
                    Suggestion: "Review error and consider manual intervention",
                },
            }
        }
        
        return ApplyCompleteMsg{
            PipelineID: pipelineID,
            Result: ExecutionResult{
                PipelineID:  pipelineID,
                Operation:   "apply",
                Status:      StatusSatisfied, // Apply success → satisfied
                Success:     true,
                StepResults: result.Steps,
                Duration:    result.Duration,
                CompletedAt: time.Now(),
            },
        }
    }
}
```

**Error Handling**:
- Permission denied → `APPLY_FAILED` with sudo suggestion
- Step failure → `APPLY_FAILED` with rollback information
- Partial completion → `APPLY_FAILED` with list of completed steps

**Functional Requirements**: FR-008

---

### 3. refreshAllCmd

**Purpose**: Verify all pipelines in parallel (FR-012)

**Signature**:
```go
func refreshAllCmd(pipelines []Pipeline) tea.Cmd
```

**Implementation**:
```go
func refreshAllCmd(pipelines []Pipeline) tea.Cmd {
    // Create individual verify commands for each pipeline
    cmds := make([]tea.Cmd, len(pipelines))
    for i, p := range pipelines {
        cmds[i] = verifyCmd(p.ID, p.Path)
    }
    
    // Batch them for parallel execution
    return tea.Sequence(
        // First, send started message
        func() tea.Msg {
            return RefreshStartedMsg{Count: len(pipelines)}
        },
        // Then run all verifications in parallel
        tea.Batch(cmds...),
    )
}
```

**Notes**:
- `tea.Batch()` executes commands concurrently
- Each command sends its own `VerifyCompleteMsg`
- Update() tracks progress via `RefreshProgressMsg`
- Order of completion is non-deterministic

**Functional Requirements**: FR-012

---

### 4. saveStatusCacheCmd

**Purpose**: Persist pipeline status to cache file (FR-017)

**Signature**:
```go
func saveStatusCacheCmd(pipelineID string, result ExecutionResult) tea.Cmd
```

**Implementation**:
```go
func saveStatusCacheCmd(pipelineID string, result ExecutionResult) tea.Cmd {
    return func() tea.Msg {
        cache, err := registry.LoadStatusCache()
        if err != nil {
            // Non-critical error, log but don't fail
            log.Warn().Err(err).Msg("Failed to load status cache")
            return nil // Or return a non-critical error message
        }
        
        cache.Set(pipelineID, registry.CachedStatus{
            Status:      result.Status,
            LastRun:     result.CompletedAt,
            Summary:     formatSummary(result),
            StepCount:   len(result.StepResults),
            FailedSteps: getFailedSteps(result),
        })
        
        if err := cache.Save(); err != nil {
            log.Warn().Err(err).Msg("Failed to save status cache")
        }
        
        return nil // No message needed, silent success
    }
}
```

**Error Handling**: Logs warnings but doesn't fail (cache is best-effort)

**Functional Requirements**: FR-017

---

### 5. loadDetailCmd

**Purpose**: Load detailed status for a pipeline (when transitioning to detail view)

**Signature**:
```go
func loadDetailCmd(pipelineID string) tea.Cmd
```

**Implementation**:
```go
func loadDetailCmd(pipelineID string) tea.Cmd {
    return func() tea.Msg {
        // Load cached status
        cache, err := registry.LoadStatusCache()
        if err != nil {
            return ErrorMsg{Error: ErrorDetail{
                Code:    "CACHE_LOAD_FAILED",
                Message: "Failed to load pipeline details",
            }}
        }
        
        status, exists := cache.Get(pipelineID)
        if !exists {
            // No cached status, trigger verification
            return VerifyStartedMsg{PipelineID: pipelineID}
        }
        
        return DetailLoadedMsg{
            PipelineID: pipelineID,
            Status:     status,
        }
    }
}
```

**Functional Requirements**: FR-006 (detail view display)

---

### 6. loadInitialStatusCmd

**Purpose**: Load statuses for all pipelines on dashboard startup

**Signature**:
```go
func loadInitialStatusCmd(pipelines []Pipeline) tea.Cmd
```

**Implementation**:
```go
func loadInitialStatusCmd(pipelines []Pipeline) tea.Cmd {
    return func() tea.Msg {
        cache, err := registry.LoadStatusCache()
        if err != nil {
            // Start with unknown statuses, don't fail startup
            return InitialStatusLoadedMsg{Statuses: map[string]PipelineStatus{}}
        }
        
        statuses := make(map[string]PipelineStatus)
        for _, p := range pipelines {
            if cached, exists := cache.Get(p.ID); exists {
                statuses[p.ID] = cached.Status
            } else {
                statuses[p.ID] = StatusUnknown
            }
        }
        
        return InitialStatusLoadedMsg{Statuses: statuses}
    }
}
```

**Functional Requirements**: FR-003 (display status on load)

---

### 7. cancelOperationCmd

**Purpose**: Cancel a running operation (FR-021, FR-022)

**Signature**:
```go
func cancelOperationCmd(pipelineID string, opType OperationType) tea.Cmd
```

**Implementation**:
```go
// Note: Requires context.Context support in verify/apply operations
func cancelOperationCmd(pipelineID string, opType OperationType) tea.Cmd {
    return func() tea.Msg {
        // Look up the operation's context
        ctx := operations[pipelineID].Context
        if ctx == nil {
            return ErrorMsg{Error: ErrorDetail{
                Code:    "CANCEL_FAILED",
                Message: "Operation cannot be cancelled",
            }}
        }
        
        // Cancel the context (goroutine will detect and abort)
        ctx.Cancel()
        
        return OperationCancelledMsg{
            PipelineID: pipelineID,
            OpType:     opType,
        }
    }
}
```

**Notes**:
- Requires verify/apply operations to accept `context.Context`
- Graceful cancellation depends on operation respecting context
- May leave partial state (documented in user messaging)

**Functional Requirements**: FR-021, FR-022

---

## Command Composition Patterns

### Sequential Commands

```go
// Execute commands in order (each waits for previous to complete)
tea.Sequence(
    verifyCmd(id, path),
    func() tea.Msg {
        // This runs after verify completes
        return ShowResultMsg{}
    },
)
```

**Use Case**: Verify → Auto-apply if successful

### Parallel Commands

```go
// Execute commands concurrently
tea.Batch(
    verifyCmd(id1, path1),
    verifyCmd(id2, path2),
    verifyCmd(id3, path3),
)
```

**Use Case**: Refresh all pipelines (FR-012)

### Conditional Commands

```go
// Return different commands based on state
func conditionalCmd(m DashboardModel) tea.Cmd {
    if m.autoRefresh {
        return refreshAllCmd(m.pipelines)
    }
    return nil
}
```

**Use Case**: Auto-refresh on startup if stale

### Delayed Commands

```go
// Execute after a delay
tea.Tick(5*time.Second, func(time.Time) tea.Msg {
    return RefreshMsg{}
})
```

**Use Case**: Auto-dismiss error messages

## Error Handling Patterns

### Pattern 1: Retry Command

```go
func retryableCmd(operation func() error, maxRetries int) tea.Cmd {
    return func() tea.Msg {
        var err error
        for i := 0; i < maxRetries; i++ {
            err = operation()
            if err == nil {
                return OperationCompleteMsg{}
            }
            time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
        }
        return OperationErrorMsg{Error: convertError(err)}
    }
}
```

**Use Case**: Network requests, transient failures

### Pattern 2: Fallback Command

```go
func commandWithFallback(primary, fallback tea.Cmd) tea.Cmd {
    return func() tea.Msg {
        // Try primary
        msg := primary()
        if errorMsg, ok := msg.(ErrorMsg); ok {
            // Primary failed, try fallback
            return fallback()
        }
        return msg
    }
}
```

**Use Case**: Load from cache, fallback to fresh verification

### Pattern 3: Timeout Command

```go
func commandWithTimeout(cmd tea.Cmd, timeout time.Duration) tea.Cmd {
    return func() tea.Msg {
        resultChan := make(chan tea.Msg, 1)
        
        go func() {
            resultChan <- cmd()
        }()
        
        select {
        case msg := <-resultChan:
            return msg
        case <-time.After(timeout):
            return ErrorMsg{Error: ErrorDetail{
                Code:    "TIMEOUT",
                Message: "Operation timed out",
            }}
        }
    }
}
```

**Use Case**: Long-running operations with timeout

## Testing Commands

### Unit Test Pattern

```go
func TestVerifyCmd(t *testing.T) {
    // Create command
    cmd := verifyCmd("test-id", "/tmp/test.yaml")
    
    // Execute synchronously (for testing)
    msg := cmd()
    
    // Assert message type and content
    completeMsg, ok := msg.(VerifyCompleteMsg)
    assert.True(t, ok)
    assert.Equal(t, "test-id", completeMsg.PipelineID)
    assert.Equal(t, StatusSatisfied, completeMsg.Result.Status)
}
```

### Integration Test Pattern

```go
func TestVerifyApplySequence(t *testing.T) {
    model := NewDashboardModel(testPipelines, testRegistry, testCache)
    
    // Simulate verify
    verifyMsg := VerifyCompleteMsg{
        PipelineID: "test-id",
        Result:     testResult,
    }
    model, cmd := model.Update(verifyMsg)
    
    // Execute returned command
    if cmd != nil {
        msg := cmd()
        model, _ = model.Update(msg)
    }
    
    // Assert final state
    assert.Equal(t, StatusSatisfied, model.pipelines[0].Status)
}
```

## Performance Considerations

### Batch Size Limits

```go
// Don't batch unlimited operations
func refreshAllCmdSafe(pipelines []Pipeline) tea.Cmd {
    const maxParallel = 10
    
    if len(pipelines) <= maxParallel {
        return refreshAllCmd(pipelines)
    }
    
    // Chunk into batches
    return tea.Sequence(
        refreshAllCmd(pipelines[:maxParallel]),
        func() tea.Msg {
            // After first batch completes, start next
            return ContinueRefreshMsg{Remaining: pipelines[maxParallel:]}
        },
    )
}
```

### Command Memoization

```go
// Cache command results for idempotent operations
var verifyCache = make(map[string]tea.Cmd)

func verifyCmdCached(pipelineID string, configPath string) tea.Cmd {
    key := pipelineID + ":" + configPath
    if cached, ok := verifyCache[key]; ok {
        return cached
    }
    
    cmd := verifyCmd(pipelineID, configPath)
    verifyCache[key] = cmd
    return cmd
}
```

**Warning**: Only cache truly idempotent operations

## Summary

Tea.Cmd patterns for the dashboard:

1. **All I/O in commands**: Never in Update() or View()
2. **Always return messages**: Even on error (no panics, no nil)
3. **Batch for parallelism**: Use `tea.Batch()` for concurrent operations
4. **Sequence for order**: Use `tea.Sequence()` for dependent operations
5. **Test synchronously**: Commands are just functions, easy to test
6. **Handle errors gracefully**: Convert to ErrorMsg, provide suggestions

These patterns ensure the dashboard remains responsive, testable, and aligned with Bubble Tea's architecture principles.

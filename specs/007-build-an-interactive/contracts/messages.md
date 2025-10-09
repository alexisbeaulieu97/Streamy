# Contract: Bubble Tea Messages

**Phase**: 1 (Design & Contracts)  
**Date**: October 8, 2025  
**Purpose**: Define all custom message types for dashboard state transitions

## Overview

This contract defines the message types used in the dashboard's Bubble Tea Update() function. Messages are the **only** way to trigger state changes in the Elm Architecture pattern. Each message type has a specific purpose and carries data needed for the state update.

## Message Categories

### 1. Navigation Messages

#### PipelineSelectedMsg

**Purpose**: User selected a pipeline from the list view (FR-005)

**Trigger**: 
- User presses Enter on highlighted pipeline
- User presses numeric key (1-9) corresponding to pipeline index

**Payload**:
```go
type PipelineSelectedMsg struct {
    Pipeline Pipeline // The selected pipeline
}
```

**Update Behavior**:
```go
case PipelineSelectedMsg:
    m.selectedID = msg.Pipeline.ID
    m.viewMode = ViewDetail
    // Load detailed status for selected pipeline
    return m, loadDetailCmd(msg.Pipeline.ID)
```

**Functional Requirements**: FR-005, FR-006

---

#### BackToListMsg

**Purpose**: Return from detail view to main list (FR-011)

**Trigger**:
- User presses Esc in detail view
- User presses 'q' in detail view

**Payload**:
```go
type BackToListMsg struct{}
```

**Update Behavior**:
```go
case BackToListMsg:
    m.viewMode = ViewList
    m.selectedID = ""
    // Clear detail-specific state
    return m, nil
```

**Functional Requirements**: FR-011

---

#### ScrollMsg

**Purpose**: User scrolled the pipeline list (FR-018)

**Trigger**:
- User presses up/down arrows beyond visible area
- Mouse wheel scroll (if terminal supports)

**Payload**:
```go
type ScrollMsg struct {
    Direction int // +1 for down, -1 for up
}
```

**Update Behavior**:
```go
case ScrollMsg:
    m.scrollOffset = clamp(m.scrollOffset + msg.Direction, 0, maxScroll)
    return m, nil
```

**Functional Requirements**: FR-018

---

### 2. Operation Messages

#### VerifyStartedMsg

**Purpose**: Verification operation initiated (FR-007)

**Trigger**: Internal message sent by `verifyCmd()` before async work begins

**Payload**:
```go
type VerifyStartedMsg struct {
    PipelineID string
}
```

**Update Behavior**:
```go
case VerifyStartedMsg:
    m.loading[msg.PipelineID] = true
    m.operations[msg.PipelineID] = Operation{
        Type:       OpVerify,
        PipelineID: msg.PipelineID,
        StartedAt:  time.Now(),
    }
    // Start spinner for this pipeline
    return m, m.spinner.Tick
```

**Functional Requirements**: FR-007, FR-009

---

#### VerifyCompleteMsg

**Purpose**: Verification operation completed successfully (FR-007, FR-010)

**Trigger**: Sent by `verifyCmd()` goroutine after verification finishes

**Payload**:
```go
type VerifyCompleteMsg struct {
    PipelineID string
    Result     ExecutionResult
}
```

**Update Behavior**:
```go
case VerifyCompleteMsg:
    // Update pipeline status
    for i := range m.pipelines {
        if m.pipelines[i].ID == msg.PipelineID {
            m.pipelines[i].Status = msg.Result.Status
            m.pipelines[i].LastRun = msg.Result.CompletedAt
            m.pipelines[i].LastResult = &msg.Result
            break
        }
    }
    
    // Clear loading state
    delete(m.loading, msg.PipelineID)
    delete(m.operations, msg.PipelineID)
    
    // Save status to cache
    return m, saveStatusCacheCmd(msg.PipelineID, msg.Result)
```

**Functional Requirements**: FR-007, FR-010, FR-017

---

#### VerifyErrorMsg

**Purpose**: Verification operation failed (FR-016)

**Trigger**: Sent by `verifyCmd()` if error occurs

**Payload**:
```go
type VerifyErrorMsg struct {
    PipelineID string
    Error      ErrorDetail
}
```

**Update Behavior**:
```go
case VerifyErrorMsg:
    // Update pipeline status to failed
    for i := range m.pipelines {
        if m.pipelines[i].ID == msg.PipelineID {
            m.pipelines[i].Status = StatusFailed
            break
        }
    }
    
    // Clear loading, set error
    delete(m.loading, msg.PipelineID)
    delete(m.operations, msg.PipelineID)
    m.errors[msg.PipelineID] = msg.Error.Message
    m.showError = true
    m.errorMsg = fmt.Sprintf("Verification failed: %s", msg.Error.Message)
    
    return m, nil
```

**Functional Requirements**: FR-016

---

#### ApplyStartedMsg

**Purpose**: Apply operation initiated (FR-008)

**Trigger**: Internal message sent by `applyCmd()` after confirmation

**Payload**:
```go
type ApplyStartedMsg struct {
    PipelineID string
}
```

**Update Behavior**:
```go
case ApplyStartedMsg:
    m.loading[msg.PipelineID] = true
    m.operations[msg.PipelineID] = Operation{
        Type:       OpApply,
        PipelineID: msg.PipelineID,
        StartedAt:  time.Now(),
    }
    // Start spinner
    return m, m.spinner.Tick
```

**Functional Requirements**: FR-008, FR-009

---

#### ApplyCompleteMsg

**Purpose**: Apply operation completed successfully (FR-008, FR-010)

**Trigger**: Sent by `applyCmd()` goroutine after apply finishes

**Payload**:
```go
type ApplyCompleteMsg struct {
    PipelineID string
    Result     ExecutionResult
}
```

**Update Behavior**:
```go
case ApplyCompleteMsg:
    // Update pipeline status (should be satisfied after successful apply)
    for i := range m.pipelines {
        if m.pipelines[i].ID == msg.PipelineID {
            m.pipelines[i].Status = msg.Result.Status
            m.pipelines[i].LastRun = msg.Result.CompletedAt
            m.pipelines[i].LastResult = &msg.Result
            break
        }
    }
    
    // Clear loading state
    delete(m.loading, msg.PipelineID)
    delete(m.operations, msg.PipelineID)
    
    // Save status and potentially auto-verify
    return m, tea.Batch(
        saveStatusCacheCmd(msg.PipelineID, msg.Result),
        verifyCmd(msg.PipelineID), // Re-verify after apply
    )
```

**Functional Requirements**: FR-008, FR-010, FR-017

---

#### ApplyErrorMsg

**Purpose**: Apply operation failed (FR-016)

**Trigger**: Sent by `applyCmd()` if error occurs

**Payload**:
```go
type ApplyErrorMsg struct {
    PipelineID string
    Error      ErrorDetail
}
```

**Update Behavior**: Similar to `VerifyErrorMsg`, update status to failed and show error

**Functional Requirements**: FR-016

---

#### RefreshStartedMsg

**Purpose**: Full status refresh initiated (FR-012)

**Trigger**: User presses 'r' key

**Payload**:
```go
type RefreshStartedMsg struct {
    Count int // Number of pipelines being refreshed
}
```

**Update Behavior**:
```go
case RefreshStartedMsg:
    m.refreshing = true
    m.refreshProgress = 0
    m.refreshTotal = msg.Count
    // Start spinner
    return m, m.spinner.Tick
```

**Functional Requirements**: FR-012

---

#### RefreshProgressMsg

**Purpose**: Report progress during multi-pipeline refresh

**Trigger**: Sent by each `verifyCmd()` during refresh operation

**Payload**:
```go
type RefreshProgressMsg struct {
    Completed int
    Total     int
}
```

**Update Behavior**:
```go
case RefreshProgressMsg:
    m.refreshProgress = msg.Completed
    // View will display "Refreshing... 5/10"
    return m, nil
```

**Functional Requirements**: FR-012

---

#### RefreshCompleteMsg

**Purpose**: All pipelines refreshed (FR-012)

**Trigger**: Sent after all `verifyCmd()` operations complete

**Payload**:
```go
type RefreshCompleteMsg struct {
    Results map[string]ExecutionResult // pipelineID -> result
}
```

**Update Behavior**:
```go
case RefreshCompleteMsg:
    // Update all pipeline statuses
    for id, result := range msg.Results {
        for i := range m.pipelines {
            if m.pipelines[i].ID == id {
                m.pipelines[i].Status = result.Status
                m.pipelines[i].LastRun = result.CompletedAt
                m.pipelines[i].LastResult = &result
                break
            }
        }
    }
    
    m.refreshing = false
    m.refreshProgress = 0
    m.refreshTotal = 0
    
    // Save all statuses to cache
    return m, saveAllStatusCmd(msg.Results)
```

**Functional Requirements**: FR-012, FR-017

---

### 3. Confirmation Messages

#### ConfirmActionMsg

**Purpose**: Prompt user for confirmation before destructive action (FR-008, FR-021)

**Trigger**: User presses 'a' to apply changes

**Payload**:
```go
type ConfirmActionMsg struct {
    Action     string // "apply", "cancel", etc.
    PipelineID string
    Message    string // Confirmation prompt text
}
```

**Update Behavior**:
```go
case ConfirmActionMsg:
    m.viewMode = ViewConfirm
    m.confirmAction = msg.Action
    m.confirmPipelineID = msg.PipelineID
    m.confirmMessage = msg.Message
    return m, nil
```

**Functional Requirements**: FR-008, FR-021

---

#### ConfirmResponseMsg

**Purpose**: User responded to confirmation prompt (FR-022)

**Trigger**: User presses 'y' (yes) or 'n' (no) in confirmation view

**Payload**:
```go
type ConfirmResponseMsg struct {
    Confirmed bool
}
```

**Update Behavior**:
```go
case ConfirmResponseMsg:
    m.viewMode = ViewDetail // Return to detail view
    
    if msg.Confirmed {
        // User confirmed, execute the action
        switch m.confirmAction {
        case "apply":
            return m, applyCmd(m.confirmPipelineID)
        case "cancel-verify":
            return m, cancelOperationCmd(m.confirmPipelineID)
        }
    }
    
    // User declined or action unknown
    return m, nil
```

**Functional Requirements**: FR-021, FR-022

---

### 4. Error Messages

#### ErrorMsg

**Purpose**: Display error banner (FR-016)

**Trigger**: Any operation encounters an error

**Payload**:
```go
type ErrorMsg struct {
    Error ErrorDetail
}
```

**Update Behavior**:
```go
case ErrorMsg:
    m.showError = true
    m.errorMsg = msg.Error.Message
    m.errorDetail = &msg.Error
    // Auto-dismiss after 5 seconds
    return m, tea.Tick(5*time.Second, func(time.Time) tea.Msg {
        return ClearErrorMsg{}
    })
```

**Functional Requirements**: FR-016

---

#### ClearErrorMsg

**Purpose**: Dismiss error banner

**Trigger**: 
- User presses Esc while error is displayed
- Auto-dismiss timeout expires

**Payload**:
```go
type ClearErrorMsg struct{}
```

**Update Behavior**:
```go
case ClearErrorMsg:
    m.showError = false
    m.errorMsg = ""
    m.errorDetail = nil
    return m, nil
```

**Functional Requirements**: FR-016

---

### 5. System Messages (Bubble Tea Built-ins)

#### tea.WindowSizeMsg

**Purpose**: Terminal size changed (FR-013)

**Trigger**: Terminal resize event

**Payload**: Built-in Bubble Tea type

**Update Behavior**:
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    
    // Recalculate layout dimensions
    m.list.SetSize(msg.Width, msg.Height - headerHeight - footerHeight)
    m.viewport.Width = msg.Width
    m.viewport.Height = msg.Height - headerHeight - footerHeight
    
    return m, nil
```

**Functional Requirements**: FR-013

---

#### tea.KeyMsg

**Purpose**: User pressed a key (FR-004, FR-005, FR-007, FR-008, FR-011, FR-012, FR-019)

**Trigger**: Keyboard input

**Payload**: Built-in Bubble Tea type

**Update Behavior**: Dispatches to key-specific handlers based on view mode

**Functional Requirements**: All keyboard interaction requirements

---

#### spinner.TickMsg

**Purpose**: Animate spinner during async operations (FR-009)

**Trigger**: Spinner component's internal ticker

**Payload**: Built-in Bubbles type

**Update Behavior**:
```go
case spinner.TickMsg:
    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd
```

**Functional Requirements**: FR-009

---

## Message Flow Patterns

### Pattern 1: Simple Navigation

```
User presses ↓
  → tea.KeyMsg (Type: KeyDown)
  → Update: m.cursor++
  → View: re-renders with new cursor position
```

### Pattern 2: Async Operation

```
User presses 'v' (verify)
  → tea.KeyMsg (Type: KeyRunes, Runes: ['v'])
  → Update: returns verifyCmd(pipelineID)
  → verifyCmd dispatches VerifyStartedMsg
  → Update: sets loading[id] = true
  → View: shows spinner for pipeline
  → (goroutine runs verification)
  → verifyCmd sends VerifyCompleteMsg
  → Update: updates status, clears loading
  → View: shows new status icon
```

### Pattern 3: Confirmation Dialog

```
User presses 'a' (apply)
  → tea.KeyMsg (Type: KeyRunes, Runes: ['a'])
  → Update: returns ConfirmActionMsg
  → Update: switches to ViewConfirm
  → View: renders "Apply changes? (y/n)"
  → User presses 'y'
  → tea.KeyMsg (Type: KeyRunes, Runes: ['y'])
  → Update: returns ConfirmResponseMsg{Confirmed: true}
  → Update: returns applyCmd(pipelineID)
  → (apply operation proceeds as async operation)
```

### Pattern 4: Parallel Refresh

```
User presses 'r' (refresh all)
  → tea.KeyMsg (Type: KeyRunes, Runes: ['r'])
  → Update: returns refreshAllCmd()
  → refreshAllCmd dispatches RefreshStartedMsg
  → refreshAllCmd returns tea.Batch([verifyCmd(id1), verifyCmd(id2), ...])
  → Multiple goroutines run concurrently
  → Each sends VerifyCompleteMsg independently
  → Update: handles each completion
  → View: updates progress "Refreshing... 3/10"
  → Last one sends RefreshCompleteMsg
  → Update: clears refresh state
  → View: shows updated list
```

## Message Ordering Guarantees

1. **Startup**: `tea.WindowSizeMsg` is always first message received
2. **Async Operations**: `*StartedMsg` is dispatched before async work begins
3. **Completion**: `*CompleteMsg` or `*ErrorMsg` is always sent after operation finishes
4. **No Guarantee**: Multiple async operations may complete in any order (refresh use case)

## Error Handling Contract

All async operations MUST send either:
- `*CompleteMsg` on success
- `*ErrorMsg` on failure

**Never** fail silently. If an operation can't complete, send `*ErrorMsg` with:
- `ErrorDetail.Code`: Machine-readable error category
- `ErrorDetail.Message`: User-friendly description
- `ErrorDetail.Suggestion`: Actionable next steps

Example:
```go
func verifyCmd(configPath string) tea.Cmd {
    return func() tea.Msg {
        result, err := engine.VerifyPipeline(context.Background(), configPath)
        if err != nil {
            return VerifyErrorMsg{
                PipelineID: pipelineID,
                Error: ErrorDetail{
                    Code:       "VERIFY_FAILED",
                    Message:    err.Error(),
                    Context:    fmt.Sprintf("Pipeline: %s", pipelineID),
                    Suggestion: "Check logs for details, or press 'v' to retry",
                },
            }
        }
        return VerifyCompleteMsg{
            PipelineID: pipelineID,
            Result:     *result,
        }
    }
}
```

## Testing Contract

Each message type MUST have:
1. **Unit Test**: Verify Update() handles message correctly
2. **State Transition Test**: Verify model state changes as expected
3. **Command Test**: Verify correct tea.Cmd is returned (if applicable)

Example test structure:
```go
func TestVerifyCompleteMsg(t *testing.T) {
    m := NewDashboardModel(testPipelines, testRegistry, testCache)
    m.loading["test-id"] = true
    
    msg := VerifyCompleteMsg{
        PipelineID: "test-id",
        Result: ExecutionResult{Status: StatusSatisfied},
    }
    
    newModel, cmd := m.Update(msg)
    
    assert.False(t, newModel.(DashboardModel).loading["test-id"])
    assert.Equal(t, StatusSatisfied, newModel.(DashboardModel).pipelines[0].Status)
    assert.NotNil(t, cmd) // Should return saveStatusCacheCmd
}
```

## Summary

This message contract defines all state transitions in the dashboard. Every user action, async operation result, and system event is represented as a message. The Update() function is a pure state machine that consumes messages and produces new state + commands.

**Key Principles**:
- Messages are immutable data structures
- Update() must return quickly (no I/O)
- Async work happens in tea.Cmd goroutines
- All operations send completion messages (success or error)
- No shared mutable state between goroutines

This contract ensures predictable, testable behavior throughout the dashboard lifecycle.

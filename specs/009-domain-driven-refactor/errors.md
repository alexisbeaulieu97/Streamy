# Error Contracts & Taxonomy

**Feature**: 009-domain-driven-refactor  
**Date**: 2025-10-15  
**Purpose**: Define canonical error types, wrapping policies, and layer responsibilities for error handling

---

## Overview

Streamy uses a layered error approach where each layer adds context while preserving the original error chain. Errors flow upward from Domain → Application → Infrastructure → CLI/TUI, with each layer enriching the error with layer-specific information.

---

## Error Taxonomy

### Domain Layer Error Codes

Domain errors represent business rule violations and invalid states. All domain errors use the `DomainError` type with typed error codes.

**Location**: `internal/domain/pipeline/errors.go`

```go
type ErrorCode string

const (
    // Validation errors (400-class)
    ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"      // Business rule violation
    ErrCodeDuplicate     ErrorCode = "DUPLICATE_ID"          // Duplicate step ID
    ErrCodeDependency    ErrorCode = "DEPENDENCY_ERROR"      // Invalid dependency reference
    ErrCodeCycle         ErrorCode = "CIRCULAR_DEPENDENCY"   // Circular dependency detected
    ErrCodeType          ErrorCode = "INVALID_TYPE"          // Invalid step/plugin type
    
    // Resource errors (404-class)
    ErrCodeNotFound      ErrorCode = "NOT_FOUND"             // Step, plugin, or config not found
    ErrCodeMissing       ErrorCode = "MISSING_REQUIRED"      // Required field missing
    
    // State errors (409-class)
    ErrCodeState         ErrorCode = "INVALID_STATE"         // Invalid entity state
    ErrCodeConflict      ErrorCode = "CONFLICT"              // State conflict (concurrent modification)
    
    // Operation errors (500-class)
    ErrCodeExecution     ErrorCode = "EXECUTION_ERROR"       // Step execution failure
    ErrCodePlugin        ErrorCode = "PLUGIN_ERROR"          // Plugin-specific failure
    ErrCodeTimeout       ErrorCode = "TIMEOUT"               // Operation timeout
    ErrCodeCancelled     ErrorCode = "CANCELLED"             // Context cancelled
    ErrCodeInternal      ErrorCode = "INTERNAL_ERROR"        // Unexpected internal error
)
```

**DomainError Structure**:

```go
type DomainError struct {
    Code    ErrorCode                 // Typed error code for pattern matching
    Message string                    // Human-readable error message
    Cause   error                     // Original error (if wrapping)
    Context map[string]interface{}    // Additional context (step ID, file path, etc.)
}

func (e *DomainError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
    return e.Cause
}
```

**Usage in Domain Layer**:

```go
// Validation error
if !isValidID(step.ID) {
    return &DomainError{
        Code:    ErrCodeValidation,
        Message: "step ID contains invalid characters",
        Context: map[string]interface{}{
            "step_id": step.ID,
            "pattern": "^[a-z0-9_-]+$",
        },
    }
}

// Dependency error
if _, err := p.GetStep(depID); err != nil {
    return &DomainError{
        Code:    ErrCodeDependency,
        Message: "step dependency not found",
        Cause:   err,
        Context: map[string]interface{}{
            "step_id":      step.ID,
            "dependency":   depID,
            "pipeline":     p.Name,
        },
    }
}
```

---

## Layer Responsibilities

### Domain Layer

**Responsibility**: Define business error types and validation failures

**Rules**:
- Return `*DomainError` for all business rule violations
- Include business context (step ID, pipeline name, validation rule)
- Never include infrastructure details (file paths, network addresses)
- Keep error messages focused on business concepts

**Error Types**:
- Validation errors (ErrCodeValidation, ErrCodeDuplicate, ErrCodeType)
- Dependency errors (ErrCodeDependency, ErrCodeCycle)
- Resource errors (ErrCodeNotFound, ErrCodeMissing)
- State errors (ErrCodeState, ErrCodeConflict)

**Example**:
```go
func (p *Pipeline) Validate() error {
    // Check for duplicate step IDs
    seen := make(map[string]bool)
    for _, step := range p.Steps {
        if seen[step.ID] {
            return &DomainError{
                Code:    ErrCodeDuplicate,
                Message: "duplicate step ID",
                Context: map[string]interface{}{
                    "step_id":  step.ID,
                    "pipeline": p.Name,
                },
            }
        }
        seen[step.ID] = true
    }
    return nil
}
```

---

### Application Layer

**Responsibility**: Add operational context and user guidance to errors

**Rules**:
- Wrap domain errors with `fmt.Errorf("%w", err)` to preserve error chain
- Add application-level context (use case name, operation attempted)
- Translate technical errors into user-actionable messages
- Aggregate multiple errors when appropriate (FR-009)

**Error Types**:
- Orchestration errors (use case failures)
- Aggregated errors (multiple step failures)
- Transaction errors (rollback scenarios)

**Wrapping Pattern**:
```go
func (u *ApplyUseCase) Apply(ctx context.Context, configPath string, dryRun bool) error {
    // Load configuration
    pip, err := u.loader.Load(ctx, configPath)
    if err != nil {
        // Add use case context
        return fmt.Errorf("failed to apply pipeline: %w. Check that config file exists and is valid YAML", err)
    }
    
    // Execute with error aggregation
    results, errs := u.executor.Execute(ctx, plan, dryRun)
    if len(errs) > 0 {
        // Aggregate multiple errors
        return &AggregateError{
            Message: fmt.Sprintf("pipeline execution failed with %d errors", len(errs)),
            Errors:  errs,
        }
    }
    
    return nil
}
```

**AggregateError** (for multi-step failures per FR-009):
```go
type AggregateError struct {
    Message string
    Errors  []error
}

func (e *AggregateError) Error() string {
    var buf strings.Builder
    buf.WriteString(e.Message)
    buf.WriteString(":\n")
    for i, err := range e.Errors {
        buf.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
    }
    return buf.String()
}

func (e *AggregateError) Unwrap() []error {
    return e.Errors
}
```

---

### Infrastructure Layer

**Responsibility**: Add technical context and translate infrastructure errors to domain errors

**Rules**:
- Wrap infrastructure errors (file I/O, network) with domain errors
- Add technical context (file paths, line numbers, system errors)
- Preserve original error in Cause field
- Map infrastructure failures to appropriate domain error codes

**Translation Examples**:

```go
// File I/O error → Domain error
func (l *YAMLLoader) Load(ctx context.Context, path string) (*pipeline.Pipeline, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, &pipeline.DomainError{
                Code:    pipeline.ErrCodeNotFound,
                Message: "configuration file not found",
                Cause:   err,
                Context: map[string]interface{}{
                    "file_path": path,
                },
            }
        }
        return nil, &pipeline.DomainError{
            Code:    pipeline.ErrCodeInternal,
            Message: "failed to read configuration file",
            Cause:   err,
            Context: map[string]interface{}{
                "file_path": path,
            },
        }
    }
    
    var pip pipeline.Pipeline
    if err := yaml.Unmarshal(data, &pip); err != nil {
        return nil, &pipeline.DomainError{
            Code:    pipeline.ErrCodeValidation,
            Message: "invalid YAML syntax",
            Cause:   err,
            Context: map[string]interface{}{
                "file_path": path,
                "error":     err.Error(),
            },
        }
    }
    
    return &pip, nil
}

// Plugin execution error → Domain error
func (e *Executor) executeStep(ctx context.Context, step *pipeline.Step) (*pipeline.StepResult, error) {
    plugin, err := e.registry.Get(step.Type)
    if err != nil {
        return nil, &pipeline.DomainError{
            Code:    pipeline.ErrCodeNotFound,
            Message: "plugin not found",
            Cause:   err,
            Context: map[string]interface{}{
                "step_id":     step.ID,
                "plugin_type": step.Type,
            },
        }
    }
    
    result, err := plugin.Apply(ctx, step)
    if err != nil {
        return nil, &pipeline.DomainError{
            Code:    pipeline.ErrCodePlugin,
            Message: "plugin execution failed",
            Cause:   err,
            Context: map[string]interface{}{
                "step_id":     step.ID,
                "plugin_type": step.Type,
                "step_name":   step.Name,
            },
        }
    }
    
    return result, nil
}
```

---

### CLI/TUI Layer

**Responsibility**: Format errors for user display and map to exit codes

**Rules**:
- Extract `DomainError` from error chain for structured display
- Show full error context in verbose mode
- Map error codes to exit codes
- Provide remediation suggestions

**Exit Code Mapping**:

```go
const (
    ExitSuccess          = 0   // Success
    ExitUsageError       = 1   // Invalid command usage
    ExitValidationError  = 2   // Config validation error (ErrCodeValidation, ErrCodeDuplicate, etc.)
    ExitNotFoundError    = 3   // Resource not found (ErrCodeNotFound, ErrCodeMissing)
    ExitExecutionError   = 4   // Execution failure (ErrCodeExecution, ErrCodePlugin)
    ExitTimeoutError     = 5   // Timeout (ErrCodeTimeout)
    ExitCancelledError   = 6   // User cancellation (ErrCodeCancelled)
    ExitInternalError    = 10  // Internal error (ErrCodeInternal)
)

func GetExitCode(err error) int {
    var domainErr *pipeline.DomainError
    if errors.As(err, &domainErr) {
        switch domainErr.Code {
        case pipeline.ErrCodeValidation, pipeline.ErrCodeDuplicate, 
             pipeline.ErrCodeDependency, pipeline.ErrCodeCycle, pipeline.ErrCodeType:
            return ExitValidationError
        case pipeline.ErrCodeNotFound, pipeline.ErrCodeMissing:
            return ExitNotFoundError
        case pipeline.ErrCodeExecution, pipeline.ErrCodePlugin:
            return ExitExecutionError
        case pipeline.ErrCodeTimeout:
            return ExitTimeoutError
        case pipeline.ErrCodeCancelled:
            return ExitCancelledError
        case pipeline.ErrCodeInternal:
            return ExitInternalError
        }
    }
    return ExitInternalError
}
```

**Error Formatting**:

```go
func FormatError(err error, verbose bool) string {
    var buf strings.Builder
    
    // Try to extract DomainError
    var domainErr *pipeline.DomainError
    if errors.As(err, &domainErr) {
        buf.WriteString(fmt.Sprintf("Error [%s]: %s\n", domainErr.Code, domainErr.Message))
        
        // Show context
        if len(domainErr.Context) > 0 {
            buf.WriteString("\nContext:\n")
            for k, v := range domainErr.Context {
                buf.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
            }
        }
        
        // Show remediation
        if suggestion := GetRemediationSuggestion(domainErr.Code); suggestion != "" {
            buf.WriteString(fmt.Sprintf("\nSuggestion: %s\n", suggestion))
        }
        
        // Show full chain in verbose mode
        if verbose && domainErr.Cause != nil {
            buf.WriteString("\nCaused by:\n")
            buf.WriteString(fmt.Sprintf("  %v\n", domainErr.Cause))
        }
    } else {
        // Generic error formatting
        buf.WriteString(fmt.Sprintf("Error: %v\n", err))
    }
    
    return buf.String()
}

func GetRemediationSuggestion(code pipeline.ErrorCode) string {
    switch code {
    case pipeline.ErrCodeValidation:
        return "Check your configuration file syntax and ensure all required fields are present"
    case pipeline.ErrCodeDependency:
        return "Verify that all step dependencies reference valid step IDs in your pipeline"
    case pipeline.ErrCodeCycle:
        return "Remove circular dependencies between steps"
    case pipeline.ErrCodeNotFound:
        return "Check that the file path is correct and the file exists"
    case pipeline.ErrCodePlugin:
        return "Check plugin configuration and ensure the plugin is properly installed"
    case pipeline.ErrCodeTimeout:
        return "Consider increasing the timeout value or checking for hanging operations"
    default:
        return ""
    }
}
```

---

## Error Flow Examples

### Example 1: Validation Error

```
Domain Layer (pipeline.Validate):
  → DomainError{Code: ErrCodeDuplicate, Message: "duplicate step ID", Context: {step_id: "install"}}

Application Layer (ApplyUseCase.Apply):
  → Wraps: "failed to apply pipeline: duplicate step ID. Check that config file has unique step IDs"

CLI Layer (cmd/streamy/apply.go):
  → Formats: "Error [DUPLICATE_ID]: duplicate step ID
               Context:
                 step_id: install
               Suggestion: Check your configuration file and ensure all step IDs are unique"
  → Exit Code: 2 (ExitValidationError)
```

### Example 2: Plugin Execution Error

```
Infrastructure Layer (Executor.executeStep):
  → DomainError{Code: ErrCodePlugin, Message: "plugin execution failed", 
      Cause: exec.Error{}, Context: {step_id: "build", plugin_type: "command"}}

Application Layer (ApplyUseCase.Apply):
  → AggregateError with multiple step failures (per FR-009)

CLI Layer:
  → Formats: "Error: Pipeline execution failed with 2 errors:
               1. [PLUGIN_ERROR] plugin execution failed (step: build)
               2. [PLUGIN_ERROR] plugin execution failed (step: test)
             Suggestion: Check plugin configurations and ensure commands are valid"
  → Exit Code: 4 (ExitExecutionError)
```

### Example 3: Context Cancellation

```
Infrastructure Layer (Executor.Execute):
  → Checks ctx.Done(), returns DomainError{Code: ErrCodeCancelled, Message: "execution cancelled"}

Application Layer:
  → Wraps: "pipeline execution cancelled by user"

CLI Layer:
  → Formats: "Execution cancelled"
  → Exit Code: 6 (ExitCancelledError)
```

---

## Testing Error Handling

### Domain Layer Tests

Test that entities return correct `DomainError` types:

```go
func TestPipeline_Validate_DuplicateIDs(t *testing.T) {
    pip := &Pipeline{
        Steps: []Step{
            {ID: "step1"},
            {ID: "step1"}, // Duplicate
        },
    }
    
    err := pip.Validate()
    
    var domainErr *DomainError
    if !errors.As(err, &domainErr) {
        t.Fatal("expected DomainError")
    }
    if domainErr.Code != ErrCodeDuplicate {
        t.Errorf("expected ErrCodeDuplicate, got %s", domainErr.Code)
    }
}
```

### Application Layer Tests

Test error wrapping and aggregation:

```go
func TestApplyUseCase_MultipleErrors(t *testing.T) {
    mockExecutor := &MockExecutor{
        Errors: []error{
            &pipeline.DomainError{Code: pipeline.ErrCodePlugin},
            &pipeline.DomainError{Code: pipeline.ErrCodePlugin},
        },
    }
    
    useCase := NewApplyUseCase(/* ... */, mockExecutor, /* ... */)
    err := useCase.Apply(context.Background(), "config.yaml", false)
    
    var aggErr *AggregateError
    if !errors.As(err, &aggErr) {
        t.Fatal("expected AggregateError")
    }
    if len(aggErr.Errors) != 2 {
        t.Errorf("expected 2 errors, got %d", len(aggErr.Errors))
    }
}
```

### Infrastructure Layer Tests

Test error translation:

```go
func TestYAMLLoader_Load_FileNotFound(t *testing.T) {
    loader := NewYAMLLoader(/* ... */)
    
    _, err := loader.Load(context.Background(), "/nonexistent/file.yaml")
    
    var domainErr *pipeline.DomainError
    if !errors.As(err, &domainErr) {
        t.Fatal("expected DomainError")
    }
    if domainErr.Code != pipeline.ErrCodeNotFound {
        t.Errorf("expected ErrCodeNotFound, got %s", domainErr.Code)
    }
    if !errors.Is(err, os.ErrNotExist) {
        t.Error("expected to wrap os.ErrNotExist")
    }
}
```

---

## Implementation Checklist

- [ ] Define `ErrorCode` constants in `internal/domain/pipeline/errors.go`
- [ ] Implement `DomainError` struct with `Error()` and `Unwrap()` methods
- [ ] Implement `AggregateError` in `internal/application/pipeline/errors.go`
- [ ] Implement exit code mapping in `cmd/streamy/errors.go`
- [ ] Implement `FormatError()` and `GetRemediationSuggestion()` in `cmd/streamy/errors.go`
- [ ] Update all domain entities to return `DomainError` for validation failures
- [ ] Update all application use cases to wrap errors with context
- [ ] Update all infrastructure adapters to translate errors to domain errors
- [ ] Update all CLI commands to use `FormatError()` and `GetExitCode()`
- [ ] Add error handling tests for each layer
- [ ] Document error codes in API documentation
- [ ] Add error handling examples to `quickstart.md`

---

## Related Documents

- **Architecture Overview**: `docs/architecture-overview.md` - Error handling section
- **Quickstart Guide**: `specs/009-domain-driven-refactor/quickstart.md` - Error wrapping patterns
- **Specification**: `specs/009-domain-driven-refactor/spec.md` - FR-027, FR-028, US6
- **Data Model**: `specs/009-domain-driven-refactor/data-model.md` - DomainError entity

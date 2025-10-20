# Data Model: Domain-Driven Architecture Refactor

**Feature**: 009-domain-driven-refactor  
**Date**: 2025-10-15  
**Purpose**: Define all domain entities, port interfaces, and their relationships for the refactored architecture

---

## Domain Entities

### 1. Pipeline (Aggregate Root)

**Purpose**: Represents a complete pipeline configuration with metadata, steps, validations, and settings.

**Fields**:
- `Version` (string): Schema version (e.g., "1.0")
- `Name` (string): Human-readable pipeline name
- `Description` (string, optional): Pipeline description
- `Settings` (Settings value object): Global execution parameters
- `Steps` ([]Step): Ordered list of steps to execute
- `Validations` ([]Validation): Post-execution validation checks

**Business Rules**:
- Version must be valid semver format
- Name is required and 1-100 characters
- Step IDs must be unique within pipeline
- Step dependencies must reference existing step IDs (no dangling references)
- No circular dependencies between steps (validated via DAG)

**Methods**:
```go
// Validate checks all pipeline invariants
func (p *Pipeline) Validate() error

// GetStep retrieves a step by ID
func (p *Pipeline) GetStep(id string) (*Step, error)

// ValidateDependencies ensures all step dependencies exist
func (p *Pipeline) ValidateDependencies() error
```

**State Transitions**: Immutable after creation (validated on construction)

**Go Struct**:
```go
type Pipeline struct {
    Version     string
    Name        string
    Description string
    Settings    Settings
    Steps       []Step
    Validations []Validation
}
```

---

### 2. Step (Entity)

**Purpose**: Represents a single unit of work with unique identity, type, configuration, and dependencies.

**Fields**:
- `ID` (string): Unique identifier within pipeline
- `Name` (string, optional): Human-readable step name
- `Type` (string): Step type (package, repo, symlink, copy, command, template, lineinfile)
- `DependsOn` ([]string): List of step IDs this step depends on
- `Enabled` (bool): Whether step is active (default: true)
- `VerifyTimeout` (int, optional): Timeout in seconds for verification (default: from settings)
- `Config` (map[string]interface{}): Type-specific configuration data

**Business Rules**:
- ID must match pattern: `^[a-z0-9_-]+$`
- Type must be one of: package, repo, symlink, copy, command, template, lineinfile
- DependsOn references must exist in pipeline
- Config must match schema for step type (validated by plugin)

**Methods**:
```go
// Validate checks step invariants
func (s *Step) Validate() error

// GetConfigValue retrieves a typed config value
func (s *Step) GetConfigValue(key string) (interface{}, error)

// HasDependency checks if step depends on another
func (s *Step) HasDependency(stepID string) bool
```

**State Transitions**: Immutable after creation

**Go Struct**:
```go
type Step struct {
    ID            string
    Name          string
    Type          string
    DependsOn     []string
    Enabled       bool
    VerifyTimeout int
    Config        map[string]interface{}
}
```

---

### 3. ExecutionPlan (Value Object)

**Purpose**: Represents a DAG-based execution plan with level-based grouping for parallel execution.

**Fields**:
- `Levels` ([]ExecutionLevel): Ordered levels of execution
- `TotalSteps` (int): Total number of steps in plan
- `EstimatedDuration` (time.Duration): Estimated execution time

**Business Rules**:
- Steps in same level have no dependencies on each other (can run in parallel)
- Steps in level N may only depend on steps in levels < N
- All dependencies must be satisfied before level execution

**Methods**:
```go
// GetLevel retrieves execution level by index
func (p *ExecutionPlan) GetLevel(index int) (*ExecutionLevel, error)

// GetStepLevel finds which level a step ID is in
func (p *ExecutionPlan) GetStepLevel(stepID string) (int, error)

// Validate checks plan invariants
func (p *ExecutionPlan) Validate() error
```

**State Transitions**: Immutable after creation (computed from DAG)

**Go Struct**:
```go
type ExecutionPlan struct {
    Levels            []ExecutionLevel
    TotalSteps        int
    EstimatedDuration time.Duration
}

type ExecutionLevel struct {
    Level   int
    StepIDs []string
}
```

---

### 4. StepResult (Value Object)

**Purpose**: Captures the outcome of step execution including status, duration, output, and errors.

**Fields**:
- `StepID` (string): Step identifier
- `Status` (ResultStatus): Success, Failure, Skipped, AlreadySatisfied
- `Duration` (time.Duration): Execution time
- `StartTime` (time.Time): When step started
- `EndTime` (time.Time): When step completed
- `Message` (string): Human-readable outcome message
- `Output` (string, optional): Command output or plugin messages
- `Error` (error, optional): Error if step failed
- `Changed` (bool): Whether step made system changes
- `Diff` (string, optional): Description of changes made

**Business Rules**:
- Status must be one of the defined ResultStatus values
- Duration = EndTime - StartTime
- Error must be non-nil if Status is Failure
- Changed is false for Skipped and AlreadySatisfied statuses

**Methods**:
```go
// IsSuccess returns true if step succeeded
func (r *StepResult) IsSuccess() bool

// IsFailure returns true if step failed
func (r *StepResult) IsFailure() bool

// FormatOutput returns formatted output for display
func (r *StepResult) FormatOutput() string
```

**State Transitions**: Created once at step completion, immutable

**Go Struct**:
```go
type ResultStatus string

const (
    StatusSuccess            ResultStatus = "success"
    StatusFailure            ResultStatus = "failure"
    StatusSkipped            ResultStatus = "skipped"
    StatusAlreadySatisfied   ResultStatus = "already_satisfied"
)

type StepResult struct {
    StepID    string
    Status    ResultStatus
    Duration  time.Duration
    StartTime time.Time
    EndTime   time.Time
    Message   string
    Output    string
    Error     error
    Changed   bool
    Diff      string
}
```

---

### 5. VerificationResult (Value Object)

**Purpose**: Captures the outcome of post-execution validation checks.

**Fields**:
- `Type` (string): Validation type (command_exists, file_exists, path_contains)
- `Status` (VerificationStatus): Satisfied, Failed, Unknown
- `Message` (string): Human-readable result message
- `Details` (map[string]interface{}): Type-specific details

**Business Rules**:
- Type must match validation definition in pipeline
- Status reflects current system state
- Details contain diagnostic information for failures

**Methods**:
```go
// IsSatisfied returns true if validation passed
func (v *VerificationResult) IsSatisfied() bool

// FormatMessage returns formatted message for display
func (v *VerificationResult) FormatMessage() string
```

**State Transitions**: Created once per validation check, immutable

**Go Struct**:
```go
type VerificationStatus string

const (
    VerificationSatisfied VerificationStatus = "satisfied"
    VerificationFailed    VerificationStatus = "failed"
    VerificationUnknown   VerificationStatus = "unknown"
)

type VerificationResult struct {
    Type    string
    Status  VerificationStatus
    Message string
    Details map[string]interface{}
}

type VerificationSummary struct {
    TotalChecks   int
    PassedChecks  int
    FailedChecks  int
    Results       []VerificationResult
}
```

---

### 6. Plugin (Domain Interface)

**Purpose**: Contract that all step type implementations must satisfy.

**Methods**:
- `Metadata() PluginMetadata`: Returns plugin identity and capabilities
- `Schema() interface{}`: Returns configuration schema for validation
- `Evaluate(ctx context.Context, step *Step) (*EvaluationResult, error)`: Read-only state assessment
- `Apply(ctx context.Context, evalResult *EvaluationResult, step *Step) (*StepResult, error)`: Mutate system state

**Business Rules**:
- Evaluate must be read-only (no system mutations)
- Apply only called if Evaluate reports RequiresAction = true
- Both methods must respect context cancellation
- Must return structured errors that implement DomainError interface

**Go Interface**:
```go
type Plugin interface {
    Metadata() PluginMetadata
    Schema() interface{}
    Evaluate(ctx context.Context, step *Step) (*EvaluationResult, error)
    Apply(ctx context.Context, evalResult *EvaluationResult, step *Step) (*StepResult, error)
}
```

---

### 7. PluginMetadata (Value Object)

**Purpose**: Describes plugin identity, version, dependencies, and capabilities.

**Fields**:
- `ID` (string): Unique plugin identifier
- `Name` (string): Human-readable name
- `Version` (string): Plugin version (semver)
- `Type` (string): Step type this plugin handles
- `Description` (string): Plugin purpose description
- `Dependencies` ([]string): Other plugin IDs this depends on
- `APIVersion` (string): Plugin API version compatibility

**Business Rules**:
- ID must be unique across all plugins
- Version must be valid semver
- Type must be unique (one plugin per step type)
- APIVersion must match core API version

**Go Struct**:
```go
type PluginMetadata struct {
    ID           string
    Name         string
    Version      string
    Type         string
    Description  string
    Dependencies []string
    APIVersion   string
}
```

---

### 8. EvaluationResult (Value Object)

**Purpose**: Result of read-only state assessment by plugin.

**Fields**:
- `RequiresAction` (bool): Whether Apply needs to be called
- `CurrentState` (string): Description of current system state
- `DesiredState` (string): Description of desired state
- `Diff` (string, optional): What changes are needed
- `InternalData` (interface{}, optional): Plugin-specific data to pass to Apply

**Business Rules**:
- If RequiresAction is false, Apply will not be called
- Diff should describe what Apply would do
- InternalData can optimize Apply (avoid redundant computation)

**Go Struct**:
```go
type EvaluationResult struct {
    RequiresAction bool
    CurrentState   string
    DesiredState   string
    Diff           string
    InternalData   interface{}
}
```

---

### 9. Settings (Value Object)

**Purpose**: Global execution parameters for pipeline.

**Fields**:
- `Parallel` (int): Max concurrent steps (default: 4)
- `Timeout` (int): Default step timeout in seconds (default: 300)
- `ContinueOnError` (bool): Continue execution if step fails
- `DryRun` (bool): Preview mode, no system mutations
- `Verbose` (bool): Enable detailed logging

**Business Rules**:
- Parallel must be 1-32
- Timeout must be 1-360000 (100 hours max)

**Go Struct**:
```go
type Settings struct {
    Parallel        int
    Timeout         int
    ContinueOnError bool
    DryRun          bool
    Verbose         bool
}
```

---

### 10. Validation (Value Object)

**Purpose**: Post-execution validation check definition.

**Fields**:
- `Type` (string): command_exists, file_exists, path_contains
- `Config` (map[string]interface{}): Type-specific configuration

**Business Rules**:
- Type must be one of defined validation types
- Config must match schema for validation type

**Go Struct**:
```go
type Validation struct {
    Type   string
    Config map[string]interface{}
}
```

---

## Port Interfaces (Application Boundary)

**Location**: Port interfaces are defined at the application boundary (`internal/ports/`) to preserve a truly pure domain core. This follows the Dependency Inversion Principle: the application layer defines the contracts it needs, and the infrastructure layer implements them. The domain layer has zero knowledge of ports or infrastructure.

**Directory Structure**:
```
internal/ports/
├── config.go          # ConfigLoader
├── execution.go       # PluginExecutor, DAGBuilder, ExecutionPlanner
├── logging.go         # Logger
├── observability.go   # MetricsCollector, Tracer
├── plugins.go         # Plugin, PluginRegistry
├── events.go          # EventPublisher, EventHandler, DomainEvent
└── registry.go        # RegistryStore, ValidationService
```

### 11. ConfigLoader (Port)

**Purpose**: Load and parse pipeline configurations from external sources.

**Location**: `internal/ports/config.go`

**Methods**:
```go
type ConfigLoader interface {
    // Load reads and parses a pipeline configuration
    Load(ctx context.Context, path string) (*Pipeline, error)
    
    // Validate checks if a config file is valid without fully loading
    Validate(ctx context.Context, path string) error
}
```

**Implementations**: YAMLConfigLoader (infrastructure layer)

---

### 12. PluginExecutor (Port)

**Purpose**: Execute plugins against steps to mutate system state.

**Location**: `internal/ports/execution.go`

**Methods**:
```go
type PluginExecutor interface {
    // Execute runs the execution plan and returns step results
    Execute(ctx context.Context, plan *ExecutionPlan, pipeline *Pipeline) ([]StepResult, error)
    
    // Verify checks if steps are in desired state without mutating
    Verify(ctx context.Context, pipeline *Pipeline) ([]VerificationResult, error)
}
```

**Implementations**: DAGPluginExecutor (infrastructure layer)

---

### 13. Logger (Port)

**Purpose**: Structured logging with context propagation.

**Location**: `internal/ports/logging.go`

**Methods**:
```go
type Logger interface {
    // Debug logs debug-level message
    Debug(ctx context.Context, msg string, fields ...interface{})
    
    // Info logs informational message
    Info(ctx context.Context, msg string, fields ...interface{})
    
    // Warn logs warning message
    Warn(ctx context.Context, msg string, fields ...interface{})
    
    // Error logs error message
    Error(ctx context.Context, msg string, fields ...interface{})
    
    // With returns a child logger with additional fields
    With(fields ...interface{}) Logger
}
```

**Implementations**: CharmbraceletLogger (infrastructure layer)

---

### 14. MetricsCollector (Port)

**Purpose**: Record execution metrics for observability.

**Location**: `internal/ports/observability.go`

**Methods**:
```go
type MetricsCollector interface {
    // RecordStepDuration records how long a step took
    RecordStepDuration(stepID string, duration time.Duration)
    
    // RecordStepStatus records step outcome
    RecordStepStatus(stepID string, status ResultStatus)
    
    // RecordPipelineExecution records overall pipeline metrics
    RecordPipelineExecution(pipeline *Pipeline, duration time.Duration, success bool)
}
```

**Implementations**: NoOpMetricsCollector, PrometheusMetricsCollector (future)

---

### 15. RegistryStore (Port)

**Purpose**: Persist and retrieve pipeline registry entries.

**Location**: `internal/ports/registry.go`

**Methods**:
```go
type RegistryStore interface {
    // Store saves a pipeline registration
    Store(ctx context.Context, registration *Registration) error
    
    // Get retrieves a pipeline registration by ID
    Get(ctx context.Context, id string) (*Registration, error)
    
    // List retrieves all pipeline registrations
    List(ctx context.Context) ([]Registration, error)
    
    // Delete removes a pipeline registration
    Delete(ctx context.Context, id string) error
}
```

**Implementations**: FileRegistryStore (infrastructure layer)

---

### 16. ValidationService (Port)

**Purpose**: Run post-execution validation checks.

**Location**: `internal/ports/registry.go`

**Methods**:
```go
type ValidationService interface {
    // RunValidations executes all validations and returns results
    RunValidations(ctx context.Context, validations []Validation) (VerificationSummary, error)
}
```

**Implementations**: CommandValidationService (infrastructure layer)

---

## Domain Errors

### 17. DomainError (Interface)

**Purpose**: Typed errors with business context for error handling strategies.

**Types**:
```go
type ErrorCode string

const (
    ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"    // Invalid config or state
    ErrCodeExecution    ErrorCode = "EXECUTION_ERROR"     // Step execution failed
    ErrCodeDependency   ErrorCode = "DEPENDENCY_ERROR"    // Missing/circular deps
    ErrCodeNotFound     ErrorCode = "NOT_FOUND"           // Entity not found
    ErrCodeTimeout      ErrorCode = "TIMEOUT"             // Operation timed out
    ErrCodeCancelled    ErrorCode = "CANCELLED"           // Context cancelled
)

type DomainError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Context map[string]interface{} // Additional context (step ID, file path, etc.)
}

func (e *DomainError) Error() string
func (e *DomainError) Unwrap() error
func (e *DomainError) Is(target error) bool
```

**Usage Examples**:
```go
// Validation error
return &DomainError{
    Code:    ErrCodeValidation,
    Message: "step dependency not found",
    Context: map[string]interface{}{
        "step_id":      "install_docker",
        "missing_dep":  "update_packages",
    },
}

// Execution error
return &DomainError{
    Code:    ErrCodeExecution,
    Message: "plugin execution failed",
    Cause:   originalErr,
    Context: map[string]interface{}{
        "step_id":   "clone_repo",
        "plugin":    "repo",
    },
}
```

---

## Entity Relationships

```
Pipeline (Aggregate Root)
├── Settings (value object, 1:1)
├── Steps (entities, 1:N)
│   └── Config (map[string]interface{})
└── Validations (value objects, 1:N)

ExecutionPlan (value object)
├── Levels (value objects, 1:N)
│   └── StepIDs (references to Steps)

StepResult (value object)
├── StepID (reference to Step)
└── Error (optional DomainError)

VerificationSummary (value object)
└── Results (value objects, 1:N)

Plugin (interface)
├── Metadata (value object)
└── implements Evaluate, Apply methods

Port Interfaces (define boundaries)
├── ConfigLoader → returns Pipeline
├── PluginExecutor → accepts ExecutionPlan, returns StepResults
├── Logger → used by all layers
├── MetricsCollector → records metrics
├── RegistryStore → persists registrations
└── ValidationService → returns VerificationSummary
```

---

## Dependency Direction

```
Infrastructure Layer (adapters)
    ↓ implements
Application Layer (use cases)
    ↓ uses
Domain Layer (entities + ports)
    ↓ defines contracts
Port Interfaces (boundaries)
```

**Rules**:
- Domain layer has ZERO imports from application or infrastructure
- Application layer imports only domain (entities and ports)
- Infrastructure layer imports domain and application
- All dependencies flow inward toward domain

---

## Summary

**Domain Entities**: 10 entities (Pipeline, Step, ExecutionPlan, StepResult, VerificationResult, Plugin, PluginMetadata, EvaluationResult, Settings, Validation)

**Port Interfaces**: 6 ports (ConfigLoader, PluginExecutor, Logger, MetricsCollector, RegistryStore, ValidationService)

**Value Objects**: 8 immutable types (ExecutionPlan, ExecutionLevel, StepResult, VerificationResult, EvaluationResult, Settings, Validation, PluginMetadata)

**Domain Errors**: 6 error codes with structured error type

**Aggregate Root**: Pipeline (controls access to Steps and Validations)

All entities designed for:
- ✅ Testability (no infrastructure dependencies)
- ✅ Immutability (value objects)
- ✅ Clear boundaries (aggregate root pattern)
- ✅ Context propagation (all operations accept context.Context)
- ✅ Error handling (structured DomainError types)

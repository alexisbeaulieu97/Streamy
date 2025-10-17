// Port Interfaces for Application Layer Dependencies
// Package: internal/ports
//
// These interfaces define contracts for application-layer infrastructure dependencies
// (typically persistence, external services, validation).
//
// All port interfaces are defined at the application boundary (internal/ports/)
// rather than split across domain/application packages. This preserves a truly
// pure domain core with zero knowledge of infrastructure contracts.
//
// Port organization:
//   - config.go: ConfigLoader
//   - execution.go: PluginExecutor, DAGBuilder, ExecutionPlanner
//   - logging.go: Logger
//   - observability.go: MetricsCollector, Tracer
//   - plugins.go: Plugin, PluginRegistry
//   - events.go: EventPublisher, EventHandler, DomainEvent
//   - registry.go: RegistryStore, ValidationService (this file)

package ports

import (
	"context"
	"time"
)

// RegistryStore persists and retrieves pipeline registry entries.
//
// Implementations: internal/infrastructure/persistence/registry_store.go
//
// The registry maps human-readable IDs to pipeline configurations, allowing
// users to reference pipelines by name rather than file path.
//
// Example:
//
//	store := persistence.NewFileRegistryStore("/var/lib/streamy/registry")
//	err := store.Store(ctx, registration)
type RegistryStore interface {
	// Store saves a pipeline registration.
	//
	// If a registration with the same ID exists, it is replaced.
	//
	// Parameters:
	//   - ctx: Context with cancellation
	//   - registration: Pipeline registration to store
	//
	// Returns:
	//   - error: Domain error:
	//     - ErrCodeValidation: Invalid registration data
	//     - ErrCodeExecution: Storage operation failed (permissions, disk full, etc.)
	Store(ctx context.Context, registration *Registration) error

	// Get retrieves a pipeline registration by ID.
	//
	// Returns:
	//   - *Registration: Pipeline registration if found
	//   - error: Domain error:
	//     - ErrCodeNotFound: No registration with given ID
	Get(ctx context.Context, id string) (*Registration, error)

	// List retrieves all pipeline registrations.
	//
	// Returns registrations sorted by ID for consistent ordering.
	//
	// Returns:
	//   - []Registration: All registered pipelines
	//   - error: Domain error if list operation failed
	List(ctx context.Context) ([]Registration, error)

	// Delete removes a pipeline registration.
	//
	// Returns:
	//   - error: Domain error:
	//     - ErrCodeNotFound: No registration with given ID
	Delete(ctx context.Context, id string) error

	// UpdateStatus updates the execution status of a registered pipeline.
	//
	// This is called after each verify/apply operation to cache the last known state.
	//
	// Returns:
	//   - error: Domain error if update failed
	UpdateStatus(ctx context.Context, id string, status ExecutionStatus) error
}

// ValidationService runs post-execution validation checks.
//
// Implementations: internal/infrastructure/validation/service.go
//
// Validations verify system state after pipeline execution. Examples:
// - command_exists: Check if a command is in PATH
// - file_exists: Check if a file exists at a path
// - path_contains: Check if a directory contains expected files
//
// Example:
//
//	validator := validation.NewService(logger)
//	summary, err := validator.RunValidations(ctx, pipeline.Validations)
type ValidationService interface {
	// RunValidations executes all validations and returns aggregated results.
	//
	// Validations run concurrently where safe. Individual validation failures
	// do not stop other validations from running.
	//
	// Parameters:
	//   - ctx: Context with cancellation and timeout
	//   - validations: Validation definitions from pipeline
	//
	// Returns:
	//   - VerificationSummary: Aggregated results with pass/fail counts
	//   - error: Domain error only if validation execution itself failed
	//            (individual validation failures are recorded in summary)
	RunValidations(ctx context.Context, validations []Validation) (VerificationSummary, error)
}

// EventPublisher publishes domain events to interested subscribers.
//
// Implementations:
//   - internal/infrastructure/events/logger_publisher.go (logs events)
//   - internal/infrastructure/events/noop_publisher.go (discard events)
//
// Events provide observability into domain state transitions. Examples:
// - PipelineStarted
// - StepExecuted
// - PipelineCompleted
// - ValidationFailed
//
// Example:
//
//	publisher := events.NewLoggerPublisher(logger)
//	publisher.Publish(ctx, PipelineStartedEvent{Name: pipeline.Name})
type EventPublisher interface {
	// Publish sends an event to all registered subscribers.
	//
	// Events are published asynchronously - this method returns immediately.
	// If context is cancelled, event may not be delivered.
	//
	// Parameters:
	//   - ctx: Context with correlation ID
	//   - event: Event to publish (must implement DomainEvent interface)
	Publish(ctx context.Context, event DomainEvent)

	// Subscribe registers a handler for specific event types.
	//
	// Returns:
	//   - Subscription: Handle to unsubscribe
	Subscribe(eventType string, handler EventHandler) Subscription
}

// EventHandler processes published events.
type EventHandler func(ctx context.Context, event DomainEvent) error

// Subscription represents an active event subscription.
type Subscription interface {
	// Unsubscribe stops receiving events.
	Unsubscribe()
}

// DomainEvent represents a significant occurrence in the domain.
type DomainEvent interface {
	// EventType returns the event type identifier.
	EventType() string

	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time

	// Payload returns event-specific data.
	Payload() interface{}
}

// Registration represents a pipeline registry entry.
type Registration struct {
	ID                  string            // Unique identifier (user-provided)
	Name                string            // Human-readable name (from pipeline)
	ConfigPath          string            // Absolute path to YAML config
	RegisteredAt        time.Time         // When registered
	LastVerifiedAt      *time.Time        // When last verified (nil if never)
	LastExecutionStatus ExecutionStatus   // Cached status from last verify/apply
	Metadata            map[string]string // User-defined metadata (tags, environment, etc.)
}

// ExecutionStatus represents the last known state of a pipeline.
type ExecutionStatus struct {
	Status    RegistryStatus // satisfied, drifted, failed, unknown
	Message   string         // Status description
	Timestamp time.Time      // When status was determined
	Duration  time.Duration  // How long operation took
	Error     *string        // Error message if failed
}

// RegistryStatus indicates pipeline verification state.
type RegistryStatus string

const (
	StatusSatisfied RegistryStatus = "satisfied" // All steps in desired state
	StatusDrifted   RegistryStatus = "drifted"   // Some steps need changes
	StatusFailed    RegistryStatus = "failed"    // Verification/execution failed
	StatusUnknown   RegistryStatus = "unknown"   // Never verified or too old
)

// Validation represents a post-execution validation check.
// Defined in internal/domain/pipeline/validation.go
type Validation struct{}

// VerificationSummary aggregates validation results.
// Defined in internal/domain/pipeline/result.go
type VerificationSummary struct{}

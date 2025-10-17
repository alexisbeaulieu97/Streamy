package ports

import "context"

const (
	// EventPipelineStarted is emitted when a pipeline begins execution.
	EventPipelineStarted = "pipeline.started"
	// EventPipelineCompleted is emitted after a successful execution.
	EventPipelineCompleted = "pipeline.completed"
	// EventPipelineFailed is emitted when execution terminates with an error.
	EventPipelineFailed = "pipeline.failed"
	// EventStepStarted is emitted before a step begins execution.
	EventStepStarted = "step.started"
	// EventStepCompleted is emitted when a step finishes successfully.
	EventStepCompleted = "step.completed"
	// EventStepFailed is emitted when a step encounters an error.
	EventStepFailed = "step.failed"
	// EventStepSkipped is emitted when a step is skipped (disabled or already satisfied).
	EventStepSkipped = "step.skipped"
	// EventValidationStarted is emitted when post-execution validations begin.
	EventValidationStarted = "validation.started"
	// EventValidationCompleted is emitted after validations finish successfully.
	EventValidationCompleted = "validation.completed"
	// EventValidationFailed is emitted when any validation fails.
	EventValidationFailed = "validation.failed"
)

// DomainEvent represents a significant occurrence within the domain or
// application layer. Events carry structured payloads that downstream
// subscribers can use for logging, UI updates, or integrations.
type DomainEvent interface {
	EventType() string
	Payload() interface{}
}

// EventPublisher distributes events to interested subscribers. Dispatch is
// synchronous—Publish blocks until all handlers run—ensuring observability
// signals appear before the process exits. Handlers may spawn goroutines for
// async processing if work should continue in the background. Implementations
// must be thread-safe.
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
	Subscribe(eventType string, handler EventHandler) (Subscription, error)
}

// EventHandler processes an event of a specific type. Handlers should avoid
// panicking; failures should be surfaced via returned errors so publishers can
// log diagnostics and continue delivering to remaining subscribers.
type EventHandler func(context.Context, DomainEvent) error

// Subscription represents a registered handler. Callers must invoke
// Unsubscribe to stop receiving events and release resources.
type Subscription interface {
	Unsubscribe()
}

# Observability Contracts & Conventions

**Feature**: 009-domain-driven-refactor  
**Date**: 2025-10-15  
**Purpose**: Define logging, metrics, tracing conventions and port contracts for unified observability

---

## Overview

Streamy uses **charmbracelet/log** as the logging backbone with structured logging throughout all layers. Observability is achieved through three complementary mechanisms:

1. **Logging** - Structured log events with correlation IDs and context
2. **Metrics** - Counters, gauges, histograms for quantitative monitoring
3. **Tracing** - Distributed spans for request flow visualization

All three integrate via **correlation IDs** propagated through `context.Context`.

---

## Correlation IDs

### Purpose

Correlation IDs uniquely identify a pipeline execution and propagate through all layers, logs, metrics, and traces. This enables:
- Following a single execution across all log entries
- Correlating metrics with specific executions
- Linking distributed traces
- Debugging failures with complete context

### Implementation

**Context Key** (internal/infrastructure/logging/context.go):

```go
type correlationIDKey struct{}

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, correlationIDKey{}, id)
}

// GetCorrelationID retrieves correlation ID from context
func GetCorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
        return id
    }
    return ""
}

// GenerateCorrelationID creates a new UUID-based correlation ID
func GenerateCorrelationID() string {
    return uuid.New().String()
}
```

**CLI Entry Point** (cmd/streamy/main.go):

```go
func main() {
    // Generate correlation ID at entry point
    correlationID := logging.GenerateCorrelationID()
    ctx := logging.WithCorrelationID(context.Background(), correlationID)
    
    // All operations use this context
    if err := rootCmd.ExecuteContext(ctx); err != nil {
        // Error handling...
    }
}
```

### Naming Convention

- Format: UUID v4 (e.g., `550e8400-e29b-41d4-a716-446655440000`)
- Field name in logs: `correlation_id`
- HTTP header (future): `X-Correlation-ID`
- Metrics label: `correlation_id`

---

## Logging

### Logger Port Interface

**Location**: `internal/domain/pipeline/ports.go` → Move to `internal/ports/logging.go`

```go
type Logger interface {
    // Debug logs debug-level message with structured fields
    Debug(ctx context.Context, msg string, fields ...interface{})
    
    // Info logs info-level message with structured fields
    Info(ctx context.Context, msg string, fields ...interface{})
    
    // Warn logs warning-level message with structured fields
    Warn(ctx context.Context, msg string, fields ...interface{})
    
    // Error logs error-level message with structured fields
    Error(ctx context.Context, msg string, fields ...interface{})
    
    // With creates child logger with additional persistent fields
    With(fields ...interface{}) Logger
}
```

### Structured Logging Conventions

**Field Naming**:
- Use snake_case for field names: `step_id`, `plugin_type`, `execution_time`
- Common fields:
  - `correlation_id` - Correlation ID from context
  - `layer` - Which layer emitted the log ("domain", "application", "infrastructure")
  - `component` - Which component emitted the log ("pipeline", "executor", "loader")
  - `step_id` - Step identifier (when applicable)
  - `plugin_type` - Plugin type (when applicable)
  - `duration_ms` - Operation duration in milliseconds
  - `error` - Error message (when logging errors)

**Log Levels**:
- **Debug**: Detailed diagnostic information (off by default)
  - Example: "loading configuration file", "building DAG", "checking dependencies"
- **Info**: Important business events
  - Example: "pipeline started", "step completed successfully", "validation passed"
- **Warn**: Non-fatal issues that should be investigated
  - Example: "step already satisfied", "slow operation detected", "deprecated configuration"
- **Error**: Failures requiring attention
  - Example: "step execution failed", "plugin not found", "validation error"

**Example Usage**:

```go
// In application layer
logger.Info(ctx, "starting pipeline execution",
    "pipeline", pip.Name,
    "step_count", len(pip.Steps),
    "dry_run", dryRun,
)

// With child logger
stepLogger := logger.With("step_id", step.ID, "plugin_type", step.Type)
stepLogger.Info(ctx, "executing step")
stepLogger.Error(ctx, "step failed", "error", err, "duration_ms", duration.Milliseconds())
```

### Implementation

**Adapter** (internal/infrastructure/logging/logger.go):

```go
type Logger struct {
    impl *log.Logger  // charmbracelet/log
}

func NewLogger(level string) *Logger {
    l := log.NewWithOptions(os.Stderr, log.Options{
        Level: parseLevel(level),
    })
    return &Logger{impl: l}
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...interface{}) {
    // Extract correlation ID from context
    correlationID := GetCorrelationID(ctx)
    
    // Build structured log entry
    entry := l.impl.Info()
    if correlationID != "" {
        entry = entry.Str("correlation_id", correlationID)
    }
    
    // Add fields
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fmt.Sprint(fields[i])
            value := fields[i+1]
            entry = entry.Any(key, value)
        }
    }
    
    entry.Msg(msg)
}

func (l *Logger) With(fields ...interface{}) Logger {
    childImpl := l.impl.With()
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fmt.Sprint(fields[i])
            value := fields[i+1]
            childImpl = childImpl.Any(key, value)
        }
    }
    return &Logger{impl: childImpl.Logger()}
}
```

### Event Buffering During Initialization

**Problem**: Domain events may be emitted before logger is initialized.

**Solution** (per FR-012): In-memory event buffer with flush on logger availability.

**Implementation** (internal/infrastructure/logging/event_buffer.go):

```go
type EventBuffer struct {
    mu     sync.Mutex
    events []Event
    maxSize int
    flushed bool
}

type Event struct {
    Level   string
    Message string
    Fields  map[string]interface{}
    Time    time.Time
}

func NewEventBuffer(maxSize int) *EventBuffer {
    return &EventBuffer{
        events:  make([]Event, 0, maxSize),
        maxSize: maxSize,
    }
}

func (b *EventBuffer) Append(level, msg string, fields ...interface{}) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.flushed {
        return // Already flushed, ignore
    }
    
    if len(b.events) >= b.maxSize {
        // Drop oldest event if buffer full
        b.events = b.events[1:]
    }
    
    event := Event{
        Level:   level,
        Message: msg,
        Fields:  fieldsToMap(fields),
        Time:    time.Now(),
    }
    b.events = append(b.events, event)
}

func (b *EventBuffer) Flush(logger Logger) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.flushed {
        return
    }
    
    for _, event := range b.events {
        // Replay buffered events to real logger
        fields := mapToFields(event.Fields)
        switch event.Level {
        case "debug":
            logger.Debug(context.Background(), event.Message, fields...)
        case "info":
            logger.Info(context.Background(), event.Message, fields...)
        case "warn":
            logger.Warn(context.Background(), event.Message, fields...)
        case "error":
            logger.Error(context.Background(), event.Message, fields...)
        }
    }
    
    b.events = nil // Release memory
    b.flushed = true
}
```

**Usage in main.go**:

```go
// Before logger is ready
eventBuffer := logging.NewEventBuffer(1000)
tempLogger := logging.NewBufferedLogger(eventBuffer)

// Use tempLogger during initialization...

// Once real logger ready
logger := logging.NewLogger("info")
eventBuffer.Flush(logger)

// Now use real logger
```

---

## Metrics

### MetricsCollector Port Interface

**Location**: `internal/domain/pipeline/ports.go` → Move to `internal/ports/metrics.go`

```go
type MetricsCollector interface {
    // IncCounter increments a counter metric
    IncCounter(ctx context.Context, name string, labels map[string]string)
    
    // SetGauge sets a gauge metric value
    SetGauge(ctx context.Context, name string, value float64, labels map[string]string)
    
    // ObserveHistogram records a histogram observation
    ObserveHistogram(ctx context.Context, name string, value float64, labels map[string]string)
}
```

### Metric Naming Conventions

**Format**: `streamy_<component>_<metric>_<unit>`

**Standard Metrics**:

```
# Counters
streamy_pipeline_executions_total{status="success|failure|cancelled"}
streamy_step_executions_total{step_type="package|repo|...", status="success|failure|skipped"}
streamy_step_changes_total{step_type="..."}
streamy_validation_checks_total{check_type="...", status="pass|fail"}

# Gauges
streamy_pipeline_active_executions
streamy_step_parallel_executions

# Histograms
streamy_pipeline_execution_duration_seconds
streamy_step_execution_duration_seconds{step_type="..."}
streamy_plugin_evaluation_duration_seconds{plugin_type="..."}
streamy_validation_check_duration_seconds{check_type="..."}
```

**Label Conventions**:
- Always include `correlation_id` label when available
- Use consistent label names across metrics:
  - `step_type` for plugin types
  - `status` for outcomes
  - `check_type` for validation types

**Example Usage**:

```go
// In executor
start := time.Now()
result, err := plugin.Apply(ctx, step)
duration := time.Since(start)

labels := map[string]string{
    "step_type":      step.Type,
    "correlation_id": logging.GetCorrelationID(ctx),
}

if err != nil {
    labels["status"] = "failure"
    collector.IncCounter(ctx, "streamy_step_executions_total", labels)
} else {
    labels["status"] = "success"
    collector.IncCounter(ctx, "streamy_step_executions_total", labels)
    
    if result.Changed {
        collector.IncCounter(ctx, "streamy_step_changes_total", labels)
    }
}

collector.ObserveHistogram(ctx, "streamy_step_execution_duration_seconds", 
    duration.Seconds(), labels)
```

### Implementation

**Adapter** (internal/infrastructure/metrics/collector.go):

```go
type Collector struct {
    // Could use prometheus, statsd, or custom backend
    counters   map[string]*prometheus.CounterVec
    gauges     map[string]*prometheus.GaugeVec
    histograms map[string]*prometheus.HistogramVec
    mu         sync.RWMutex
}

func NewCollector() *Collector {
    return &Collector{
        counters:   make(map[string]*prometheus.CounterVec),
        gauges:     make(map[string]*prometheus.GaugeVec),
        histograms: make(map[string]*prometheus.HistogramVec),
    }
}

func (c *Collector) IncCounter(ctx context.Context, name string, labels map[string]string) {
    // Extract correlation ID from context if not in labels
    if _, ok := labels["correlation_id"]; !ok {
        if id := logging.GetCorrelationID(ctx); id != "" {
            labels = copyLabels(labels)
            labels["correlation_id"] = id
        }
    }
    
    // Get or create counter
    counter := c.getOrCreateCounter(name, labels)
    counter.With(prometheus.Labels(labels)).Inc()
}

// Similar for SetGauge, ObserveHistogram...
```

---

## Tracing

### Tracer Port Interface

**Location**: `internal/domain/pipeline/ports.go` → Move to `internal/ports/tracing.go`

```go
type Tracer interface {
    // StartSpan creates a new span
    StartSpan(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
    // SetAttribute adds metadata to span
    SetAttribute(key string, value interface{})
    
    // SetStatus sets span status
    SetStatus(status SpanStatus, message string)
    
    // End completes the span
    End()
}

type SpanStatus string

const (
    SpanStatusOK    SpanStatus = "ok"
    SpanStatusError SpanStatus = "error"
)
```

### Span Naming Conventions

**Format**: `<component>.<operation>`

**Standard Spans**:

```
# Application layer spans
pipeline.apply
pipeline.verify
pipeline.prepare

# Infrastructure layer spans
config.load
config.parse
dag.build
executor.execute
executor.execute_level
step.execute
plugin.evaluate
plugin.apply
validation.run
```

**Span Attributes**:
- Always include `correlation_id` attribute
- Component-specific attributes:
  - `pipeline.name`
  - `step.id`
  - `step.type`
  - `plugin.type`
  - `level` (for parallel execution levels)

**Example Usage**:

```go
// In ApplyUseCase
ctx, span := tracer.StartSpan(ctx, "pipeline.apply")
defer span.End()

span.SetAttribute("correlation_id", logging.GetCorrelationID(ctx))
span.SetAttribute("pipeline.name", pip.Name)
span.SetAttribute("step.count", len(pip.Steps))
span.SetAttribute("dry_run", dryRun)

// Execute pipeline...

if err != nil {
    span.SetStatus(SpanStatusError, err.Error())
    return err
}

span.SetStatus(SpanStatusOK, "pipeline applied successfully")
return nil
```

**Nested Spans**:

```go
// Parent span
ctx, pipelineSpan := tracer.StartSpan(ctx, "pipeline.apply")
defer pipelineSpan.End()

// Child span (shares correlation ID via context)
ctx, stepSpan := tracer.StartSpan(ctx, "step.execute")
stepSpan.SetAttribute("step.id", step.ID)
defer stepSpan.End()

// Execute step...
```

### Implementation

**Adapter** (internal/infrastructure/tracing/tracer.go):

```go
type Tracer struct {
    // Could use OpenTelemetry, Jaeger, or custom backend
    impl trace.Tracer
}

func NewTracer() *Tracer {
    // Initialize tracing backend
    return &Tracer{impl: otel.Tracer("streamy")}
}

func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
    // Extract correlation ID and add as attribute
    correlationID := logging.GetCorrelationID(ctx)
    
    ctx, otSpan := t.impl.Start(ctx, name)
    
    if correlationID != "" {
        otSpan.SetAttributes(attribute.String("correlation_id", correlationID))
    }
    
    return ctx, &span{impl: otSpan}
}

type span struct {
    impl trace.Span
}

func (s *span) SetAttribute(key string, value interface{}) {
    s.impl.SetAttributes(attribute.Any(key, value))
}

func (s *span) SetStatus(status SpanStatus, message string) {
    switch status {
    case SpanStatusOK:
        s.impl.SetStatus(codes.Ok, message)
    case SpanStatusError:
        s.impl.SetStatus(codes.Error, message)
    }
}

func (s *span) End() {
    s.impl.End()
}
```

---

## Integration Example

Complete example showing logging, metrics, and tracing together:

```go
// In ApplyUseCase.Apply
func (u *ApplyUseCase) Apply(ctx context.Context, configPath string, dryRun bool) error {
    // Start tracing span
    ctx, span := u.tracer.StartSpan(ctx, "pipeline.apply")
    defer span.End()
    
    correlationID := logging.GetCorrelationID(ctx)
    span.SetAttribute("correlation_id", correlationID)
    span.SetAttribute("config_path", configPath)
    span.SetAttribute("dry_run", dryRun)
    
    // Log start
    u.logger.Info(ctx, "starting pipeline execution",
        "config_path", configPath,
        "dry_run", dryRun,
    )
    
    // Increment execution counter
    u.metrics.IncCounter(ctx, "streamy_pipeline_executions_total", map[string]string{
        "status": "started",
    })
    
    start := time.Now()
    
    // Load configuration
    pip, err := u.loader.Load(ctx, configPath)
    if err != nil {
        u.logger.Error(ctx, "failed to load configuration", "error", err)
        span.SetStatus(SpanStatusError, "config load failed")
        u.metrics.IncCounter(ctx, "streamy_pipeline_executions_total", map[string]string{
            "status": "failure",
        })
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    span.SetAttribute("pipeline.name", pip.Name)
    span.SetAttribute("step.count", len(pip.Steps))
    
    // Execute pipeline...
    // (similar logging/metrics/tracing for each operation)
    
    duration := time.Since(start)
    
    // Log completion
    u.logger.Info(ctx, "pipeline execution completed",
        "pipeline", pip.Name,
        "duration_ms", duration.Milliseconds(),
        "dry_run", dryRun,
    )
    
    // Record metrics
    u.metrics.IncCounter(ctx, "streamy_pipeline_executions_total", map[string]string{
        "status": "success",
    })
    u.metrics.ObserveHistogram(ctx, "streamy_pipeline_execution_duration_seconds",
        duration.Seconds(), map[string]string{
            "pipeline": pip.Name,
        })
    
    span.SetStatus(SpanStatusOK, "pipeline applied successfully")
    return nil
}
```

---

## Testing Observability

### Logging Tests

Use `NoOpLogger` or `MockLogger` to verify logging behavior:

```go
func TestApplyUseCase_LogsExecution(t *testing.T) {
    mockLogger := &testutil.MockLogger{}
    useCase := NewApplyUseCase(/* ... */, mockLogger, /* ... */)
    
    err := useCase.Apply(context.Background(), "config.yaml", false)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    // Verify Info called with correct message
    if mockLogger.InfoCallCount != 2 { // Start + completion
        t.Errorf("expected 2 Info calls, got %d", mockLogger.InfoCallCount)
    }
    
    // Verify correlation ID present in logs
    if !mockLogger.HasField("correlation_id") {
        t.Error("expected correlation_id field in logs")
    }
}
```

### Metrics Tests

Use `NoOpCollector` or `MockCollector`:

```go
func TestApplyUseCase_RecordsMetrics(t *testing.T) {
    mockMetrics := &testutil.MockCollector{}
    useCase := NewApplyUseCase(/* ... */, mockMetrics)
    
    useCase.Apply(context.Background(), "config.yaml", false)
    
    // Verify counter incremented
    if !mockMetrics.CounterIncremented("streamy_pipeline_executions_total") {
        t.Error("expected pipeline execution counter incremented")
    }
    
    // Verify histogram recorded
    if !mockMetrics.HistogramObserved("streamy_pipeline_execution_duration_seconds") {
        t.Error("expected duration histogram observed")
    }
}
```

### Tracing Tests

Use `NoOpTracer` or `MockTracer`:

```go
func TestApplyUseCase_CreatesSpan(t *testing.T) {
    mockTracer := &testutil.MockTracer{}
    useCase := NewApplyUseCase(/* ... */, mockTracer)
    
    useCase.Apply(context.Background(), "config.yaml", false)
    
    // Verify span created
    if mockTracer.SpanCount != 1 {
        t.Errorf("expected 1 span, got %d", mockTracer.SpanCount)
    }
    
    // Verify span name
    if mockTracer.LastSpanName != "pipeline.apply" {
        t.Errorf("expected span name 'pipeline.apply', got %s", mockTracer.LastSpanName)
    }
    
    // Verify attributes
    if !mockTracer.SpanHasAttribute("correlation_id") {
        t.Error("expected correlation_id attribute")
    }
}
```

---

## Implementation Checklist

- [ ] Define Logger, MetricsCollector, Tracer port interfaces in `internal/ports/`
- [ ] Implement Logger adapter wrapping charmbracelet/log
- [ ] Implement correlation ID context helpers
- [ ] Implement event buffer for initialization phase
- [ ] Implement MetricsCollector adapter (Prometheus or no-op)
- [ ] Implement Tracer adapter (OpenTelemetry or no-op)
- [ ] Add observability to all use cases (logging, metrics, tracing)
- [ ] Add observability to all infrastructure adapters
- [ ] Create NoOp implementations for testing
- [ ] Create Mock implementations for testing
- [ ] Document standard metric names and span names
- [ ] Add observability examples to quickstart.md

---

## Event Publisher Architecture

### Purpose

EventPublisher provides a lightweight in-process event bus for domain events, enabling:
- Decoupling between event producers (domain/application) and consumers (observability, future workflows)
- Asynchronous event handling without blocking domain operations
- Observability into domain state transitions
- Future extensibility (webhooks, integrations, workflow triggers)

### EventPublisher Port Interface

**Location**: `internal/ports/events.go`

```go
type EventPublisher interface {
    // Publish sends an event to all subscribers
    Publish(ctx context.Context, event DomainEvent) error
    
    // Subscribe registers a handler for events of a specific type
    Subscribe(eventType string, handler EventHandler) Subscription
}

type DomainEvent interface {
    // Type returns the event type identifier
    Type() string
    
    // Payload returns event-specific data
    Payload() interface{}
    
    // Timestamp returns when event occurred
    Timestamp() time.Time
    
    // CorrelationID returns correlation ID from context
    CorrelationID() string
}

type EventHandler func(ctx context.Context, event DomainEvent) error

type Subscription interface {
    // Unsubscribe removes the subscription
    Unsubscribe()
}
```

### Event Types

**Standard Domain Events**:

```go
const (
    EventTypePipelineStarted     = "pipeline.started"
    EventTypePipelineCompleted   = "pipeline.completed"
    EventTypePipelineFailed      = "pipeline.failed"
    EventTypeStepStarted         = "step.started"
    EventTypeStepCompleted       = "step.completed"
    EventTypeStepFailed          = "step.failed"
    EventTypeStepSkipped         = "step.skipped"
    EventTypeValidationStarted   = "validation.started"
    EventTypeValidationCompleted = "validation.completed"
    EventTypeValidationFailed    = "validation.failed"
)
```

**Event Payload Examples**:

```go
type PipelineStartedPayload struct {
    PipelineName string
    StepCount    int
    DryRun       bool
}

type StepCompletedPayload struct {
    StepID      string
    StepType    string
    Duration    time.Duration
    Changed     bool
    Status      string
}
```

### Design Decisions

#### 1. In-Process Only (No External Message Broker)

**Rationale**:
- Streamy is a CLI tool with short-lived processes
- No need for persistent message queues or distributed pub/sub
- External brokers (Kafka, RabbitMQ, NATS) add unnecessary complexity
- In-process events sufficient for observability and future extensibility

**Trade-off**: Events lost if process crashes (acceptable for CLI tool).

#### 2. Synchronous Dispatch with Asynchronous Handlers

**Delivery Model**:
- `Publish()` dispatches to all subscribers **synchronously** within the same goroutine
- Each subscriber's handler **may** spawn a goroutine for async processing
- Publisher waits for synchronous handlers but doesn't wait for async work

**Rationale**:
- Synchronous dispatch ensures events delivered before process exits
- Handlers can choose async processing if needed (e.g., logging is fire-and-forget)
- Simpler than full async queue with guaranteed delivery

**Example**:

```go
// Publisher dispatches synchronously
func (p *EventPublisher) Publish(ctx context.Context, event DomainEvent) error {
    for _, handler := range p.subscribers[event.Type()] {
        // Call handler synchronously
        if err := handler(ctx, event); err != nil {
            // Log error but continue to other handlers
            p.logger.Warn(ctx, "event handler error", "error", err)
        }
    }
    return nil
}

// Handler can spawn goroutine for async work
func LoggingEventHandler(logger Logger) EventHandler {
    return func(ctx context.Context, event DomainEvent) error {
        // Log synchronously (fast)
        logger.Info(ctx, "domain event", 
            "event_type", event.Type(),
            "payload", event.Payload(),
        )
        return nil
    }
}

// Or handler can be fully synchronous (blocks publisher)
func MetricsEventHandler(collector MetricsCollector) EventHandler {
    return func(ctx context.Context, event DomainEvent) error {
        // Update metrics synchronously (also fast)
        collector.IncCounter(ctx, "streamy_domain_events_total", map[string]string{
            "event_type": event.Type(),
        })
        return nil
    }
}
```

#### 3. No Ordering Guarantees Between Event Types

**Ordering Model**:
- Events of the **same type** delivered in order published
- Events of **different types** have **no ordering guarantees**
- Handlers execute in subscription order

**Rationale**:
- Simpler implementation (no global event queue)
- Most handlers don't depend on cross-event ordering
- If ordering matters, use single event type or serialize in application layer

**Example**:

```go
// Events published in this order:
publisher.Publish(ctx, &PipelineStartedEvent{})
publisher.Publish(ctx, &StepStartedEvent{})
publisher.Publish(ctx, &StepCompletedEvent{})

// Handlers receive:
// - PipelineStartedEvent handlers called first (in subscription order)
// - StepStartedEvent handlers called second (in subscription order)
// - StepCompletedEvent handlers called third (in subscription order)

// But handlers for different event types may interleave if async
```

#### 4. No Backpressure or Buffering

**Buffering Model**:
- **No event buffer** between publisher and subscribers
- Publisher blocks until all synchronous handlers complete
- If handler is slow, publisher is blocked

**Rationale**:
- Simple synchronous dispatch model
- Handlers expected to be fast (logging, metrics)
- Slow handlers should spawn goroutines for heavy work
- CLI tool doesn't need complex backpressure mechanisms

**Mitigation for Slow Handlers**:

```go
// BAD: Slow synchronous handler blocks publisher
func SlowEventHandler(logger Logger) EventHandler {
    return func(ctx context.Context, event DomainEvent) error {
        time.Sleep(1 * time.Second) // Blocks publisher!
        logger.Info(ctx, "event", "type", event.Type())
        return nil
    }
}

// GOOD: Fast synchronous handler with async work
func FastEventHandler(logger Logger) EventHandler {
    return func(ctx context.Context, event DomainEvent) error {
        // Spawn goroutine for slow work
        go func() {
            time.Sleep(1 * time.Second)
            logger.Info(context.Background(), "event", "type", event.Type())
        }()
        return nil
    }
}

// BETTER: Use event queue in handler if needed
func QueuedEventHandler(queue EventQueue, logger Logger) EventHandler {
    return func(ctx context.Context, event DomainEvent) error {
        // Fast enqueue operation
        queue.Enqueue(event)
        return nil
    }
}
```

#### 5. Handler Errors Logged, Not Propagated

**Error Handling**:
- Handler errors logged but **do not** stop event dispatch
- Other handlers still receive the event
- Publisher returns `nil` even if handlers error

**Rationale**:
- Domain operations shouldn't fail because observability handler failed
- Handlers are side effects, not critical path
- Errors logged for debugging

**Example**:

```go
func (p *EventPublisher) Publish(ctx context.Context, event DomainEvent) error {
    for _, handler := range p.subscribers[event.Type()] {
        if err := handler(ctx, event); err != nil {
            // Log but continue
            p.logger.Warn(ctx, "event handler failed",
                "event_type", event.Type(),
                "error", err,
            )
        }
    }
    // Always return nil (handler errors don't fail publish)
    return nil
}
```

#### 6. Thread-Safe Subscription Management

**Concurrency Model**:
- `Subscribe()` and `Unsubscribe()` protected by mutex
- Safe to subscribe/unsubscribe from multiple goroutines
- `Publish()` creates snapshot of subscribers (no mid-publish subscription changes)

**Implementation**:

```go
type EventPublisher struct {
    mu          sync.RWMutex
    subscribers map[string][]EventHandler
    logger      Logger
}

func (p *EventPublisher) Subscribe(eventType string, handler EventHandler) Subscription {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.subscribers[eventType] = append(p.subscribers[eventType], handler)
    
    return &subscription{
        publisher: p,
        eventType: eventType,
        handler:   handler,
    }
}

func (p *EventPublisher) Publish(ctx context.Context, event DomainEvent) error {
    p.mu.RLock()
    handlers := append([]EventHandler{}, p.subscribers[event.Type()]...)
    p.mu.RUnlock()
    
    for _, handler := range handlers {
        if err := handler(ctx, event); err != nil {
            p.logger.Warn(ctx, "event handler error", "error", err)
        }
    }
    return nil
}
```

### Implementation

**Adapter** (internal/infrastructure/events/publisher.go):

```go
type EventPublisher struct {
    mu          sync.RWMutex
    subscribers map[string][]EventHandler
    logger      Logger
}

func NewEventPublisher(logger Logger) *EventPublisher {
    return &EventPublisher{
        subscribers: make(map[string][]EventHandler),
        logger:      logger,
    }
}

func (p *EventPublisher) Publish(ctx context.Context, event DomainEvent) error {
    // Get snapshot of subscribers
    p.mu.RLock()
    handlers := append([]EventHandler{}, p.subscribers[event.Type()]...)
    p.mu.RUnlock()
    
    // Dispatch to all handlers
    for _, handler := range handlers {
        if err := handler(ctx, event); err != nil {
            p.logger.Warn(ctx, "event handler failed",
                "event_type", event.Type(),
                "error", err,
            )
        }
    }
    
    return nil
}

func (p *EventPublisher) Subscribe(eventType string, handler EventHandler) Subscription {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.subscribers[eventType] = append(p.subscribers[eventType], handler)
    
    return &subscription{
        publisher: p,
        eventType: eventType,
        handler:   handler,
    }
}

type subscription struct {
    publisher *EventPublisher
    eventType string
    handler   EventHandler
}

func (s *subscription) Unsubscribe() {
    s.publisher.mu.Lock()
    defer s.publisher.mu.Unlock()
    
    handlers := s.publisher.subscribers[s.eventType]
    for i, h := range handlers {
        // Compare function pointers
        if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", s.handler) {
            // Remove handler
            s.publisher.subscribers[s.eventType] = append(handlers[:i], handlers[i+1:]...)
            break
        }
    }
}
```

### Usage Example

**Publishing Events** (in ApplyUseCase):

```go
func (u *ApplyUseCase) Apply(ctx context.Context, configPath string, dryRun bool) error {
    // Load pipeline...
    
    // Publish pipeline started event
    u.publisher.Publish(ctx, &PipelineStartedEvent{
        timestamp:     time.Now(),
        correlationID: logging.GetCorrelationID(ctx),
        payload: PipelineStartedPayload{
            PipelineName: pip.Name,
            StepCount:    len(pip.Steps),
            DryRun:       dryRun,
        },
    })
    
    // Execute steps...
    for _, step := range pip.Steps {
        u.publisher.Publish(ctx, &StepStartedEvent{...})
        
        result, err := u.executor.ExecuteStep(ctx, step)
        
        if err != nil {
            u.publisher.Publish(ctx, &StepFailedEvent{...})
        } else {
            u.publisher.Publish(ctx, &StepCompletedEvent{...})
        }
    }
    
    // Publish pipeline completed
    u.publisher.Publish(ctx, &PipelineCompletedEvent{...})
    
    return nil
}
```

**Subscribing to Events** (in main.go):

```go
func main() {
    // Create observability components
    logger := logging.NewLogger("info")
    metrics := metrics.NewCollector()
    publisher := events.NewEventPublisher(logger)
    
    // Subscribe to events for observability
    publisher.Subscribe(events.EventTypePipelineStarted, func(ctx context.Context, event DomainEvent) error {
        payload := event.Payload().(PipelineStartedPayload)
        logger.Info(ctx, "pipeline started", 
            "pipeline", payload.PipelineName,
            "step_count", payload.StepCount,
            "dry_run", payload.DryRun,
        )
        metrics.IncCounter(ctx, "streamy_pipeline_executions_total", map[string]string{
            "status": "started",
        })
        return nil
    })
    
    publisher.Subscribe(events.EventTypeStepCompleted, func(ctx context.Context, event DomainEvent) error {
        payload := event.Payload().(StepCompletedPayload)
        logger.Info(ctx, "step completed",
            "step_id", payload.StepID,
            "duration_ms", payload.Duration.Milliseconds(),
            "changed", payload.Changed,
        )
        metrics.ObserveHistogram(ctx, "streamy_step_execution_duration_seconds",
            payload.Duration.Seconds(), map[string]string{
                "step_type": payload.StepType,
            })
        return nil
    })
    
    // Wire publisher into use cases
    applyUseCase := pipeline.NewApplyUseCase(/* ... */, publisher)
    
    // Run CLI
    rootCmd.ExecuteContext(ctx)
}
```

### Testing

**Mock EventPublisher**:

```go
type MockEventPublisher struct {
    PublishedEvents []DomainEvent
    mu              sync.Mutex
}

func (m *MockEventPublisher) Publish(ctx context.Context, event DomainEvent) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.PublishedEvents = append(m.PublishedEvents, event)
    return nil
}

func (m *MockEventPublisher) Subscribe(eventType string, handler EventHandler) Subscription {
    // No-op for testing
    return &noopSubscription{}
}

// In tests
func TestApplyUseCase_PublishesEvents(t *testing.T) {
    mockPublisher := &MockEventPublisher{}
    useCase := NewApplyUseCase(/* ... */, mockPublisher)
    
    useCase.Apply(context.Background(), "config.yaml", false)
    
    // Verify events published
    if len(mockPublisher.PublishedEvents) != 3 {
        t.Errorf("expected 3 events, got %d", len(mockPublisher.PublishedEvents))
    }
    
    // Verify event types
    if mockPublisher.PublishedEvents[0].Type() != events.EventTypePipelineStarted {
        t.Error("expected first event to be pipeline started")
    }
}
```

### Future Extensibility

The EventPublisher design enables future features without domain changes:

1. **Webhooks**: Subscribe webhook handler to send HTTP requests on events
2. **Audit Trail**: Subscribe handler to write events to audit log file
3. **Workflow Triggers**: Subscribe handler to trigger dependent pipelines
4. **Real-time Dashboard**: Subscribe WebSocket handler to push events to TUI
5. **Notifications**: Subscribe handler to send Slack/email alerts on failures

**Example - Webhook Handler**:

```go
func NewWebhookHandler(webhookURL string) EventHandler {
    return func(ctx context.Context, event DomainEvent) error {
        // Spawn goroutine for async HTTP request
        go func() {
            payload, _ := json.Marshal(event)
            http.Post(webhookURL, "application/json", bytes.NewReader(payload))
        }()
        return nil
    }
}

// Register in main.go
publisher.Subscribe(events.EventTypePipelineCompleted, NewWebhookHandler("https://example.com/webhook"))
```

---

## Implementation Checklist

- [ ] Define EventPublisher, DomainEvent interfaces in `internal/ports/events.go`
- [ ] Define standard event types (pipeline.*, step.*, validation.*)
- [ ] Define event payload structs
- [ ] Implement EventPublisher adapter in `internal/infrastructure/events/publisher.go`
- [ ] Implement base DomainEvent struct with common fields
- [ ] Implement thread-safe subscription management
- [ ] Add event publishing to all use cases (Apply, Verify, Prepare)
- [ ] Subscribe logging handler in main.go
- [ ] Subscribe metrics handler in main.go
- [ ] Create MockEventPublisher for testing
- [ ] Add event publishing tests to use case tests
- [ ] Document event types and payloads in API documentation

---

## Related Documents

- **Architecture Overview**: `docs/architecture-overview.md`
- **Specification**: `specs/009-domain-driven-refactor/spec.md` - FR-012 (event emission)
- **Data Model**: `specs/009-domain-driven-refactor/data-model.md` - EventPublisher port
- **Error Contracts**: `specs/009-domain-driven-refactor/errors.md`

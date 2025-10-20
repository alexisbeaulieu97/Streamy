package testutil

import (
	"context"
	"sync"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// MockConfigLoader provides configurable config loading behavior for tests.
type MockConfigLoader struct {
	LoadFunc     func(ctx context.Context, path string) (*domainpipeline.Pipeline, error)
	ValidateFunc func(ctx context.Context, path string) error
}

// Load invokes LoadFunc when provided, returning nil otherwise.
func (m *MockConfigLoader) Load(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
	if m != nil && m.LoadFunc != nil {
		return m.LoadFunc(ctx, path)
	}

	return nil, nil
}

// Validate invokes ValidateFunc when provided, returning nil otherwise.
func (m *MockConfigLoader) Validate(ctx context.Context, path string) error {
	if m != nil && m.ValidateFunc != nil {
		return m.ValidateFunc(ctx, path)
	}

	return nil
}

// MockDAGBuilder captures DAG build requests made by the application layer.
type MockDAGBuilder struct {
	Calls     []BuildCall
	BuildFunc func(ctx context.Context, steps []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error)
	mu        sync.Mutex
}

// BuildCall records a single Build invocation for later inspection.
type BuildCall struct {
	Ctx   context.Context
	Steps []domainpipeline.Step
}

// Build tracks the invocation and delegates to BuildFunc when available.
func (m *MockDAGBuilder) Build(ctx context.Context, steps []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
	if m == nil {
		return nil, nil
	}

	m.mu.Lock()
	m.Calls = append(m.Calls, BuildCall{Ctx: ctx, Steps: steps})
	m.mu.Unlock()

	if m.BuildFunc != nil {
		return m.BuildFunc(ctx, steps)
	}

	return nil, nil
}

// MockPluginExecutor records plugin execution requests and allows test hooks.
type MockPluginExecutor struct {
	ExecuteFunc func(ctx context.Context, plan *domainpipeline.ExecutionPlan, pipeline *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error)
	VerifyFunc  func(ctx context.Context, pipeline *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error)

	ExecuteCalls []ExecuteCall
	VerifyCalls  []VerifyCall

	mu sync.Mutex
}

// ExecuteCall captures Execute parameters for asserting test expectations.
type ExecuteCall struct {
	Ctx      context.Context
	Plan     *domainpipeline.ExecutionPlan
	Pipeline *domainpipeline.Pipeline
}

// VerifyCall captures Verify parameters for asserting test expectations.
type VerifyCall struct {
	Ctx      context.Context
	Pipeline *domainpipeline.Pipeline
}

// Execute records the call and forwards to ExecuteFunc when configured.
func (m *MockPluginExecutor) Execute(ctx context.Context, plan *domainpipeline.ExecutionPlan, pipeline *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
	if m == nil {
		return nil, nil
	}

	m.mu.Lock()
	m.ExecuteCalls = append(m.ExecuteCalls, ExecuteCall{Ctx: ctx, Plan: plan, Pipeline: pipeline})
	m.mu.Unlock()

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, plan, pipeline)
	}

	return nil, nil
}

// Verify records the call and forwards to VerifyFunc when configured.
func (m *MockPluginExecutor) Verify(ctx context.Context, pipeline *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
	if m == nil {
		return nil, nil
	}

	m.mu.Lock()
	m.VerifyCalls = append(m.VerifyCalls, VerifyCall{Ctx: ctx, Pipeline: pipeline})
	m.mu.Unlock()

	if m.VerifyFunc != nil {
		return m.VerifyFunc(ctx, pipeline)
	}

	return nil, nil
}

var _ ports.PluginExecutor = (*MockPluginExecutor)(nil)

// EventRecord describes a domain event published via MockEventPublisher.
type EventRecord struct {
	Ctx     context.Context
	Event   ports.DomainEvent
	Type    string
	Payload interface{}
}

// MockEventPublisher tracks published events for verification in tests.
type MockEventPublisher struct {
	Published   []EventRecord
	PublishFunc func(ctx context.Context, event ports.DomainEvent) error
	mu          sync.Mutex
}

// Publish appends the event and delegates to PublishFunc when provided.
func (m *MockEventPublisher) Publish(ctx context.Context, event ports.DomainEvent) error {
	if m == nil {
		return nil
	}

	record := EventRecord{Ctx: ctx, Event: event, Type: event.EventType(), Payload: event.Payload()}

	m.mu.Lock()
	m.Published = append(m.Published, record)
	m.mu.Unlock()

	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, event)
	}

	return nil
}

// Subscribe satisfies EventPublisher and returns a no-op subscription.
func (m *MockEventPublisher) Subscribe(_ string, _ ports.EventHandler) (ports.Subscription, error) {
	return noopSubscription{}, nil
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() {}

var _ ports.EventPublisher = (*MockEventPublisher)(nil)

// MockLogger records structured log entries.
type MockLogger struct {
	entries []LogEntry
	fields  []interface{}
	mu      sync.Mutex
}

// LogEntry captures a structured log emitted by MockLogger.
type LogEntry struct {
	Level  string
	Msg    string
	Fields map[string]interface{}
}

// NewMockLogger returns a logger that records entries for assertions.
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

// Debug records a debug-level log entry.
func (l *MockLogger) Debug(_ context.Context, msg string, fields ...interface{}) {
	l.append("debug", msg, fields...)
}

// Info records an info-level log entry.
func (l *MockLogger) Info(_ context.Context, msg string, fields ...interface{}) {
	l.append("info", msg, fields...)
}

// Warn records a warning-level log entry.
func (l *MockLogger) Warn(_ context.Context, msg string, fields ...interface{}) {
	l.append("warn", msg, fields...)
}

// Error records an error-level log entry.
func (l *MockLogger) Error(_ context.Context, msg string, fields ...interface{}) {
	l.append("error", msg, fields...)
}

// With clones the logger and appends persistent structured fields.
func (l *MockLogger) With(fields ...interface{}) ports.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	clonedEntries := make([]LogEntry, len(l.entries))
	copy(clonedEntries, l.entries)

	clonedFields := append([]interface{}{}, l.fields...)
	clonedFields = append(clonedFields, fields...)

	return &MockLogger{
		entries: clonedEntries,
		fields:  clonedFields,
	}
}

// Entries returns a snapshot of all recorded log entries.
func (l *MockLogger) Entries() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]LogEntry, len(l.entries))
	copy(out, l.entries)

	return out
}

func (l *MockLogger) append(level, msg string, fields ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Level:  level,
		Msg:    msg,
		Fields: toFieldMap(append(l.fields, fields...)),
	}
	l.entries = append(l.entries, entry)
}

func toFieldMap(fields []interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return map[string]interface{}{}
	}

	result := make(map[string]interface{}, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok || key == "" {
			continue
		}

		result[key] = fields[i+1]
	}

	return result
}

var _ ports.Logger = (*MockLogger)(nil)

// MetricsCounterCall captures a counter increment for assertions.
type MetricsCounterCall struct {
	Ctx    context.Context
	Name   string
	Labels map[string]string
}

// MetricsGaugeCall captures a gauge set invocation.
type MetricsGaugeCall struct {
	Ctx    context.Context
	Name   string
	Value  float64
	Labels map[string]string
}

// MetricsHistogramCall captures a histogram observation.
type MetricsHistogramCall struct {
	Ctx    context.Context
	Name   string
	Value  float64
	Labels map[string]string
}

// MockMetricsCollector records metrics interactions for verification.
type MockMetricsCollector struct {
	CounterCalls   []MetricsCounterCall
	GaugeCalls     []MetricsGaugeCall
	HistogramCalls []MetricsHistogramCall
	mu             sync.Mutex
}

// NewMockMetricsCollector constructs a metrics collector that records calls.
func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{}
}

// IncCounter records a counter increment call.
func (m *MockMetricsCollector) IncCounter(ctx context.Context, name string, labels map[string]string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	m.CounterCalls = append(m.CounterCalls, MetricsCounterCall{
		Ctx:    ctx,
		Name:   name,
		Labels: cloneLabels(labels),
	})
	m.mu.Unlock()
}

// SetGauge records a gauge update call.
func (m *MockMetricsCollector) SetGauge(ctx context.Context, name string, value float64, labels map[string]string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	m.GaugeCalls = append(m.GaugeCalls, MetricsGaugeCall{
		Ctx:    ctx,
		Name:   name,
		Value:  value,
		Labels: cloneLabels(labels),
	})
	m.mu.Unlock()
}

// ObserveHistogram records a histogram observation call.
func (m *MockMetricsCollector) ObserveHistogram(ctx context.Context, name string, value float64, labels map[string]string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	m.HistogramCalls = append(m.HistogramCalls, MetricsHistogramCall{
		Ctx:    ctx,
		Name:   name,
		Value:  value,
		Labels: cloneLabels(labels),
	})
	m.mu.Unlock()
}

var _ ports.MetricsCollector = (*MockMetricsCollector)(nil)

// MockTracer records span creation.
type MockTracer struct {
	StartCalls []TracerStartCall
	mu         sync.Mutex
}

// TracerStartCall captures parameters passed to StartSpan.
type TracerStartCall struct {
	Ctx        context.Context
	Name       string
	Attributes []interface{}
}

// NewMockTracer constructs a tracer that records StartSpan calls.
func NewMockTracer() *MockTracer {
	return &MockTracer{}
}

// StartSpan records the span request and returns a mock span implementation.
func (m *MockTracer) StartSpan(ctx context.Context, name string, attributes ...interface{}) (context.Context, ports.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	if m != nil {
		m.mu.Lock()
		m.StartCalls = append(m.StartCalls, TracerStartCall{
			Ctx:        ctx,
			Name:       name,
			Attributes: append([]interface{}(nil), attributes...),
		})
		m.mu.Unlock()
	}

	return ctx, &MockSpan{}
}

// Inject satisfies the Tracer interface without propagating context.
func (*MockTracer) Inject(_ context.Context, _ interface{}) error {
	return nil
}

// Extract returns the supplied context to satisfy the Tracer interface.
func (*MockTracer) Extract(ctx context.Context, _ interface{}) (context.Context, error) {
	return ctx, nil
}

var _ ports.Tracer = (*MockTracer)(nil)

// MockSpan captures span lifecycle interactions.
type MockSpan struct {
	mu            sync.Mutex
	Attributes    map[string]interface{}
	Status        ports.SpanStatus
	StatusMessage string
	EndCount      int
}

// SetAttribute records an attribute assignment on the mock span.
func (s *MockSpan) SetAttribute(key string, value interface{}) {
	if key == "" {
		return
	}

	s.mu.Lock()

	if s.Attributes == nil {
		s.Attributes = make(map[string]interface{})
	}

	s.Attributes[key] = value
	s.mu.Unlock()
}

// SetStatus records the status set on the mock span.
func (s *MockSpan) SetStatus(status ports.SpanStatus, message string) {
	s.mu.Lock()
	s.Status = status
	s.StatusMessage = message
	s.mu.Unlock()
}

// End increments the end counter to track span completion.
func (s *MockSpan) End() {
	s.mu.Lock()
	s.EndCount++
	s.mu.Unlock()
}

var _ ports.Span = (*MockSpan)(nil)

func cloneLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}

	copied := make(map[string]string, len(labels))
	for k, v := range labels {
		copied[k] = v
	}

	return copied
}

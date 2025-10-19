package testutil

import (
	"context"
	"sync"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type MockConfigLoader struct {
	LoadFunc     func(ctx context.Context, path string) (*domainpipeline.Pipeline, error)
	ValidateFunc func(ctx context.Context, path string) error
}

func (m *MockConfigLoader) Load(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
	if m != nil && m.LoadFunc != nil {
		return m.LoadFunc(ctx, path)
	}
	return nil, nil
}

func (m *MockConfigLoader) Validate(ctx context.Context, path string) error {
	if m != nil && m.ValidateFunc != nil {
		return m.ValidateFunc(ctx, path)
	}
	return nil
}

type MockDAGBuilder struct {
	Calls     []BuildCall
	BuildFunc func(ctx context.Context, steps []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error)
	mu        sync.Mutex
}

type BuildCall struct {
	Ctx   context.Context
	Steps []domainpipeline.Step
}

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

type MockPluginExecutor struct {
	ExecuteFunc func(ctx context.Context, plan *domainpipeline.ExecutionPlan, pipeline *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error)
	VerifyFunc  func(ctx context.Context, pipeline *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error)

	ExecuteCalls []ExecuteCall
	VerifyCalls  []VerifyCall

	mu sync.Mutex
}

type ExecuteCall struct {
	Ctx      context.Context
	Plan     *domainpipeline.ExecutionPlan
	Pipeline *domainpipeline.Pipeline
}

type VerifyCall struct {
	Ctx      context.Context
	Pipeline *domainpipeline.Pipeline
}

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

type EventRecord struct {
	Ctx     context.Context
	Event   ports.DomainEvent
	Type    string
	Payload interface{}
}

type MockEventPublisher struct {
	Published   []EventRecord
	PublishFunc func(ctx context.Context, event ports.DomainEvent) error
	mu          sync.Mutex
}

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

func (m *MockEventPublisher) Subscribe(eventType string, handler ports.EventHandler) (ports.Subscription, error) {
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

type LogEntry struct {
	Level  string
	Msg    string
	Fields map[string]interface{}
}

func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (l *MockLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	l.append("debug", msg, fields...)
}

func (l *MockLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	l.append("info", msg, fields...)
}

func (l *MockLogger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	l.append("warn", msg, fields...)
}

func (l *MockLogger) Error(ctx context.Context, msg string, fields ...interface{}) {
	l.append("error", msg, fields...)
}

func (l *MockLogger) With(fields ...interface{}) ports.Logger {
	clone := &MockLogger{entries: l.entries}
	clone.fields = append(clone.fields, l.fields...)
	clone.fields = append(clone.fields, fields...)
	return clone
}

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

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{}
}

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

func NewMockTracer() *MockTracer {
	return &MockTracer{}
}

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

func (*MockTracer) Inject(_ context.Context, _ interface{}) error {
	return nil
}

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

func (s *MockSpan) SetStatus(status ports.SpanStatus, message string) {
	s.mu.Lock()
	s.Status = status
	s.StatusMessage = message
	s.mu.Unlock()
}

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

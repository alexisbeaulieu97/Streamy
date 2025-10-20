// Package tracing offers lightweight span instrumentation for pipelines.
package tracing

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Tracer provides a lightweight Span implementation that records attributes
// and emits structured logs when spans complete.
type Tracer struct {
	logger ports.Logger
}

// NewTracer constructs a tracer that emits span lifecycle events through the
// supplied logger. The logger may be nil when tracing is optional.
func NewTracer(logger ports.Logger) *Tracer {
	return &Tracer{logger: logger}
}

// StartSpan creates a new span with the provided name. Attributes are supplied
// as alternating key/value pairs.
func (t *Tracer) StartSpan(ctx context.Context, name string, attributes ...interface{}) (context.Context, ports.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	span := &span{
		ctx:        ctx,
		name:       name,
		start:      time.Now(),
		logger:     t.logger,
		attributes: make(map[string]interface{}),
	}

	if correlationID := ports.GetCorrelationID(ctx); correlationID != "" {
		span.attributes["correlation_id"] = correlationID
	}

	span.applyAttributes(attributes)

	return context.WithValue(ctx, spanContextKey{}, span), span
}

// Inject is a no-op for the in-process tracer.
func (t *Tracer) Inject(context.Context, interface{}) error {
	return nil
}

// Extract is a no-op for the in-process tracer.
func (t *Tracer) Extract(ctx context.Context, _ interface{}) (context.Context, error) {
	return ctx, nil
}

type spanContextKey struct{}

type span struct {
	ctx           context.Context
	name          string
	start         time.Time
	logger        ports.Logger
	mu            sync.Mutex
	attributes    map[string]interface{}
	status        ports.SpanStatus
	statusMessage string
	completed     bool
}

func (s *span) SetAttribute(key string, value interface{}) {
	if key == "" {
		return
	}

	s.mu.Lock()
	s.attributes[key] = value
	s.mu.Unlock()
}

func (s *span) SetStatus(status ports.SpanStatus, message string) {
	s.mu.Lock()
	s.status = status
	s.statusMessage = message
	s.mu.Unlock()
}

func (s *span) End() {
	s.mu.Lock()

	if s.completed {
		s.mu.Unlock()
		return
	}

	s.completed = true
	duration := time.Since(s.start)
	status := s.status
	message := s.statusMessage
	attributes := s.copyAttributesLocked()
	s.mu.Unlock()

	if s.logger != nil {
		keys := make([]string, 0, len(attributes))
		for key := range attributes {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		logFields := []interface{}{
			"span", s.name,
			"duration_ms", duration.Milliseconds(),
		}
		for _, key := range keys {
			logFields = append(logFields, key, attributes[key])
		}

		if status != "" {
			logFields = append(logFields, "status", string(status))
		}

		if message != "" {
			logFields = append(logFields, "status_message", message)
		}

		s.logger.Debug(s.ctx, "span completed", logFields...)
	}
}

func (s *span) copyAttributesLocked() map[string]interface{} {
	if len(s.attributes) == 0 {
		return nil
	}

	cloned := make(map[string]interface{}, len(s.attributes))
	for k, v := range s.attributes {
		cloned[k] = v
	}

	return cloned
}

func (s *span) applyAttributes(attributes []interface{}) {
	if len(attributes) == 0 {
		return
	}

	for i := 0; i+1 < len(attributes); i += 2 {
		key, ok := attributes[i].(string)
		if !ok || key == "" {
			continue
		}

		s.attributes[key] = attributes[i+1]
	}
}

// NoOpTracer returns spans that discard all operations.
type NoOpTracer struct{}

// NewNoOpTracer constructs a tracer that does nothing.
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

// StartSpan returns the original context with a no-op span.
func (*NoOpTracer) StartSpan(ctx context.Context, _ string, _ ...interface{}) (context.Context, ports.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	return ctx, noopSpan{}
}

// Inject implements ports.Tracer.
func (*NoOpTracer) Inject(_ context.Context, _ interface{}) error { return nil }

// Extract implements ports.Tracer.
func (*NoOpTracer) Extract(ctx context.Context, _ interface{}) (context.Context, error) {
	return ctx, nil
}

type noopSpan struct{}

func (noopSpan) SetAttribute(_ string, _ interface{}) {}

func (noopSpan) SetStatus(_ ports.SpanStatus, _ string) {}

func (noopSpan) End() {}

var (
	_ ports.Tracer = (*Tracer)(nil)
	_ ports.Tracer = (*NoOpTracer)(nil)
	_ ports.Span   = (*span)(nil)
	_ ports.Span   = noopSpan{}
)

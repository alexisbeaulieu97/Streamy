package tracing

import (
	"context"
	"sync"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

func TestTracer_StartSpanRecordsAttributes(t *testing.T) {
	t.Parallel()

	logger := newRecordingLogger()
	tracer := NewTracer(logger)

	ctx := ports.WithCorrelationID(context.Background(), "1234")
	ctx, spanHandle := tracer.StartSpan(ctx, "pipeline.apply", "config_path", "pipeline.yaml", "dry_run", true)

	if spanHandle == nil {
		t.Fatal("expected span instance, got nil")
	}

	spanHandle.SetAttribute("pipeline", "demo")
	spanHandle.SetStatus(ports.SpanStatusOK, "completed")
	spanHandle.End()

	entries := logger.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Level != "debug" || entry.Msg != "span completed" {
		t.Fatalf("unexpected log entry: %+v", entry)
	}

	if got := entry.Fields["span"]; got != "pipeline.apply" {
		t.Fatalf("expected span name, got %v", got)
	}

	if got := entry.Fields["correlation_id"]; got != "1234" {
		t.Fatalf("expected correlation id propagated, got %v", got)
	}

	if got := entry.Fields["config_path"]; got != "pipeline.yaml" {
		t.Fatalf("expected config_path attribute, got %v", got)
	}

	if got := entry.Fields["pipeline"]; got != "demo" {
		t.Fatalf("expected pipeline attribute, got %v", got)
	}

	if got := entry.Fields["status"]; got != string(ports.SpanStatusOK) {
		t.Fatalf("expected status ok, got %v", got)
	}

	if got := entry.Fields["status_message"]; got != "completed" {
		t.Fatalf("expected status message, got %v", got)
	}

	// Ensure context propagation round-trips unchanged.
	if stored := ctx.Value(spanContextKey{}); stored == nil {
		t.Fatal("expected span stored in context")
	}
}

func TestNoOpTracer_NoPanics(t *testing.T) {
	t.Parallel()

	tracer := NewNoOpTracer()
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "noop")
	if ctx == nil || span == nil {
		t.Fatal("expected non-nil context and span")
	}

	span.SetAttribute("key", "value")
	span.SetStatus(ports.SpanStatusError, "error")
	span.End()
}

type recordingLogger struct {
	mu      sync.Mutex
	entries []logEntry
	fields  []interface{}
}

type logEntry struct {
	Level  string
	Msg    string
	Fields map[string]interface{}
}

func newRecordingLogger() *recordingLogger {
	return &recordingLogger{}
}

func (l *recordingLogger) Debug(_ context.Context, msg string, fields ...interface{}) {
	l.append("debug", msg, fields...)
}

func (l *recordingLogger) Info(_ context.Context, msg string, fields ...interface{}) {
	l.append("info", msg, fields...)
}

func (l *recordingLogger) Warn(_ context.Context, msg string, fields ...interface{}) {
	l.append("warn", msg, fields...)
}

func (l *recordingLogger) Error(_ context.Context, msg string, fields ...interface{}) {
	l.append("error", msg, fields...)
}

func (l *recordingLogger) With(fields ...interface{}) ports.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	clone := &recordingLogger{
		fields: append(append([]interface{}{}, l.fields...), fields...),
	}

	return clone
}

func (l *recordingLogger) Entries() []logEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	snapshot := make([]logEntry, len(l.entries))
	copy(snapshot, l.entries)

	return snapshot
}

func (l *recordingLogger) append(level, msg string, fields ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := logEntry{
		Level:  level,
		Msg:    msg,
		Fields: fieldMap(append(append([]interface{}{}, l.fields...), fields...)),
	}
	l.entries = append(l.entries, entry)
}

func fieldMap(fields []interface{}) map[string]interface{} {
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

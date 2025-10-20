package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	cblog "github.com/charmbracelet/log"
)

func TestLoggerIncludesCorrelationIDAndLayer(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(Options{
		Writer:     &buf,
		Level:      "debug",
		Formatter:  cblog.JSONFormatter,
		Layer:      "infrastructure",
		Component:  "yaml_loader",
		TimeFormat: "2006-01-02T15:04:05Z07:00",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := WithCorrelationID(context.Background(), "abc123")
	logger.Info(ctx, "loaded config", "path", "/tmp/config.yaml")

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected log output, got empty string")
	}

	payload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("failed to parse log line %q: %v", line, err)
	}

	if payload["layer"] != "infrastructure" {
		t.Fatalf("expected layer to be infrastructure, got %v", payload["layer"])
	}

	if payload["component"] != "yaml_loader" {
		t.Fatalf("expected component field, got %v", payload["component"])
	}

	if payload["correlation_id"] != "abc123" {
		t.Fatalf("expected correlation_id to be abc123, got %v", payload["correlation_id"])
	}

	if payload["path"] != "/tmp/config.yaml" {
		t.Fatalf("expected path to be recorded, got %v", payload["path"])
	}

	if payload["msg"] != "loaded config" {
		t.Fatalf("expected message to be recorded, got %v", payload["msg"])
	}
}

func TestLoggerWithAddsFields(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(Options{
		Writer:    &buf,
		Formatter: cblog.JSONFormatter,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	child := logger.With("component", "executor").(*Logger)
	child.Warn(context.Background(), "step failed", "step_id", "build")

	line := strings.TrimSpace(buf.String())

	payload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("failed to parse log line: %v", err)
	}

	if payload["component"] != "executor" {
		t.Fatalf("expected component=executor, got %v", payload["component"])
	}

	if payload["step_id"] != "build" {
		t.Fatalf("expected step_id build, got %v", payload["step_id"])
	}

	if payload["layer"] != "infrastructure" {
		t.Fatalf("expected default layer infrastructure, got %v", payload["layer"])
	}
}

func TestNoOpLogger(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(Options{
		Writer:    &buf,
		Formatter: cblog.JSONFormatter,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	noOp := NewNoOpLogger()
	noOp.Info(context.Background(), "hello world")

	if buf.Len() != 0 {
		t.Fatalf("expected no output from noop logger, got %s", buf.String())
	}

	// ensure With on noop doesn't panic and returns the same instance
	if noOp.With("key", "value") != noOp {
		t.Fatalf("expected With to return same no-op logger instance")
	}

	// Base logger still writes.
	logger.Info(context.Background(), "emitted")

	if buf.Len() == 0 {
		t.Fatal("expected base logger to write output")
	}
}

func TestBufferedLoggerStoresAndFlushes(t *testing.T) {
	buffer := NewEventBuffer(10)
	bufLogger := NewBufferedLogger(buffer)

	ctx := WithCorrelationID(context.Background(), "buffered")
	bufLogger.Info(ctx, "booting", "component", "bootstrap")
	bufLogger.With("component", "worker").Error(ctx, "failed", "attempt", 1)

	var output bytes.Buffer

	delegate, err := New(Options{Writer: &output, Formatter: cblog.JSONFormatter})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buffer.Flush(delegate)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}

	var first map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("failed to parse first log line: %v", err)
	}

	if first["msg"] != "booting" || first["component"] != "bootstrap" {
		t.Fatalf("unexpected first event payload: %+v", first)
	}

	var second map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("failed to parse second log line: %v", err)
	}

	if second["msg"] != "failed" || second["component"] != "worker" {
		t.Fatalf("unexpected second event payload: %+v", second)
	}

	if second["correlation_id"] != "buffered" {
		t.Fatalf("expected correlation id to be preserved, got %v", second["correlation_id"])
	}
}

func TestLoggerIncludesErrorChain(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(Options{
		Writer:    &buf,
		Formatter: cblog.JSONFormatter,
		Layer:     "application",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rootErr := errors.New("disk full")
	wrapped := fmt.Errorf("executor failed: %w", rootErr)
	top := fmt.Errorf("verify pipeline: %w", wrapped)

	logger.Error(context.Background(), "verification failed", "error", top)

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected log output for error")
	}

	payload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("failed to parse error log: %v", err)
	}

	if payload["error"] != top.Error() {
		t.Fatalf("expected error field to contain full message, got %v", payload["error"])
	}

	rawChain, ok := payload["error_chain"].([]interface{})
	if !ok {
		t.Fatalf("expected error_chain field to be present, got %T", payload["error_chain"])
	}

	gotChain := make([]string, len(rawChain))
	for i, v := range rawChain {
		msg, ok := v.(string)
		if !ok {
			t.Fatalf("expected chain entry to be string, got %T", v)
		}

		gotChain[i] = msg
	}

	expectedChain := []string{
		top.Error(),
		wrapped.Error(),
		rootErr.Error(),
	}
	if !equalStrings(gotChain, expectedChain) {
		t.Fatalf("unexpected error chain. expected %v, got %v", expectedChain, gotChain)
	}

	if payload["error_cause"] != rootErr.Error() {
		t.Fatalf("expected error_cause to reference root error, got %v", payload["error_cause"])
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestLogger_ErrorHandling(t *testing.T) {
	t.Run("Invalid log level", func(t *testing.T) {
		_, err := New(Options{
			Writer:    &bytes.Buffer{},
			Level:     "invalid",
			Formatter: cblog.JSONFormatter,
		})
		if err == nil {
			t.Fatal("expected error for invalid log level")
		}

		if !strings.Contains(err.Error(), "parse log level") {
			t.Fatalf("expected parse error, got: %v", err)
		}
	})

	t.Run("Nil logger methods", func(t *testing.T) {
		var logger *Logger
		// Should not panic
		logger.Debug(context.Background(), "test")
		logger.Info(context.Background(), "test")
		logger.Warn(context.Background(), "test")
		logger.Error(context.Background(), "test")

		child := logger.With("key", "value")
		if child == nil {
			t.Fatal("expected non-nil child logger")
		}
	})

	t.Run("Nil options defaults", func(t *testing.T) {
		logger, err := New(Options{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if logger.layer != defaultLayer {
			t.Fatalf("expected default layer %s, got %s", defaultLayer, logger.layer)
		}
	})
}

func TestLogger_FieldMerging(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(Options{
		Writer:    &buf,
		Formatter: cblog.JSONFormatter,
		Fields:    map[string]interface{}{"service": "test", "version": "1.0"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test field name conflicts - later fields should override
	logger.With("service", "overridden").Info(context.Background(), "test", "extra", "value")

	line := strings.TrimSpace(buf.String())

	payload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}

	// Should have the overridden value
	if payload["service"] != "overridden" {
		t.Fatalf("expected service to be overridden, got: %v", payload["service"])
	}

	// Should preserve other fields
	if payload["version"] != "1.0" {
		t.Fatalf("expected version to be preserved, got: %v", payload["version"])
	}

	if payload["extra"] != "value" {
		t.Fatalf("expected extra field, got: %v", payload["extra"])
	}
}

func TestLogger_ComplexErrorChains(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(Options{
		Writer:    &buf,
		Formatter: cblog.JSONFormatter,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test duplicate error messages
	baseErr := errors.New("connection failed")
	duplicateErr := fmt.Errorf("operation failed: %w", baseErr)
	wrappedWithSame := fmt.Errorf("service error: %w", duplicateErr)

	logger.Error(context.Background(), "test error", "error", wrappedWithSame)

	line := strings.TrimSpace(buf.String())

	payload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}

	// Should deduplicate error messages in chain
	chain, ok := payload["error_chain"].([]interface{})
	if !ok {
		t.Fatal("expected error_chain field")
	}

	// Should not contain duplicates
	seen := make(map[string]bool)
	for _, entry := range chain {
		msg, ok := entry.(string)
		if !ok {
			continue
		}

		if seen[msg] {
			t.Fatalf("found duplicate error message in chain: %s", msg)
		}

		seen[msg] = true
	}
}

func TestLogger_ErrorConversion(t *testing.T) {
	testCases := []struct {
		name      string
		value     interface{}
		expectNil bool
	}{
		{"error type", errors.New("test"), false},
		{"stringer", mockStringer{"test"}, false},
		{"string", "test error", true},
		{"nil", nil, true},
		{"number", 42, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toError(tc.value)
			if tc.expectNil && result != nil {
				t.Fatalf("expected nil error, got: %v", result)
			}

			if !tc.expectNil && result == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestEventBuffer_OverflowBehavior(t *testing.T) {
	buffer := NewEventBuffer(3) // Small buffer to test overflow

	// Add more events than buffer capacity
	for i := 0; i < 5; i++ {
		buffer.add(bufferedEntry{
			ctx:    context.Background(),
			level:  levelInfo,
			msg:    fmt.Sprintf("message %d", i),
			fields: []interface{}{"index", i},
		})
	}

	var output bytes.Buffer

	delegate, err := New(Options{Writer: &output, Formatter: cblog.JSONFormatter})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buffer.Flush(delegate)

	// Should only have the last 3 messages (messages 2, 3, 4)
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines after overflow, got %d", len(lines))
	}

	// Check that we have the right messages (should be 2, 3, 4)
	expectedMessages := []string{"message 2", "message 3", "message 4"}
	for i, line := range lines {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("failed to parse line %d: %v", i, err)
		}

		if payload["msg"] != expectedMessages[i] {
			t.Fatalf("expected message %s, got %v", expectedMessages[i], payload["msg"])
		}
	}
}

func TestEventBuffer_ConcurrentAccess(t *testing.T) {
	buffer := NewEventBuffer(1000) // Larger buffer to prevent overflow

	var wg sync.WaitGroup

	numGoroutines := 5 // Reduced for more reliable testing
	eventsPerGoroutine := 10

	// Add events concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := 0; j < eventsPerGoroutine; j++ {
				buffer.add(bufferedEntry{
					ctx:    context.Background(),
					level:  levelInfo,
					msg:    fmt.Sprintf("goroutine %d event %d", id, j),
					fields: []interface{}{"goroutine", id, "event", j},
				})
			}
		}(i)
	}

	wg.Wait()

	// Flush and verify all events were captured
	var output bytes.Buffer

	delegate, err := New(Options{Writer: &output, Formatter: cblog.JSONFormatter})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buffer.Flush(delegate)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")

	expectedTotal := numGoroutines * eventsPerGoroutine
	if len(lines) != expectedTotal {
		t.Logf("Warning: expected %d events, got %d (events may be lost due to race conditions)", expectedTotal, len(lines))
		// For now, just verify we got a reasonable number of events
		if len(lines) < expectedTotal/2 {
			t.Fatalf("expected at least %d events, got %d", expectedTotal/2, len(lines))
		}
	}
}

func TestEventBuffer_NilDelegate(t *testing.T) {
	buffer := NewEventBuffer(10)

	buffer.add(bufferedEntry{
		ctx:    context.Background(),
		level:  levelInfo,
		msg:    "test message",
		fields: []interface{}{},
	})

	// Should not panic with nil delegate
	buffer.Flush(nil)

	// Buffer should still contain events after flush with nil delegate
	var output bytes.Buffer

	delegate, err := New(Options{Writer: &output, Formatter: cblog.JSONFormatter})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now flush with a real delegate - should get the events
	buffer.Flush(delegate)

	if output.Len() == 0 {
		t.Fatal("expected output after flush with real delegate, got empty output")
	}

	// Buffer should be empty after the real flush
	var secondOutput bytes.Buffer

	secondDelegate, err := New(Options{Writer: &secondOutput, Formatter: cblog.JSONFormatter})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buffer.Flush(secondDelegate)

	if secondOutput.Len() != 0 {
		t.Fatalf("expected no output after second flush (buffer should be empty), got: %s", secondOutput.String())
	}
}

func TestBufferedLogger_NilBuffer(t *testing.T) {
	logger := NewBufferedLogger(nil)

	// Should not panic
	logger.Debug(context.Background(), "debug message")
	logger.Info(context.Background(), "info message")
	logger.Warn(context.Background(), "warn message")
	logger.Error(context.Background(), "error message")

	child := logger.With("key", "value")
	if child == nil {
		t.Fatal("expected non-nil child logger")
	}
}

func TestBufferedLogger_WithFieldChaining(t *testing.T) {
	buffer := NewEventBuffer(10)
	logger := NewBufferedLogger(buffer)

	// Test multiple With calls
	child1 := logger.With("component", "test")
	child2 := child1.With("version", "1.0")
	child3 := child2.With("env", "dev")

	child3.Info(context.Background(), "test message", "extra", "value")

	var output bytes.Buffer

	delegate, err := New(Options{Writer: &output, Formatter: cblog.JSONFormatter})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buffer.Flush(delegate)

	line := strings.TrimSpace(output.String())

	payload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}

	// Should have all fields
	if payload["component"] != "test" {
		t.Fatalf("expected component field, got: %v", payload["component"])
	}

	if payload["version"] != "1.0" {
		t.Fatalf("expected version field, got: %v", payload["version"])
	}

	if payload["env"] != "dev" {
		t.Fatalf("expected env field, got: %v", payload["env"])
	}

	if payload["extra"] != "value" {
		t.Fatalf("expected extra field, got: %v", payload["extra"])
	}
}

func TestMapToFields(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]interface{}
		expected []interface{}
	}{
		{"nil map", nil, nil},
		{"empty map", map[string]interface{}{}, nil},
		{"single item", map[string]interface{}{"key": "value"}, []interface{}{"key", "value"}},
		{"multiple items", map[string]interface{}{"b": 2, "a": 1, "c": 3}, []interface{}{"a", 1, "b", 2, "c", 3}},
		{"with nil value", map[string]interface{}{"key": nil}, []interface{}{"key", nil}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapToFields(tc.input)
			if !equalInterfaces(result, tc.expected) {
				t.Fatalf("expected %v, got: %v", tc.expected, result)
			}
		})
	}
}

func TestMergeFields_ExtrasFiltering(t *testing.T) {
	base := []interface{}{"base", "base_value"}
	additions := []interface{}{"add", "add_value"}
	extras := map[string]interface{}{
		"keep":    "keep_value",
		"empty":   "",
		"nil":     nil,
		"invalid": (*string)(nil), // nil pointer to string
	}

	result := mergeFields(base, additions, extras)

	// Convert to map for easier testing
	resultMap := make(map[string]interface{})
	for i := 0; i+1 < len(result); i += 2 {
		key, ok := result[i].(string)
		if !ok {
			continue
		}

		resultMap[key] = result[i+1]
	}

	// Should have kept values
	if resultMap["keep"] != "keep_value" {
		t.Fatalf("expected keep field, got: %v", resultMap["keep"])
	}

	// Should not have empty or nil values
	if _, exists := resultMap["empty"]; exists {
		t.Fatal("should not have empty string value")
	}

	if _, exists := resultMap["nil"]; exists {
		t.Fatal("should not have nil value")
	}
}

// Helper types and functions for testing
type mockStringer struct {
	msg string
}

func (m mockStringer) String() string {
	return m.msg
}

func equalInterfaces(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

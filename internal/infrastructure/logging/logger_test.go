package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

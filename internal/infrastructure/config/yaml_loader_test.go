package config

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	apperrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestDomainErrorFromContextErr(t *testing.T) {
	fields := map[string]interface{}{"path": "pipeline.yaml"}

	timeoutErr := domainErrorFromContextErr(context.DeadlineExceeded, "cancelled", "timed out", fields)
	if timeoutErr == nil {
		t.Fatalf("expected timeout error")
	}

	var domainErr *domain.DomainError
	if !errors.As(timeoutErr, &domainErr) || domainErr.Code != domain.ErrCodeTimeout {
		t.Fatalf("expected timeout domain error, got %v", timeoutErr)
	}

	if domainErr.Context["path"] != "pipeline.yaml" {
		t.Fatalf("expected context to include path, got %+v", domainErr.Context)
	}

	cancelErr := domainErrorFromContextErr(context.Canceled, "cancelled", "timed out", nil)
	if !errors.As(cancelErr, &domainErr) || domainErr.Code != domain.ErrCodeCancelled {
		t.Fatalf("expected cancelled domain error, got %v", cancelErr)
	}

	otherErr := domainErrorFromContextErr(errors.New("other"), "cancelled", "timed out", nil)
	if !errors.As(otherErr, &domainErr) || domainErr.Code != domain.ErrCodeCancelled {
		t.Fatalf("expected fallback cancellation error, got %v", otherErr)
	}

	if domainErrorFromContextErr(nil, "cancelled", "timed out", nil) != nil {
		t.Fatal("expected nil error when input error is nil")
	}
}

func TestContextDomainError(t *testing.T) {
	if err := contextDomainError(context.TODO(), "cancelled", "timed out", nil); err != nil {
		t.Fatalf("expected nil error for nil context, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := contextDomainError(ctx, "cancelled", "timed out", nil)

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != domain.ErrCodeCancelled {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}

func TestConvertErrorMapsDeadline(t *testing.T) {
	err := convertError(context.DeadlineExceeded, "config.yaml")

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != domain.ErrCodeTimeout {
		t.Fatalf("expected timeout domain error, got %v", err)
	}
}

func TestCtxAwareReaderHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reader := &ctxAwareReader{ctx: ctx, reader: strings.NewReader("data")}
	if _, err := reader.Read(make([]byte, 4)); err == nil {
		t.Fatal("expected cancellation error while reading")
	}
}

func TestCtxAwareReaderPassesThroughRead(t *testing.T) {
	ctx := context.Background()
	reader := &ctxAwareReader{ctx: ctx, reader: strings.NewReader("data")}

	buf := make([]byte, 4)

	n, err := reader.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("unexpected read error: %v", err)
	}

	if string(buf[:n]) != "data" {
		t.Fatalf("expected to read 'data', got %q", buf[:n])
	}
}

type capturingLogger struct {
	entries []struct {
		level  string
		msg    string
		fields []interface{}
	}
}

func (c *capturingLogger) Debug(_ context.Context, msg string, fields ...interface{}) {
	c.entries = append(c.entries, struct {
		level  string
		msg    string
		fields []interface{}
	}{"debug", msg, fields})
}

func (c *capturingLogger) Info(_ context.Context, msg string, fields ...interface{}) {
	c.entries = append(c.entries, struct {
		level  string
		msg    string
		fields []interface{}
	}{"info", msg, fields})
}

func (c *capturingLogger) Warn(_ context.Context, msg string, fields ...interface{}) {
	c.entries = append(c.entries, struct {
		level  string
		msg    string
		fields []interface{}
	}{"warn", msg, fields})
}

func (c *capturingLogger) Error(_ context.Context, msg string, fields ...interface{}) {
	c.entries = append(c.entries, struct {
		level  string
		msg    string
		fields []interface{}
	}{"error", msg, fields})
}

func (c *capturingLogger) With(_ ...interface{}) ports.Logger { return c }

func TestYAMLLoaderLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pipeline.yaml")

	yamlContent := `name: demo
steps:
  - id: setup
    type: command
    enabled: true
    command: echo hi
`
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	logger := &capturingLogger{}
	loader := NewYAMLLoader(logger)

	pipeline, err := loader.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load pipeline: %v", err)
	}

	if pipeline.Name != "demo" || len(pipeline.Steps) != 1 {
		t.Fatalf("unexpected pipeline: %+v", pipeline)
	}

	if err := loader.Validate(context.Background(), path); err != nil {
		t.Fatalf("validate: %v", err)
	}

	if len(logger.entries) == 0 {
		t.Fatal("expected logger to capture entries")
	}
}

func TestYAMLLoaderLoadCancelled(t *testing.T) {
	loader := NewYAMLLoader(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := loader.Load(ctx, "ignored.yaml"); err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestMapValidationReadError(t *testing.T) {
	domainErr := domain.NewDomainError(domain.ErrCodeValidation, "invalid", nil, nil)
	if err := mapValidationReadError(domainErr, "config.yaml"); !errors.Is(err, domainErr) {
		t.Fatalf("expected domain error to passthrough")
	}

	cancelErr := mapValidationReadError(context.Canceled, "config.yaml")

	var derr *domain.DomainError
	if !errors.As(cancelErr, &derr) || derr.Code != domain.ErrCodeCancelled {
		t.Fatalf("expected cancellation domain error, got %v", cancelErr)
	}

	timeoutErr := mapValidationReadError(context.DeadlineExceeded, "config.yaml")
	if !errors.As(timeoutErr, &derr) || derr.Code != domain.ErrCodeTimeout {
		t.Fatalf("expected timeout domain error, got %v", timeoutErr)
	}

	otherErr := mapValidationReadError(errors.New("io failure"), "config.yaml")
	if !errors.As(otherErr, &derr) || derr.Code != domain.ErrCodeValidation {
		t.Fatalf("expected validation domain error, got %v", otherErr)
	}
}

func TestContextCheck(t *testing.T) {
	require := func(err error) {
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	}

	require(contextCheck(context.TODO()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := contextCheck(ctx); err == nil {
		t.Fatal("expected cancellation error from contextCheck")
	}
}

func TestFlattenFields(t *testing.T) {
	fields := flattenFields(map[string]interface{}{"b": 2, "a": 1})
	if len(fields) != 4 {
		t.Fatalf("expected flattened key/value pairs, got %v", fields)
	}

	if fields[0] != "a" || fields[2] != "b" {
		t.Fatalf("expected keys to be sorted, got %v", fields)
	}
}

func TestConvertErrorParseAndValidation(t *testing.T) {
	parseErr := apperrors.NewParseError("config.yaml", 3, os.ErrNotExist)
	converted := convertError(parseErr, "config.yaml")

	var domainErr *domain.DomainError
	if !errors.As(converted, &domainErr) || domainErr.Code != domain.ErrCodeNotFound {
		t.Fatalf("expected not found error, got %v", converted)
	}

	valErr := apperrors.NewValidationError("field", "duplicate step id", errors.New("duplicate"))

	converted = convertError(valErr, "config.yaml")
	if !errors.As(converted, &domainErr) || domainErr.Code != domain.ErrCodeDuplicate {
		t.Fatalf("expected duplicate code, got %v", converted)
	}

	valErr = apperrors.NewValidationError("dependency", "depends on missing", errors.New("missing"))

	converted = convertError(valErr, "config.yaml")
	if !errors.As(converted, &domainErr) || domainErr.Code != domain.ErrCodeDependency {
		t.Fatalf("expected dependency code, got %v", converted)
	}
}

func TestExtractLine(t *testing.T) {
	err := errors.New("parse error: line 12: invalid mapping")
	if line := extractLine(err); line != 12 {
		t.Fatalf("expected line 12, got %d", line)
	}

	if line := extractLine(nil); line != 0 {
		t.Fatalf("expected zero line for nil error, got %d", line)
	}
}

func TestLogErrorCapturesFields(t *testing.T) {
	logger := &capturingLogger{}
	loader := &YAMLLoader{logger: logger}

	err := errors.New("boom")
	loader.logError(context.Background(), "failed", err, map[string]interface{}{"path": "config.yaml"})

	if len(logger.entries) == 0 {
		t.Fatal("expected log entry to be captured")
	}

	entry := logger.entries[len(logger.entries)-1]
	if entry.level != "error" || entry.msg != "failed" {
		t.Fatalf("unexpected log entry: %+v", entry)
	}
}

package main

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
	require "github.com/stretchr/testify/require"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	metricsinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/metrics"
	tracinginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/tracing"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type syncBufferWriter struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

func (w *syncBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.Write(p) //nolint:wrapcheck // test helper forwards writes to an in-memory buffer
}

func TestVerifyCommandStructuredLogging(t *testing.T) {
	var logBuf bytes.Buffer

	writer := &syncBufferWriter{buf: &logBuf}
	logger, err := logginginfra.New(logginginfra.Options{
		Writer:    writer,
		Formatter: cblog.JSONFormatter,
		Level:     "info",
		Layer:     "infrastructure",
		Component: "cli",
	})
	require.NoError(t, err)

	eventPublisher := &eventsRecordingPublisher{logger: logger}

	configLoader := &stubConfigLoader{
		pipeline: &domainpipeline.Pipeline{
			Name: "demo",
			Steps: []domainpipeline.Step{
				{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
			},
		},
	}
	dagBuilder := &stubDAGBuilder{}
	executor := &stubExecutor{
		verifyResults: []domainpipeline.VerificationResult{{
			StepID: "setup",
			Type:   "command",
			Status: domainpipeline.VerificationSatisfied,
		}},
	}
	validator := &stubValidationService{}

	tracer := tracinginfra.NewNoOpTracer()
	metrics := metricsinfra.NewNoOpCollector()

	prepareUseCase := applicationpipeline.NewPrepareUseCase(
		configLoader,
		dagBuilder,
		logger.With("component", "prepare_usecase"),
		tracer,
		eventPublisher,
	)
	applyUseCase := applicationpipeline.NewApplyUseCase(
		prepareUseCase,
		executor,
		validator,
		logger.With("component", "apply_usecase"),
		metrics,
		tracer,
		eventPublisher,
	)
	verifyUseCase := applicationpipeline.NewVerifyUseCase(
		prepareUseCase,
		executor,
		logger.With("component", "verify_usecase"),
		metrics,
		tracer,
		eventPublisher,
	)

	app := &AppContext{
		Logger:         logger,
		Events:         eventPublisher,
		PrepareUseCase: prepareUseCase,
		ApplyUseCase:   applyUseCase,
		VerifyUseCase:  verifyUseCase,
	}

	rootCmd := newRootCmd(app)

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"verify", "--json", "pipeline.yaml"})

	originalExit := exitFunc
	exitFunc = func(int) {}

	t.Cleanup(func() { exitFunc = originalExit })

	ctx := logginginfra.WithCorrelationID(context.Background(), "cli-corr-id")
	require.NoError(t, rootCmd.ExecuteContext(ctx))

	logLines := filterLines(logBuf.String())
	require.NotEmpty(t, logLines, "expected structured log output")

	for _, line := range logLines {
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(line), &payload))
		require.Equal(t, "cli-corr-id", payload["correlation_id"])

		layer, hasLayer := payload["layer"]
		require.True(t, hasLayer, "expected layer field in log entry")
		require.NotEmpty(t, layer)
	}

	require.NotEmpty(t, eventPublisher.events)

	for _, event := range eventPublisher.events {
		require.Equal(t, "cli-corr-id", event.correlationID)
		require.NotEmpty(t, event.eventType)
	}
}

func TestVerifyCommandStructuredLogging_WithFailure(t *testing.T) {
	var logBuf bytes.Buffer

	logger, err := logginginfra.New(logginginfra.Options{
		Writer:    &logBuf,
		Formatter: cblog.JSONFormatter,
		Level:     "info",
		Layer:     "infrastructure",
		Component: "cli",
	})
	require.NoError(t, err)

	eventPublisher := &eventsRecordingPublisher{logger: logger}

	configLoader := &stubConfigLoader{
		pipeline: &domainpipeline.Pipeline{
			Name: "demo",
			Steps: []domainpipeline.Step{
				{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
			},
		},
	}
	dagBuilder := &stubDAGBuilder{}
	rootErr := errors.New("plugin boom")
	execErr := fmt.Errorf("executor failed: %w", rootErr)
	verifyErr := fmt.Errorf("verify failed: %w", execErr)
	executor := &stubExecutor{verifyErr: verifyErr}
	validator := &stubValidationService{}

	tracer := tracinginfra.NewNoOpTracer()
	metrics := metricsinfra.NewNoOpCollector()

	prepareUseCase := applicationpipeline.NewPrepareUseCase(
		configLoader,
		dagBuilder,
		logger.With("component", "prepare_usecase"),
		tracer,
		eventPublisher,
	)
	applyUseCase := applicationpipeline.NewApplyUseCase(
		prepareUseCase,
		executor,
		validator,
		logger.With("component", "apply_usecase"),
		metrics,
		tracer,
		eventPublisher,
	)
	verifyUseCase := applicationpipeline.NewVerifyUseCase(
		prepareUseCase,
		executor,
		logger.With("component", "verify_usecase"),
		metrics,
		tracer,
		eventPublisher,
	)

	app := &AppContext{
		Logger:         logger,
		Events:         eventPublisher,
		PrepareUseCase: prepareUseCase,
		ApplyUseCase:   applyUseCase,
		VerifyUseCase:  verifyUseCase,
	}

	rootCmd := newRootCmd(app)

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"verify", "--json", "pipeline.yaml"})

	originalExit := exitFunc
	exitFunc = func(int) {}

	t.Cleanup(func() { exitFunc = originalExit })

	ctx := logginginfra.WithCorrelationID(context.Background(), "cli-corr-id")
	require.NoError(t, rootCmd.ExecuteContext(ctx))

	logLines := filterLines(logBuf.String())
	require.NotEmpty(t, logLines)

	var (
		errorSeen     bool
		observedChain []string
		observedCause string
	)

	for _, line := range logLines {
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(line), &payload))
		require.Equal(t, "cli-corr-id", payload["correlation_id"])
		require.NotEmpty(t, payload["layer"])

		if strings.Contains(line, "boom") {
			errorSeen = true

			if msg, ok := payload["msg"].(string); ok && msg == "pipeline verification failed" {
				rawChain, ok := payload["error_chain"].([]interface{})
				require.True(t, ok, "expected error_chain field in payload")

				for _, entry := range rawChain {
					str, ok := entry.(string)
					require.True(t, ok, "expected string in error_chain")

					observedChain = append(observedChain, str)
				}

				if cause, ok := payload["error_cause"].(string); ok {
					observedCause = cause
				}
			}
		}
	}

	require.True(t, errorSeen, "expected error log containing boom")
	require.Equal(t, []string{
		verifyErr.Error(),
		execErr.Error(),
		rootErr.Error(),
	}, observedChain)
	require.Equal(t, rootErr.Error(), observedCause)

	require.True(t, eventPublisher.contains(ports.EventValidationFailed))

	for _, event := range eventPublisher.events {
		require.Equal(t, "cli-corr-id", event.correlationID)
	}
}

func filterLines(output string) []string {
	raw := strings.Split(output, "\n")

	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	return lines
}

type eventsRecordingPublisher struct {
	logger ports.Logger
	events []eventRecord
	mu     sync.RWMutex
}

type eventRecord struct {
	eventType     string
	payload       map[string]interface{}
	correlationID string
}

func (e *eventsRecordingPublisher) Publish(ctx context.Context, event ports.DomainEvent) error {
	if event == nil {
		return nil
	}

	payload := map[string]interface{}{}
	if raw, ok := event.Payload().(map[string]interface{}); ok {
		payload = raw
	}

	record := eventRecord{
		eventType:     event.EventType(),
		payload:       payload,
		correlationID: ports.GetCorrelationID(ctx),
	}

	e.mu.Lock()
	e.events = append(e.events, record)
	e.mu.Unlock()

	if e.logger != nil {
		e.logger.Info(ctx, "test event", "event_type", record.eventType)
	}

	return nil
}

func (*eventsRecordingPublisher) Subscribe(string, ports.EventHandler) (ports.Subscription, error) {
	return noopSubscription{}, nil
}

func (e *eventsRecordingPublisher) contains(eventType string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, evt := range e.events {
		if evt.eventType == eventType {
			return true
		}
	}

	return false
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() {}

type stubConfigLoader struct {
	pipeline *domainpipeline.Pipeline
	err      error
}

func (s *stubConfigLoader) Load(context.Context, string) (*domainpipeline.Pipeline, error) {
	return s.pipeline, s.err
}

func (s *stubConfigLoader) Validate(context.Context, string) error { return nil }

type stubDAGBuilder struct{}

func (stubDAGBuilder) Build(context.Context, []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
	return &domainpipeline.ExecutionPlan{
		Levels:     []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
		TotalSteps: 1,
	}, nil
}

type stubExecutor struct {
	results       []domainpipeline.StepResult
	err           error
	verifyResults []domainpipeline.VerificationResult
	verifyErr     error
}

func (s *stubExecutor) Execute(context.Context, *domainpipeline.ExecutionPlan, *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
	return s.results, s.err
}

func (s *stubExecutor) Verify(context.Context, *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
	return s.verifyResults, s.verifyErr
}

type stubValidationService struct{}

func (stubValidationService) RunValidations(context.Context, []domainpipeline.Validation) (domainpipeline.VerificationSummary, error) {
	return domainpipeline.VerificationSummary{}, nil
}

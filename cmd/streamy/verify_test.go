package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/application/pipeline/testutil"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/pipelineconv"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type bufferPrinter struct {
	buf *bytes.Buffer
}

func (p *bufferPrinter) Printf(format string, args ...interface{}) {
	if p == nil {
		return
	}
	_, _ = fmt.Fprintf(p.buf, format, args...)
}

func (p *bufferPrinter) Println(args ...interface{}) {
	if p == nil {
		return
	}
	_, _ = fmt.Fprintln(p.buf, args...)
}

func withTestWriters(t *testing.T) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	origStdout := stdoutWriter
	origStderr := stderrWriter

	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	stdoutWriter = &bufferPrinter{buf: stdoutBuf}
	stderrWriter = stderrBuf

	t.Cleanup(func() {
		stdoutWriter = origStdout
		stderrWriter = origStderr
	})

	return stdoutBuf, stderrBuf
}

type verifySummaryRecorder struct {
	summary *pipelineconv.VerificationSummary
}

func (r *verifySummaryRecorder) Summary() *pipelineconv.VerificationSummary {
	return r.summary
}

func withPrintHooks(t *testing.T) *verifySummaryRecorder {
	t.Helper()
	recorder := &verifySummaryRecorder{}
	origTable := printTableOutputFunc
	origVerbose := printVerboseOutputFunc
	origJSON := printJSONOutputFunc

	printTableOutputFunc = func(summary *pipelineconv.VerificationSummary) {
		recorder.summary = summary
	}
	printVerboseOutputFunc = func(summary *pipelineconv.VerificationSummary) {
		recorder.summary = summary
	}
	printJSONOutputFunc = func(summary *pipelineconv.VerificationSummary, _ string) error {
		recorder.summary = summary
		return nil
	}

	t.Cleanup(func() {
		printTableOutputFunc = origTable
		printVerboseOutputFunc = origVerbose
		printJSONOutputFunc = origJSON
	})

	return recorder
}

func newPreparedApp(t *testing.T, pipeline *domainpipeline.Pipeline, verifyResults []domainpipeline.VerificationResult, verifyErr error) *AppContext {
	t.Helper()

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
			return pipeline, nil
		},
	}

	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(ctx context.Context, steps []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}}}, nil
		},
	}

	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}
	tracer := testutil.NewMockTracer()
	metrics := testutil.NewMockMetricsCollector()

	prepare := applicationpipeline.NewPrepareUseCase(loader, builder, logger, tracer, events)

	executor := &testutil.MockPluginExecutor{
		VerifyFunc: func(ctx context.Context, pip *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
			return verifyResults, verifyErr
		},
	}

	verify := applicationpipeline.NewVerifyUseCase(prepare, executor, logger, metrics, tracer, events)

	return &AppContext{
		Logger:         logger,
		PrepareUseCase: prepare,
		VerifyUseCase:  verify,
	}
}

func TestRunVerifyInternal_Success(t *testing.T) {
	ctx := context.Background()
	pipeline := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{{
			ID:      "setup",
			Type:    domainpipeline.StepTypeCommand,
			Enabled: true,
		}},
	}
	results := []domainpipeline.VerificationResult{{
		StepID:  "setup",
		Status:  domainpipeline.VerificationSatisfied,
		Message: "ok",
	}}

	app := newPreparedApp(t, pipeline, results, nil)

	_, stderrBuf := withTestWriters(t)
	recorder := withPrintHooks(t)

	exitCode, err := runVerifyInternal(ctx, app, verifyOptions{ConfigPath: "pipeline.yaml"})

	if err != nil || exitCode != 0 {
		t.Fatalf("unexpected result: exit=%d err=%v stderr=%q", exitCode, err, stderrBuf.String())
	}
	require.NotNil(t, recorder.Summary())
	require.Equal(t, 1, recorder.Summary().TotalSteps)
	require.Equal(t, 1, recorder.Summary().Satisfied)
	require.Equal(t, "setup", recorder.Summary().Results[0].StepID)
	require.Equal(t, 0, stderrBuf.Len())
}

func TestRunVerifyInternal_JSONOutputError(t *testing.T) {
	ctx := context.Background()
	pipeline := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{{
			ID:      "setup",
			Type:    domainpipeline.StepTypeCommand,
			Enabled: true,
		}},
	}
	results := []domainpipeline.VerificationResult{{
		StepID:  "setup",
		Status:  domainpipeline.VerificationSatisfied,
		Message: "ok",
	}}

	app := newPreparedApp(t, pipeline, results, nil)

	_, stderrBuf := withTestWriters(t)

	origJSON := printJSONOutputFunc
	printJSONOutputFunc = func(summary *pipelineconv.VerificationSummary, _ string) error {
		return errors.New("encode failure")
	}
	t.Cleanup(func() { printJSONOutputFunc = origJSON })

	exitCode, err := runVerifyInternal(ctx, app, verifyOptions{ConfigPath: "pipeline.yaml", JSON: true})

	require.NoError(t, err)
	require.Equal(t, 3, exitCode)
	require.Contains(t, stderrBuf.String(), "Failed to generate JSON output")
}

func TestRunVerifyInternal_PrepareFailure(t *testing.T) {
	ctx := context.Background()
	parseErr := streamyerrors.NewParseError("pipeline.yaml", 0, errors.New("bad yaml"))

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
			return nil, parseErr
		},
	}
	builder := &testutil.MockDAGBuilder{}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}
	tracer := testutil.NewMockTracer()
	metrics := testutil.NewMockMetricsCollector()

	prepare := applicationpipeline.NewPrepareUseCase(loader, builder, logger, tracer, events)
	executor := &testutil.MockPluginExecutor{
		VerifyFunc: func(ctx context.Context, pip *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
			t.Fatalf("verify should not be invoked on prepare failure")
			return nil, nil
		},
	}
	verify := applicationpipeline.NewVerifyUseCase(prepare, executor, logger, metrics, tracer, events)

	app := &AppContext{PrepareUseCase: prepare, VerifyUseCase: verify}

	_, stderrBuf := withTestWriters(t)

	exitCode, err := runVerifyInternal(ctx, app, verifyOptions{ConfigPath: "pipeline.yaml"})

	require.NoError(t, err)
	require.Equal(t, 2, exitCode)
	require.Contains(t, stderrBuf.String(), "Error parsing configuration")
}

func TestRunVerifyInternal_VerifyFailure(t *testing.T) {
	ctx := context.Background()
	pipeline := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{{
			ID:      "setup",
			Type:    domainpipeline.StepTypeCommand,
			Enabled: true,
		}},
	}

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
			return pipeline, nil
		},
	}
	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(ctx context.Context, steps []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}}}, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}
	tracer := testutil.NewMockTracer()
	metrics := testutil.NewMockMetricsCollector()

	prepare := applicationpipeline.NewPrepareUseCase(loader, builder, logger, tracer, events)
	execErr := streamyerrors.NewValidationError("field", "invalid", nil)
	executor := &testutil.MockPluginExecutor{
		VerifyFunc: func(ctx context.Context, pip *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
			return nil, execErr
		},
	}
	verify := applicationpipeline.NewVerifyUseCase(prepare, executor, logger, metrics, tracer, events)

	app := &AppContext{PrepareUseCase: prepare, VerifyUseCase: verify}

	_, stderrBuf := withTestWriters(t)

	exitCode, err := runVerifyInternal(ctx, app, verifyOptions{ConfigPath: "pipeline.yaml"})

	require.NoError(t, err)
	require.Equal(t, 2, exitCode)
	require.Contains(t, stderrBuf.String(), "Configuration error")
}

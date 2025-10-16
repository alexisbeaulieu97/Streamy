package pipeline

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutor is a mock implementation of the executor interface for testing.
type mockExecutor struct {
	VerifyFunc func(*engine.ExecutionContext, []config.Step, time.Duration) (*model.VerificationSummary, error)
}

func (m *mockExecutor) VerifySteps(ctx *engine.ExecutionContext, steps []config.Step, timeout time.Duration) (*model.VerificationSummary, error) {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(ctx, steps, timeout)
	}
	return nil, errors.New("VerifySteps mock not implemented")
}

func TestService_Prepare(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
	}{
		{
			name: "valid config should prepare successfully",
			configContent: `
version: 0.1.0
name: test
steps:
  - id: step1
    type: command
    command: echo 'hello'
`,
			expectError: false,
		},
		{
			name:          "non-existent config file should fail",
			configContent: "",
			expectError:   true,
		},
		{
			name:          "invalid yaml should fail parsing",
			configContent: `steps: [`,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			if tt.configContent != "" {
				tempDir := t.TempDir()
				configPath = filepath.Join(tempDir, "streamy.yaml")
				err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
				require.NoError(t, err)
			} else {
				configPath = "/path/to/non/existent/file.yaml"
			}

			svc := NewService(nil) // Registry not needed for Prepare
			prepared, err := svc.Prepare(configPath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, prepared)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, prepared)
				assert.Equal(t, configPath, prepared.Path)
				assert.NotNil(t, prepared.Config)
				assert.NotNil(t, prepared.Graph)
				assert.NotNil(t, prepared.Plan)
			}
		})
	}
}

func TestService_Verify(t *testing.T) {
	// Setup a valid prepared pipeline for reuse
	cfg := &config.Config{
		Steps: []config.Step{{ID: "step1", Type: "command"}},
	}
	graph, _ := engine.BuildDAG(cfg.Steps)
	plan, _ := engine.GeneratePlan(graph)
	prepared := &PreparedPipeline{
		Path:   "/fake/path.yaml",
		Config: cfg,
		Graph:  graph,
		Plan:   plan,
	}

	log, err := logger.New(logger.Options{Writer: io.Discard})
	require.NoError(t, err)

	tests := []struct {
		name        string
		req         VerifyRequest
		expectError bool
		mockVerify  func(*engine.ExecutionContext, []config.Step, time.Duration) (*model.VerificationSummary, error)
	}{
		{
			name: "successful verification",
			req: VerifyRequest{
				Prepared: prepared,
				Logger:   log,
			},
			expectError: false,
			mockVerify: func(ctx *engine.ExecutionContext, steps []config.Step, d time.Duration) (*model.VerificationSummary, error) {
				return &model.VerificationSummary{TotalSteps: 1}, nil
			},
		},
		{
			name: "verification fails with error",
			req: VerifyRequest{
				Prepared: prepared,
				Logger:   log,
			},
			expectError: true,
			mockVerify: func(ctx *engine.ExecutionContext, steps []config.Step, d time.Duration) (*model.VerificationSummary, error) {
				return &model.VerificationSummary{TotalSteps: 1}, errors.New("verify boom")
			},
		},
		{
			name: "nil logger should fail",
			req: VerifyRequest{
				Prepared: prepared,
				Logger:   nil,
			},
			expectError: true,
		},
		{
			name: "unprepared pipeline with no path should fail",
			req: VerifyRequest{
				Prepared:   nil,
				ConfigPath: "",
				Logger:     log,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&plugin.PluginRegistry{}) // Pass a dummy registry
			svc.newExecutor = func(l *logger.Logger) executor {
				return &mockExecutor{
					VerifyFunc: tt.mockVerify,
				}
			}

			outcome, err := svc.Verify(context.Background(), tt.req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, outcome)
				assert.NotNil(t, outcome.Summary)
			}
		})
	}
}

func TestService_Apply(t *testing.T) {
	// Setup a valid prepared pipeline for reuse
	cfg := &config.Config{
		Steps: []config.Step{{ID: "step1", Type: "command"}},
	}
	graph, _ := engine.BuildDAG(cfg.Steps)
	plan, _ := engine.GeneratePlan(graph)
	prepared := &PreparedPipeline{
		Path:   "/fake/path.yaml",
		Config: cfg,
		Graph:  graph,
		Plan:   plan,
	}

	log, err := logger.New(logger.Options{Writer: io.Discard})
	require.NoError(t, err)

	tests := []struct {
		name        string
		req         ApplyRequest
		expectError bool
		mockApply   func(*engine.ExecutionContext, *engine.ExecutionPlan) ([]model.StepResult, error)
	}{
		{
			name: "successful apply",
			req: ApplyRequest{
				Prepared: prepared,
				Logger:   log,
			},
			expectError: false,
			mockApply: func(ctx *engine.ExecutionContext, plan *engine.ExecutionPlan) ([]model.StepResult, error) {
				return []model.StepResult{{StepID: "step1", Status: model.StatusSuccess}}, nil
			},
		},
		{
			name: "apply fails with error",
			req: ApplyRequest{
				Prepared: prepared,
				Logger:   log,
			},
			expectError: true,
			mockApply: func(ctx *engine.ExecutionContext, plan *engine.ExecutionPlan) ([]model.StepResult, error) {
				return nil, errors.New("apply boom")
			},
		},
		{
			name: "nil logger should fail",
			req: ApplyRequest{
				Prepared: prepared,
				Logger:   nil,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&plugin.PluginRegistry{})
			svc.executePlan = tt.mockApply

			outcome, err := svc.Apply(context.Background(), tt.req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, outcome)
				assert.Len(t, outcome.Results, 1)
			}
		})
	}
}

package dashboard

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type stubPipelineService struct {
	verifyResult *registry.ExecutionResult
	verifyErr    error
	applyResult  *registry.ExecutionResult
	applyErr     error
}

func (s *stubPipelineService) Verify(ctx context.Context, opts VerifyOptions) (*registry.ExecutionResult, error) {
	if opts.Timeout > 0 {
		// simulate timeout handling by respecting context deadline
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}
	return s.verifyResult, s.verifyErr
}

func (s *stubPipelineService) Apply(ctx context.Context, opts ApplyOptions) (*registry.ExecutionResult, error) {
	return s.applyResult, s.applyErr
}

func TestLoadInitialStatusCmd(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1"},
		{ID: "test-2", Name: "Test 2"},
	}

	cmd := loadInitialStatusCmd(pipelines, cache)
	assert.NotNil(t, cmd)

	// Execute command
	msg := cmd()
	assert.NotNil(t, msg)

	// Should return InitialStatusLoadedMsg
	loadedMsg, ok := msg.(InitialStatusLoadedMsg)
	assert.True(t, ok)
	assert.NotNil(t, loadedMsg.Statuses)
}

func TestVerifyCmd(t *testing.T) {
	ctx := context.Background()
	result := &registry.ExecutionResult{Operation: "verify", Status: registry.StatusSatisfied}
	svc := &stubPipelineService{verifyResult: result}

	cmd := verifyCmd(ctx, "test-1", "/tmp/config.yaml", svc)
	assert.NotNil(t, cmd)

	// Execute command
	msg := cmd()
	assert.NotNil(t, msg)

	// Should return either VerifyCompleteMsg or VerifyErrorMsg
	switch msg.(type) {
	case VerifyCompleteMsg:
		// Success case
	case VerifyErrorMsg:
		// Error case
	default:
		t.Fatalf("Unexpected message type: %T", msg)
	}
}

func TestApplyCmd(t *testing.T) {
	ctx := context.Background()
	result := &registry.ExecutionResult{Operation: "apply", Status: registry.StatusSatisfied}
	svc := &stubPipelineService{applyResult: result}

	cmd := applyCmd(ctx, "test-1", "/tmp/config.yaml", svc)
	assert.NotNil(t, cmd)

	// Execute command
	msg := cmd()
	assert.NotNil(t, msg)

	// Should return either ApplyCompleteMsg or ApplyErrorMsg
	switch msg.(type) {
	case ApplyCompleteMsg:
		// Success case
	case ApplyErrorMsg:
		// Error case (expected since /tmp/src doesn't exist)
	default:
		t.Fatalf("Unexpected message type: %T", msg)
	}
}

func TestRefreshAllCmd(t *testing.T) {
	ctx := context.Background()
	svc := &stubPipelineService{}

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Path: "/tmp/test1.yaml"},
		{ID: "test-2", Name: "Test 2", Path: "/tmp/test2.yaml"},
	}

	cmd := refreshAllCmd(ctx, pipelines, svc)
	assert.NotNil(t, cmd)

	// Execute command - this returns RefreshStartedMsg
	msg := cmd()
	assert.NotNil(t, msg)

	// Should return RefreshStartedMsg
	startedMsg, ok := msg.(RefreshStartedMsg)
	assert.True(t, ok)
	assert.Equal(t, 2, startedMsg.Total)
}

func TestRefreshSingleCmd(t *testing.T) {
	ctx := context.Background()
	result := &registry.ExecutionResult{Operation: "verify", Status: registry.StatusSatisfied}
	svc := &stubPipelineService{verifyResult: result}

	pipeline := registry.Pipeline{
		ID:   "test-1",
		Name: "Test 1",
		Path: "/tmp/config.yaml",
	}

	cmd := refreshSingleCmd(ctx, pipeline, svc, 0, 1)
	assert.NotNil(t, cmd)

	// Execute command
	msg := cmd()
	assert.NotNil(t, msg)

	// Should return RefreshPipelineCompleteMsg
	completeMsg, ok := msg.(RefreshPipelineCompleteMsg)
	assert.True(t, ok)
	assert.Equal(t, "test-1", completeMsg.PipelineID)
	assert.Equal(t, 0, completeMsg.Index)
	assert.Equal(t, 1, completeMsg.Total)
}

func TestVerifyCmd_InvalidPipeline(t *testing.T) {
	svc := &stubPipelineService{verifyErr: errors.New("boom")}
	ctx := context.Background()

	cmd := verifyCmd(ctx, "test-1", "/nonexistent/path.yaml", svc)
	assert.NotNil(t, cmd)

	msg := cmd()
	assert.NotNil(t, msg)

	// Should return VerifyErrorMsg for nonexistent file
	errMsg, ok := msg.(VerifyErrorMsg)
	assert.True(t, ok)
	assert.Equal(t, "test-1", errMsg.PipelineID)
	assert.Error(t, errMsg.Error)
}

func TestApplyCmd_InvalidPipeline(t *testing.T) {
	svc := &stubPipelineService{applyErr: errors.New("boom")}
	ctx := context.Background()

	cmd := applyCmd(ctx, "test-1", "/nonexistent/path.yaml", svc)
	assert.NotNil(t, cmd)

	msg := cmd()
	assert.NotNil(t, msg)

	// Should return ApplyErrorMsg for nonexistent file
	errMsg, ok := msg.(ApplyErrorMsg)
	assert.True(t, ok)
	assert.Equal(t, "test-1", errMsg.PipelineID)
	assert.Error(t, errMsg.Error)
}

func TestRefreshAllCmd_EmptyPipelines(t *testing.T) {
	ctx := context.Background()
	svc := &stubPipelineService{}

	pipelines := []registry.Pipeline{}

	cmd := refreshAllCmd(ctx, pipelines, svc)
	assert.NotNil(t, cmd)

	// Execute command
	msg := cmd()
	assert.NotNil(t, msg)

	// Should return RefreshStartedMsg with 0 total
	startedMsg, ok := msg.(RefreshStartedMsg)
	assert.True(t, ok)
	assert.Equal(t, 0, startedMsg.Total)
}

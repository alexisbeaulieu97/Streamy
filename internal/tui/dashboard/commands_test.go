package dashboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pipelineapp "github.com/alexisbeaulieu97/streamy/internal/app/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

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
	tmpDir := t.TempDir()

	// Create a test pipeline config
	configPath := filepath.Join(tmpDir, "test.yaml")
	testConfig := `version: 1
steps:
  - name: test-step
    plugin: copy
    config:
      source: /tmp/src
      dest: /tmp/dst
`
	require.NoError(t, os.WriteFile(configPath, []byte(testConfig), 0644))

	pluginReg := plugin.NewPluginRegistry(&plugin.RegistryConfig{}, nil)
	svc := pipelineapp.NewService(pluginReg)
	ctx := context.Background()

	cmd := verifyCmd(ctx, "test-1", configPath, svc)
	assert.NotNil(t, cmd)

	// Execute command
	msg := cmd()
	assert.NotNil(t, msg)

	// Should return either VerifyCompleteMsg or VerifyErrorMsg
	switch msg.(type) {
	case VerifyCompleteMsg:
		// Success case
	case VerifyErrorMsg:
		// Error case (expected since /tmp/src doesn't exist)
	default:
		t.Fatalf("Unexpected message type: %T", msg)
	}
}

func TestApplyCmd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test pipeline config
	configPath := filepath.Join(tmpDir, "test.yaml")
	testConfig := `version: 1
steps:
  - name: test-step
    plugin: copy
    config:
      source: /tmp/src
      dest: /tmp/dst
`
	require.NoError(t, os.WriteFile(configPath, []byte(testConfig), 0644))

	pluginReg := plugin.NewPluginRegistry(&plugin.RegistryConfig{}, nil)
	svc := pipelineapp.NewService(pluginReg)
	ctx := context.Background()

	cmd := applyCmd(ctx, "test-1", configPath, svc)
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
	pluginReg := plugin.NewPluginRegistry(&plugin.RegistryConfig{}, nil)
	svc := pipelineapp.NewService(pluginReg)

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
	tmpDir := t.TempDir()

	// Create a test pipeline config
	configPath := filepath.Join(tmpDir, "test.yaml")
	testConfig := `version: 1
steps:
  - name: test-step
    plugin: copy
    config:
      source: /tmp/src
      dest: /tmp/dst
`
	require.NoError(t, os.WriteFile(configPath, []byte(testConfig), 0644))

	pluginReg := plugin.NewPluginRegistry(&plugin.RegistryConfig{}, nil)
	svc := pipelineapp.NewService(pluginReg)
	ctx := context.Background()

	pipeline := registry.Pipeline{
		ID:   "test-1",
		Name: "Test 1",
		Path: configPath,
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
	pluginReg := plugin.NewPluginRegistry(&plugin.RegistryConfig{}, nil)
	svc := pipelineapp.NewService(pluginReg)
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
	pluginReg := plugin.NewPluginRegistry(&plugin.RegistryConfig{}, nil)
	svc := pipelineapp.NewService(pluginReg)
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
	pluginReg := plugin.NewPluginRegistry(&plugin.RegistryConfig{}, nil)
	svc := pipelineapp.NewService(pluginReg)

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

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	require "github.com/stretchr/testify/require"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	configinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/config"
	engineinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/engine"
	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	tracinginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/tracing"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestRunAddRegistersPipelineUsingPrepareUseCase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, "sample-pipeline.yaml")
	configContent := `id: sample-pipeline
version: "1.0"
name: "Sample Pipeline"
steps:
  - id: greet
    type: command
    command: "echo hello"
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	logger := logginginfra.NewNoOpLogger()
	loader := configinfra.NewYAMLLoader(logger)
	dagBuilder := engineinfra.NewDAGBuilder()
	tracer := tracinginfra.NewNoOpTracer()

	prepare := applicationpipeline.NewPrepareUseCase(loader, dagBuilder, logger, tracer, nil)
	app := &AppContext{
		Logger:         logger,
		PrepareUseCase: prepare,
	}

	cmd := &cobra.Command{}

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	opts := &addOptions{}
	err := runAdd(context.Background(), logger, app, cmd, configPath, opts)
	require.NoError(t, err)

	registryPath, err := defaultRegistryPath()
	require.NoError(t, err)
	reg, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)

	pipelines := reg.List()
	require.Len(t, pipelines, 1)
	require.Equal(t, "sample-pipeline@1.0", pipelines[0].ID)
	require.Equal(t, "Sample Pipeline", pipelines[0].Name)
	require.Equal(t, configPath, pipelines[0].Path)
	require.Empty(t, pipelines[0].Dependencies)
}

func TestRunAddInvalidConfigFailsValidation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, "invalid.yaml")
	invalidContent := `id: broken
version: "1.0"
name: "Broken"
steps: []`
	require.NoError(t, os.WriteFile(configPath, []byte(invalidContent), 0o644))

	logger := logginginfra.NewNoOpLogger()
	loader := configinfra.NewYAMLLoader(logger)
	dagBuilder := engineinfra.NewDAGBuilder()
	tracer := tracinginfra.NewNoOpTracer()

	prepare := applicationpipeline.NewPrepareUseCase(loader, dagBuilder, logger, tracer, nil)
	app := &AppContext{
		Logger:         logger,
		PrepareUseCase: prepare,
	}

	cmd := &cobra.Command{}

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	opts := &addOptions{}
	err := runAdd(context.Background(), logger, app, cmd, configPath, opts)
	require.Error(t, err)

	registryPath, pathErr := defaultRegistryPath()
	require.NoError(t, pathErr)

	_, statErr := os.Stat(registryPath)
	require.True(t, os.IsNotExist(statErr), "registry file should not be created on validation failure")
}

func TestRunAddWarnsOnMissingDependencies(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, "frontend.yaml")
	configContent := `id: frontend
version: "1.0"
name: "Frontend Deployment"
dependencies:
  - backend@2.0
  - network@1.3
steps:
  - id: deploy
    type: command
    command: "./deploy.sh"
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	logger := logginginfra.NewNoOpLogger()
	loader := configinfra.NewYAMLLoader(logger)
	dagBuilder := engineinfra.NewDAGBuilder()
	tracer := tracinginfra.NewNoOpTracer()

	prepare := applicationpipeline.NewPrepareUseCase(loader, dagBuilder, logger, tracer, nil)
	app := &AppContext{
		Logger:         logger,
		PrepareUseCase: prepare,
	}

	cmd := &cobra.Command{}

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	opts := &addOptions{}
	err := runAdd(context.Background(), logger, app, cmd, configPath, opts)
	require.NoError(t, err)

	outStr := output.String()
	require.Contains(t, outStr, "⚠ Warning: unregistered dependencies: [backend@2.0 network@1.3]")
	require.Contains(t, outStr, "Status: 🟠 BLOCKED")
	require.Contains(t, outStr, "Missing dependencies:\n    - backend@2.0\n    - network@1.3")

	registryPath, err := defaultRegistryPath()
	require.NoError(t, err)

	regFile, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)

	pipelines := regFile.List()
	require.Len(t, pipelines, 1)
	require.Equal(t, "frontend@1.0", pipelines[0].ID)
	require.Equal(t, []string{"backend@2.0", "network@1.3"}, pipelines[0].Dependencies)
}

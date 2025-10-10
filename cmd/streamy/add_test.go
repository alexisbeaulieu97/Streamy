package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestAddCommand_Success(t *testing.T) {
	home := setupAddHome(t)
	configPath := writeAddConfig(t, "pipeline.yaml")

	stdout, err := executeAddCommand(configPath, "--id", "dev-setup", "--name", "Development Setup", "--description", "Test pipeline")
	require.NoError(t, err)
	require.Contains(t, stdout, "dev-setup")
	require.Contains(t, stdout, "Added pipeline")

	reg := loadAddRegistry(t, filepath.Join(home, ".streamy", "registry.json"))
	pipelines := reg.List()
	require.Len(t, pipelines, 1)
	require.Equal(t, "dev-setup", pipelines[0].ID)
	require.Equal(t, "Development Setup", pipelines[0].Name)
	require.Contains(t, pipelines[0].Description, "Test pipeline")
	require.Equal(t, absAddPath(t, configPath), pipelines[0].Path)
	require.WithinDuration(t, time.Now(), pipelines[0].RegisteredAt, 5*time.Second)
}

func TestAddCommand_DuplicateID(t *testing.T) {
	home := setupAddHome(t)
	configPath := writeAddConfig(t, "pipeline.yaml")
	registryPath := filepath.Join(home, ".streamy", "registry.json")

	seedAddRegistry(t, registryPath, registry.Pipeline{ID: "dev-setup", Name: "Existing", Path: "/tmp/existing.yaml", RegisteredAt: time.Now()})

	_, err := executeAddCommand(configPath, "--id", "dev-setup", "--name", "Development Setup")
	require.Error(t, err)
	require.Contains(t, err.Error(), "dev-setup")
}

func TestAddCommand_InvalidConfig(t *testing.T) {
	setupAddHome(t)
	invalidConfig := writeAddRawFile(t, "invalid.yaml", []byte("invalid: yaml: content: ["))

	_, err := executeAddCommand(invalidConfig, "--id", "invalid-config", "--name", "Invalid Config")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to add: validating configuration")
}

func TestAddCommand_GeneratesID(t *testing.T) {
	home := setupAddHome(t)
	configPath := writeAddConfig(t, "My Pipeline.yaml")

	stdout, err := executeAddCommand(configPath)
	require.NoError(t, err)
	require.Contains(t, stdout, "my-pipeline")

	reg := loadAddRegistry(t, filepath.Join(home, ".streamy", "registry.json"))
	require.Len(t, reg.List(), 1)
	require.Equal(t, "my-pipeline", reg.List()[0].ID)
}

func executeAddCommand(configPath string, extraArgs ...string) (string, error) {
	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)

	args := append([]string{"registry", "add"}, append(extraArgs, configPath)...)
	root.SetArgs(args)

	err := root.Execute()
	return buf.String(), err
}

func setupAddHome(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeAddConfig(t *testing.T, name string) string {
	content := []byte(`version: "1.0"
name: test
settings:
  parallel: 1
steps:
  - id: test_step
    type: command
    command: "echo test"
`)
	return writeAddRawFile(t, name, content)
}

func writeAddRawFile(t *testing.T, name string, data []byte) string {
	home := os.Getenv("HOME")
	path := filepath.Join(home, "configs", name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

func loadAddRegistry(t *testing.T, path string) *registry.Registry {
	reg, err := registry.NewRegistry(path)
	require.NoError(t, err)
	require.NoError(t, reg.Load())
	return reg
}

func seedAddRegistry(t *testing.T, path string, pipelines ...registry.Pipeline) {
	file := registry.RegistryFile{Version: "1.0", Pipelines: pipelines}
	data, err := json.MarshalIndent(file, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func absAddPath(t *testing.T, path string) string {
	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	return abs
}

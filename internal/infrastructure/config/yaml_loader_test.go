package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
)

func TestYAMLLoaderLoadSuccess(t *testing.T) {
	loader := newTestLoader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "pipeline.yaml")

	yamlContent := `version: "1.0"
name: "demo"
steps:
  - id: "setup"
    type: "command"
    enabled: true
    command: "echo hi"
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	pip, err := loader.Load(ctx, configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pip == nil {
		t.Fatal("expected pipeline, got nil")
	}
	if pip.Name != "demo" {
		t.Fatalf("expected name demo, got %s", pip.Name)
	}
	if len(pip.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(pip.Steps))
	}
	if pip.Steps[0].Config["command"] != "echo hi" {
		t.Fatalf("expected command config to be preserved")
	}
}

func TestYAMLLoaderLoadMissingFile(t *testing.T) {
	loader := newTestLoader()
	ctx := context.Background()

	_, err := loader.Load(ctx, "does-not-exist.yaml")
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
	assertDomainError(t, err, pipeline.ErrCodeNotFound)
}

func TestYAMLLoaderLoadParseError(t *testing.T) {
	loader := newTestLoader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bad.yaml")

	if err := os.WriteFile(configPath, []byte("version: ["), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := loader.Load(ctx, configPath)
	if err == nil {
		t.Fatalf("expected parse error")
	}
	assertDomainError(t, err, pipeline.ErrCodeValidation)
}

func TestYAMLLoaderLoadDomainValidationError(t *testing.T) {
	loader := newTestLoader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	yamlContent := `version: "1.0"
name: "demo"
steps:
  - id: "duplicate"
    type: "command"
    enabled: true
    command: "echo hi"
  - id: "duplicate"
    type: "command"
    enabled: true
    command: "echo bye"
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := loader.Load(ctx, configPath)
	if err == nil {
		t.Fatalf("expected domain validation error")
	}
	assertDomainError(t, err, pipeline.ErrCodeDuplicate)
}

func TestYAMLLoaderLoadCancelled(t *testing.T) {
	loader := newTestLoader()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := loader.Load(ctx, "whatever.yaml")
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	assertDomainError(t, err, pipeline.ErrCodeCancelled)
}

func TestYAMLLoaderValidate(t *testing.T) {
	loader := newTestLoader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "pipeline.yaml")

	yamlContent := `version: "1.0"
name: "demo"
steps:
  - id: "setup"
    type: "command"
    enabled: true
    command: "echo hi"
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := loader.Validate(ctx, configPath); err != nil {
		t.Fatalf("expected validate success, got %v", err)
	}
}

func assertDomainError(t *testing.T, err error, code pipeline.ErrorCode) {
	t.Helper()
	var domainErr *pipeline.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != code {
		t.Fatalf("expected code %s, got %s", code, domainErr.Code)
	}
}

func newTestLoader() *YAMLLoader {
	return NewYAMLLoader(logging.NewNoOpLogger())
}

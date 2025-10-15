package symlinkplugin

import (
	"context"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestSymlinkBasicFunctionality(t *testing.T) {
	p := New()

	// Test that plugin implements the interface
	if p == nil {
		t.Fatal("Plugin constructor returned nil")
	}

	// Test metadata
	meta := p.PluginMetadata()
	if meta.Name != "symlink" {
		t.Errorf("Expected plugin name 'symlink', got '%s'", meta.Name)
	}

	// Test schema
	schema := p.Schema()
	if schema == nil {
		t.Fatal("Schema returned nil")
	}

	// Test basic evaluation
	step := &config.Step{ID: "test-symlink"}
	if err := step.SetConfig(config.SymlinkStep{Source: "/nonexistent/source", Target: "/tmp/test-symlink"}); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	evalResult, err := p.Evaluate(context.TODO(), step)
	if err != nil {
		t.Fatalf("Evaluate() returned error: %v", err)
	}

	if evalResult.StepID != step.ID {
		t.Errorf("Expected StepID '%s', got '%s'", step.ID, evalResult.StepID)
	}

	if evalResult.CurrentState == model.StatusUnknown {
		t.Errorf("Expected valid status, got '%s'", evalResult.CurrentState)
	}

	if evalResult.Message == "" {
		t.Error("Expected non-empty message")
	}

	// Test that it returns Missing for nonexistent symlink
	if evalResult.CurrentState != model.StatusMissing {
		t.Errorf("Expected StatusMissing for nonexistent symlink, got '%s'", evalResult.CurrentState)
	}
}

func TestSymlinkEvaluateSatisfied(t *testing.T) {
	// This test would need actual filesystem setup
	// For now, we just verify the plugin can be called
	p := New()

	step := &config.Step{
		ID: "test-symlink-invalid",
		// No Symlink configuration
	}

	_, err := p.Evaluate(context.TODO(), step)
	if err == nil {
		t.Error("Expected error for missing configuration")
	}
}

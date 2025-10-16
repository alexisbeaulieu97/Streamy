package copyplugin

import (
	"context"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestCopyBasicFunctionality(t *testing.T) {
	p := New()

	// Test that plugin implements the interface
	if p == nil {
		t.Fatal("Plugin constructor returned nil")
	}

	// Test metadata
	meta := p.PluginMetadata()
	if meta.Name != "copy" {
		t.Errorf("Expected plugin name 'copy', got '%s'", meta.Name)
	}

	// Test schema
	schema := p.Schema()
	if schema == nil {
		t.Fatal("Schema returned nil")
	}

	// Test basic evaluation with missing source
	step := &config.Step{ID: "test-copy"}
	if err := step.SetConfig(config.CopyStep{Source: "/nonexistent/source", Destination: "/tmp/test-copy"}); err != nil {
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

	// Test that it returns Missing for nonexistent source
	if evalResult.CurrentState != model.StatusMissing {
		t.Errorf("Expected StatusMissing for nonexistent source, got '%s'", evalResult.CurrentState)
	}

	if !evalResult.RequiresAction {
		t.Error("Expected RequiresAction to be true for missing source")
	}
}

func TestCopyEvaluateInvalidConfig(t *testing.T) {
	p := New()

	step := &config.Step{
		ID: "test-copy-invalid",
		// No Copy configuration
	}

	_, err := p.Evaluate(context.TODO(), step)
	if err == nil {
		t.Error("Expected error for missing configuration")
	}
}

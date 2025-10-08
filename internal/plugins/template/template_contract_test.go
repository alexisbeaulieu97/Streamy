package templateplugin

import (
	"context"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestTemplateBasicFunctionality(t *testing.T) {
	p := New()

	// Test that plugin implements the interface
	if p == nil {
		t.Fatal("Plugin constructor returned nil")
	}

	// Test metadata
	meta := p.PluginMetadata()
	if meta.Name != "template" {
		t.Errorf("Expected plugin name 'template', got '%s'", meta.Name)
	}

	// Test schema
	schema := p.Schema()
	if schema == nil {
		t.Fatal("Schema returned nil")
	}

	// Test basic evaluation with missing source
	step := &config.Step{
		ID: "test-template",
		Template: &config.TemplateStep{
			Source:      "/nonexistent/source.template",
			Destination: "/tmp/test-template",
		},
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

func TestTemplateEvaluateInvalidConfig(t *testing.T) {
	p := New()

	step := &config.Step{
		ID: "test-template-invalid",
		// No Template configuration
	}

	_, err := p.Evaluate(nil, step)
	if err == nil {
		t.Error("Expected error for missing configuration")
	}
}

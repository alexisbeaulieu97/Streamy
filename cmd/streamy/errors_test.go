package main

import (
	"errors"
	"strings"
	"testing"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

func TestFormatError_DomainError(t *testing.T) {
	base := errors.New("command exited with status 1")
	derr := domainpipeline.NewExecutionError("plugin execution failed", base, map[string]interface{}{
		"pipeline":    "demo",
		"step_id":     "build",
		"plugin_type": "command",
	})

	output := FormatError(derr)
	if !strings.Contains(output, "[EXECUTION_ERROR] plugin execution failed") {
		t.Fatalf("expected execution header, got %q", output)
	}
	if !strings.Contains(output, "Suggestion:") {
		t.Fatalf("expected suggestion in output, got %q", output)
	}
	if !strings.Contains(output, "step_id: build") {
		t.Fatalf("expected context details, got %q", output)
	}
	if !strings.Contains(output, "Cause:") || !strings.Contains(output, base.Error()) {
		t.Fatalf("expected cause chain, got %q", output)
	}
}

func TestFormatError_AggregateError(t *testing.T) {
	stepErr := domainpipeline.NewValidationError("step requires command field", map[string]interface{}{"step_id": "setup"})
	agg := &applicationpipeline.AggregateError{
		Message: "pipeline execution failed",
		Errors:  []error{stepErr},
	}

	output := FormatError(agg)
	if !strings.Contains(output, "pipeline execution failed") {
		t.Fatalf("expected aggregate message, got %q", output)
	}
	if !strings.Contains(output, "1.") {
		t.Fatalf("expected enumerated child, got %q", output)
	}
	if !strings.Contains(output, "[VALIDATION_ERROR] step requires command field") {
		t.Fatalf("expected formatted domain error, got %q", output)
	}
}

func TestFormatError_ConfigParseError(t *testing.T) {
	parseErr := errors.New("unexpected indent")
	derr := domainpipeline.NewConfigError("invalid configuration syntax", parseErr, map[string]interface{}{
		"path":  "/tmp/pipeline.yaml",
		"line":  17,
		"field": "steps[0].type",
	})

	output := FormatError(derr)
	if !strings.Contains(output, "[CONFIG_ERROR] invalid configuration syntax") {
		t.Fatalf("expected config error header, got %q", output)
	}
	if !strings.Contains(output, "Suggestion: Fix configuration syntax errors and try again.") {
		t.Fatalf("expected remediation suggestion, got %q", output)
	}

	fieldIndex := strings.Index(output, "field: steps[0].type")
	lineIndex := strings.Index(output, "line: 17")
	pathIndex := strings.Index(output, "path: /tmp/pipeline.yaml")
	if fieldIndex == -1 || lineIndex == -1 || pathIndex == -1 {
		t.Fatalf("expected context entries for field, line, and path, got %q", output)
	}
	if fieldIndex >= lineIndex || lineIndex >= pathIndex {
		t.Fatalf("expected context to be sorted alphabetically, got %q", output)
	}
}

func TestFormatError_PluginExecutionError(t *testing.T) {
	cause := errors.New("exit status 1")
	derr := domainpipeline.NewExecutionError("plugin execution failed", cause, map[string]interface{}{
		"step_id":     "deploy",
		"plugin_type": "command",
		"operation":   "apply",
	})

	output := FormatError(derr)
	if !strings.Contains(output, "[EXECUTION_ERROR] plugin execution failed") {
		t.Fatalf("expected execution error header, got %q", output)
	}
	if !strings.Contains(output, "step_id: deploy") || !strings.Contains(output, "plugin_type: command") || !strings.Contains(output, "operation: apply") {
		t.Fatalf("expected execution context fields, got %q", output)
	}
	if !strings.Contains(output, "Cause:") || !strings.Contains(output, cause.Error()) {
		t.Fatalf("expected root cause output, got %q", output)
	}
}

func TestFormatError_DependencyCycle(t *testing.T) {
	derr := domainpipeline.NewCycleError([]string{"build", "test", "deploy", "build"})

	output := FormatError(derr)
	if !strings.Contains(output, "[CIRCULAR_DEPENDENCY] circular dependency detected") {
		t.Fatalf("expected cycle error header, got %q", output)
	}
	if !strings.Contains(output, "Context:") || !strings.Contains(output, "path: build -> test -> deploy -> build") {
		t.Fatalf("expected cycle path context, got %q", output)
	}
}

func TestFormatError_AggregateMultipleErrors(t *testing.T) {
	first := domainpipeline.NewValidationError("missing command field", map[string]interface{}{"step_id": "setup"})
	second := domainpipeline.NewExecutionError("plugin execution failed", errors.New("exit 1"), map[string]interface{}{
		"step_id":     "deploy",
		"plugin_type": "command",
		"operation":   "apply",
	})

	agg := &applicationpipeline.AggregateError{
		Message: "pipeline execution failed",
		Errors:  []error{first, second},
	}

	output := FormatError(agg)
	if !strings.Contains(output, "1.") || !strings.Contains(output, "2.") {
		t.Fatalf("expected enumerated errors, got %q", output)
	}
	if !strings.Contains(output, "[VALIDATION_ERROR] missing command field") {
		t.Fatalf("expected validation error details, got %q", output)
	}
	if !strings.Contains(output, "[EXECUTION_ERROR] plugin execution failed") {
		t.Fatalf("expected execution error details, got %q", output)
	}
	if !strings.Contains(output, "operation: apply") {
		t.Fatalf("expected operation context, got %q", output)
	}
	if strings.Count(output, "Suggestion:") < 2 {
		t.Fatalf("expected remediation suggestions for both errors, got %q", output)
	}
}

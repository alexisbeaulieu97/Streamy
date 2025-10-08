package templateplugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/pkg/diff"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

const templatePluginType = "template"

// Internal data for template operations
type templateEvaluationData struct {
	RenderedContent string
	RenderedHash    string
	DesiredMode     os.FileMode
	ExistingHash    string
	ExistingMode    os.FileMode
	ExistingExists  bool
	SourceExists    bool
}

type templatePlugin struct{}

// New creates a new instance of the template plugin.
func New() plugin.Plugin {
	return &templatePlugin{}
}

var _ plugin.Plugin = (*templatePlugin)(nil)

// PluginMetadata describes the plugin for the dependency registry.
//
// The empty Dependencies slice documents that template does not require other plugins.
// APIVersion pins compatibility with other plugins using the registry-provided interface.
func (p *templatePlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:         "template",
		Type:         "template",
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: []plugin.Dependency{},
		Stateful:     false,
		Description:  "Renders Go templates to files with variable substitution.",
	}
}

func (p *templatePlugin) Schema() any {
	return config.TemplateStep{}
}

func (p *templatePlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	cfg := step.Template
	if cfg == nil {
		return nil, plugin.NewValidationError(step.ID, fmt.Errorf("template configuration missing"))
	}

	if err := ctx.Err(); err != nil {
		return nil, plugin.NewStateError(step.ID, fmt.Errorf("context cancelled: %w", err))
	}

	// Check source template exists first
	if _, err := os.Stat(cfg.Source); err != nil {
		if os.IsNotExist(err) {
			return &model.EvaluationResult{
				StepID:         step.ID,
				CurrentState:   model.StatusMissing,
				RequiresAction: true,
				Message:        fmt.Sprintf("template source %s does not exist", cfg.Source),
				Diff:           fmt.Sprintf("Would create template from source: %s", cfg.Source),
			}, nil
		}
		return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot stat template source %s: %w", cfg.Source, err))
	}

	// Render the template (read-only operation)
	rendered, err := p.renderTemplate(ctx, cfg)
	if err != nil {
		return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to render template: %w", err))
	}

	renderedHash := hashContent(rendered)
	desiredMode, err := determineFileMode(cfg)
	if err != nil {
		return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to determine file mode: %w", err))
	}

	// Check destination state
	existingHash, existingMode, exists, err := existingDestinationState(cfg.Destination)
	if err != nil {
		return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot check destination: %w", err))
	}

	// Store evaluation data to avoid recomputation
	internalData := &templateEvaluationData{
		RenderedContent: rendered,
		RenderedHash:    renderedHash,
		DesiredMode:     desiredMode,
		ExistingHash:    existingHash,
		ExistingMode:    existingMode,
		ExistingExists:  exists,
		SourceExists:    true,
	}

	// Determine current state
	if !exists {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusMissing,
			RequiresAction: true,
			Message:        fmt.Sprintf("destination %s does not exist", cfg.Destination),
			Diff:           fmt.Sprintf("Would create: %s", cfg.Destination),
			InternalData:   internalData,
		}, nil
	}

	// Compare content and permissions
	contentMatches := renderedHash == existingHash
	modeMatches := desiredMode.Perm() == existingMode.Perm()

	if contentMatches && modeMatches {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusSatisfied,
			RequiresAction: false,
			Message:        fmt.Sprintf("template output is up to date: %s", cfg.Destination),
			InternalData:   internalData,
		}, nil
	}

	// Content differs - needs update
	var currentState model.VerificationStatus
	if contentMatches && !modeMatches {
		currentState = model.StatusDrifted // Only permissions differ
	} else {
		currentState = model.StatusDrifted // Content differs
	}

	// Generate diff
	diffStr := ""
	if !contentMatches {
		existingContent, readErr := os.ReadFile(cfg.Destination)
		if readErr != nil {
			diffStr = fmt.Sprintf("Cannot read existing file for diff: %v", readErr)
		} else {
			// Generate unified diff
			diffBytes := diff.GenerateUnifiedDiff([]byte(rendered), existingContent, cfg.Destination, cfg.Destination)
			diffStr = string(diffBytes)
		}
	}

	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   currentState,
		RequiresAction: true,
		Message:        fmt.Sprintf("template output differs from desired: %s", cfg.Destination),
		Diff:           diffStr,
		InternalData:   internalData,
	}, nil
}

func (p *templatePlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	cfg := step.Template
	if cfg == nil {
		return nil, plugin.NewValidationError(step.ID, fmt.Errorf("template configuration missing"))
	}

	// Use evaluation data to avoid recomputation
	var data *templateEvaluationData
	if evalResult.InternalData != nil {
		data = evalResult.InternalData.(*templateEvaluationData)
	} else {
		// Fallback to re-evaluating
		evalResult, err := p.Evaluate(ctx, step)
		if err != nil {
			return nil, convertError(step.ID, err)
		}
		if evalResult.InternalData == nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: "evaluation failed during apply",
				Error:   fmt.Errorf("evaluation failed"),
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("evaluation failed during apply"))
		}
		data = evalResult.InternalData.(*templateEvaluationData)
	}

	// Only apply if changes are needed
	if !evalResult.RequiresAction {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusSkipped,
			Message: "no changes needed",
		}, nil
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(cfg.Destination), 0755); err != nil {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: fmt.Sprintf("failed to create destination directory: %v", err),
			Error:   err,
		}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to create destination directory: %w", err))
	}

	// Write the rendered content
	if err := os.WriteFile(cfg.Destination, []byte(data.RenderedContent), data.DesiredMode); err != nil {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: fmt.Sprintf("failed to write template output: %v", err),
			Error:   err,
		}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to write template output: %w", err))
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("template applied successfully: %s", cfg.Destination),
	}, nil
}

// Helper functions

func convertError(stepID string, err error) error {
	// Convert legacy errors to new plugin errors
	var valErr *streamyerrors.ValidationError
	if errors.As(err, &valErr) {
		return plugin.NewValidationError(stepID, valErr.Err)
	}

	var execErr *streamyerrors.ExecutionError
	if errors.As(err, &execErr) {
		return plugin.NewExecutionError(stepID, execErr.Err)
	}

	// Fallback to ExecutionError for unknown error types
	return plugin.NewExecutionError(stepID, err)
}

// Preserve all the existing internal helper functions from the original file
func (p *templatePlugin) renderTemplate(ctx context.Context, cfg *config.TemplateStep) (string, error) {
	if cfg.Source == "" {
		return "", errors.New("template source cannot be empty")
	}

	// Read template file
	templateContent, err := os.ReadFile(cfg.Source)
	if err != nil {
		return "", fmt.Errorf("read template file %q: %w", cfg.Source, err)
	}

	// Parse and render template
	tmpl, err := template.New(cfg.Source).Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", cfg.Source, err)
	}

	var renderedContent bytes.Buffer
	if err := tmpl.Execute(&renderedContent, cfg.Vars); err != nil {
		return "", fmt.Errorf("render template %q: %w", cfg.Source, err)
	}

	return renderedContent.String(), nil
}

func hashContent(content string) string {
	hasher := sha256.New()
	hasher.Write([]byte(content))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func existingDestinationState(destination string) (string, os.FileMode, bool, error) {
	info, err := os.Stat(destination)
	if err != nil {
		if os.IsNotExist(err) {
			return "", os.FileMode(0), false, nil
		}
		return "", os.FileMode(0), false, err
	}

	content, err := os.ReadFile(destination)
	if err != nil {
		return "", os.FileMode(0), true, err
	}

	return hashContent(string(content)), info.Mode(), true, nil
}

func determineFileMode(cfg *config.TemplateStep) (os.FileMode, error) {
	mode := cfg.Mode
	if cfg.Mode == nil || *cfg.Mode == 0 {
		mode = &[]uint32{0644}[0] // default
	}
	return os.FileMode(*mode), nil
}

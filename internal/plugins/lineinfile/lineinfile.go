package lineinfileplugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type lineInFilePlugin struct{}

// New creates a new line_in_file plugin instance.
func New() plugin.Plugin {
	return &lineInFilePlugin{}
}

var _ plugin.Plugin = (*lineInFilePlugin)(nil)

// PluginMetadata describes the plugin for the dependency registry.
//
// The empty Dependencies slice documents that line_in_file does not require other plugins and
// demonstrates the pattern new plugins should follow when there are no dependency edges.
// APIVersion pins compatibility with other plugins using the registry-provided interface.
func (p *lineInFilePlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:         "line_in_file",
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: []plugin.Dependency{},
		Stateful:     false,
		Description:  "Manages ensuring specific lines exist within files.",
	}
}

func (p *lineInFilePlugin) Schema() any {
	return config.LineInFileStep{}
}

// Evaluation data for lineinfile operations
type lineInFileEvaluationData struct {
	State           *FileState
	CurrentContent  string
	UpdatedLines    []string
	TrailingNewline bool
	Changed         bool
	Action          string
	ChangeSet       *ChangeSet
}

// Convert from internal evaluationResult to our standard interface
func convertEvaluationResult(stepID string, result *evaluationResult) (*model.EvaluationResult, error) {
	var currentState model.VerificationStatus
	var requiresAction bool
	var message string
	var diff string

	if result == nil {
		return &model.EvaluationResult{
			StepID:         stepID,
			CurrentState:   model.StatusUnknown,
			RequiresAction: false,
			Message:        "evaluation failed - no result returned",
			InternalData:   nil,
		}, nil
	}

	if result.changed {
		currentState = model.StatusDrifted
		requiresAction = true
		message = fmt.Sprintf("line action needed: %s", result.action)
		// Use diff from ChangeSet
		if result.changeSet != nil {
			diff = result.changeSet.Diff
		}
	} else {
		currentState = model.StatusSatisfied
		requiresAction = false
		message = "line configuration satisfied"
	}

	internalData := &lineInFileEvaluationData{
		State:           result.state,
		CurrentContent:  result.original,
		UpdatedLines:    result.lines,
		TrailingNewline: result.trailing,
		Changed:         result.changed,
		Action:          result.action,
		ChangeSet:       result.changeSet,
	}

	return &model.EvaluationResult{
		StepID:         stepID,
		CurrentState:   currentState,
		RequiresAction: requiresAction,
		Message:        message,
		Diff:           diff,
		InternalData:   internalData,
	}, nil
}

func (p *lineInFilePlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	cfg, err := newConfigFromStep(step)
	if err != nil {
		return nil, convertError(step.ID, err)
	}

	// Use existing evaluation logic (which is already read-only)
	result, err := p.evaluate(ctx, step.ID, cfg)
	if err != nil {
		return nil, convertError(step.ID, err)
	}

	// Convert to standard EvaluationResult
	return convertEvaluationResult(step.ID, result)
}

func (p *lineInFilePlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	cfg, err := newConfigFromStep(step)
	if err != nil {
		return nil, convertError(step.ID, err)
	}

	// Use evaluation data to avoid recomputation
	var data *lineInFileEvaluationData
	if evalResult.InternalData != nil {
		data = evalResult.InternalData.(*lineInFileEvaluationData)
	} else {
		// Fallback to re-evaluating
		result, err := p.evaluate(ctx, step.ID, cfg)
		if err != nil {
			return nil, convertError(step.ID, err)
		}
		if result == nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: "evaluation failed during apply",
				Error:   fmt.Errorf("evaluation failed"),
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("evaluation failed during apply"))
		}
		data = &lineInFileEvaluationData{
			State:           result.state,
			CurrentContent:  result.original,
			UpdatedLines:    result.lines,
			TrailingNewline: result.trailing,
			Changed:         result.changed,
			Action:          result.action,
			ChangeSet:       result.changeSet,
		}
	}

	// Only apply if changes are needed
	if !data.Changed {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusSkipped,
			Message: "no changes needed",
		}, nil
	}

	// Write the updated content
	newContent := joinLines(data.UpdatedLines, data.TrailingNewline)

	// Handle backup if needed
	if cfg.Backup && data.State.Exists {
		originalBytes, err := encodeContent(data.CurrentContent, cfg.Encoding)
		if err != nil {
			return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to encode backup content: %w", err))
		}
		if _, err := createBackup(data.State.Path, cfg.BackupDir, originalBytes, data.State.Permissions); err != nil {
			return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to create backup: %w", err))
		}
	}

	encoded, err := encodeContent(newContent, cfg.Encoding)
	if err != nil {
		return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to encode content: %w", err))
	}

	if err := writeFileAtomic(data.State.Path, encoded, data.State.Permissions); err != nil {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: fmt.Sprintf("failed to write file: %v", err),
			Error:   err,
		}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to write file: %w", err))
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("line action completed: %s", data.Action),
	}, nil
}

// Helper functions for migration

func convertError(stepID string, err error) error {
	// Convert legacy streamyerrors to new plugin errors
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

// Legacy types and methods (keep for compatibility)
type evaluationResult struct {
	state     *FileState
	lines     []string
	trailing  bool
	changed   bool
	action    string
	content   string
	original  string
	changeSet *ChangeSet
}

// Preserve all the existing internal types and methods for compatibility
// These would be copied from the original file...

// Use FileState from file_ops.go which has all the required fields
// ChangeSet is defined in differ.go

// Copy all the internal helper functions from the original file
// For brevity, I'm just including the essential ones...

func (p *lineInFilePlugin) evaluate(ctx context.Context, stepID string, cfg *LineInFileConfig) (*evaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, streamyerrors.NewExecutionError(stepID, err)
	}

	state, err := readFileState(cfg)
	if err != nil {
		return nil, streamyerrors.NewExecutionError(stepID, fmt.Errorf("failed to read file: %w", err))
	}

	lines := append([]string{}, state.Lines...)
	trailing := state.TrailingNewline
	action := "none"
	changed := false

	switch cfg.State {
	case statePresent:
		if cfg.pattern == nil {
			var appended bool
			lines, appended = appendLineIfMissing(lines, cfg.Line)
			if appended {
				changed = true
				action = "append"
				trailing = true
			}
		} else {
			matches := findMatches(lines, cfg.pattern)
			if matches.MatchCount == 0 {
				lines = append(lines, cfg.Line)
				changed = true
				action = "append"
				trailing = true
			} else {
				updated, replaced, replaceErr := replaceLines(lines, matches, cfg.Line, cfg.OnMultipleMatches)
				if replaceErr != nil {
					if cfg.OnMultipleMatches == onMultiplePrompt {
						return nil, streamyerrors.NewExecutionError(stepID, fmt.Errorf("on_multiple_matches=prompt requires interactive session"))
					}
					return nil, streamyerrors.NewExecutionError(stepID, replaceErr)
				}
				if replaced {
					lines = updated
					changed = true
					action = "replace"
				}
			}
		}
	case stateAbsent:
		matches := findMatches(lines, cfg.pattern)
		updated, removed := removeMatchedLines(lines, matches)
		if removed {
			lines = updated
			changed = true
			action = "remove"
			trailing = len(lines) > 0 && trailing
		}
	}

	if len(lines) == 0 {
		trailing = false
	}

	originalContent := joinLines(state.Lines, state.TrailingNewline)
	newContent := joinLines(lines, trailing)

	if originalContent == newContent {
		changed = false
		action = "none"
	}

	changeSet := generateChangeSet(state.Lines, lines)
	if changeSet != nil {
		changeSet.Action = action
		changeSet.Changed = changed
	}

	return &evaluationResult{
		state:     state,
		lines:     lines,
		trailing:  trailing,
		changed:   changed,
		action:    action,
		content:   newContent,
		original:  originalContent,
		changeSet: changeSet,
	}, nil
}


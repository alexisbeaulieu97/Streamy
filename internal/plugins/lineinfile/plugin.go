package lineinfileplugin

import (
	"context"
	"errors"
	"fmt"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// Plugin implements the line_in_file step using the ports.Plugin contract.
type Plugin struct{}

var _ ports.Plugin = (*Plugin)(nil)

// New constructs a ports-native line_in_file plugin.
func New() ports.Plugin {
	return &Plugin{}
}

// Metadata returns the plugin metadata for registry wiring.
func (Plugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "line_in_file",
		Name:        "line_in_file",
		Version:     "1.0.0",
		Type:        domainplugin.TypeLineInFile,
		Description: "Manages ensuring specific lines exist within files.",
	}
}

// lineInFileEvaluationData is exposed via EvaluationResult.InternalData for apply reuse.
type lineInFileEvaluationData struct {
	State           *FileState
	CurrentContent  string
	UpdatedLines    []string
	TrailingNewline bool
	Changed         bool
	Action          string
	ChangeSet       *ChangeSet
}

// Evaluate analyses the target file and reports whether reconciliation is required.
func (Plugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("line_in_file evaluation cancelled", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeLineInFile),
		})
	}

	cfg, err := newConfigFromDomainStep(step)
	if err != nil {
		return nil, convertError(step.ID, err)
	}

	result, err := evaluateLineInFile(ctx, step.ID, cfg)
	if err != nil {
		return nil, convertError(step.ID, err)
	}

	eval, err := convertEvaluationResult(step.ID, result)
	if err != nil {
		return nil, convertError(step.ID, err)
	}
	return eval, nil
}

// Apply mutates the target file to satisfy the desired line state.
func (Plugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("line_in_file apply cancelled", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeLineInFile),
		})
	}

	cfg, err := newConfigFromDomainStep(step)
	if err != nil {
		return nil, convertError(step.ID, err)
	}

	var data *lineInFileEvaluationData
	if evaluation != nil {
		if typed, ok := evaluation.InternalData.(*lineInFileEvaluationData); ok && typed != nil {
			data = typed
		}
	}
	if data == nil {
		fallbackEval, evalErr := evaluateLineInFile(ctx, step.ID, cfg)
		if evalErr != nil {
			return nil, convertError(step.ID, evalErr)
		}
		converted, convErr := convertEvaluationResult(step.ID, fallbackEval)
		if convErr != nil {
			return nil, convertError(step.ID, convErr)
		}
		var ok bool
		data, ok = converted.InternalData.(*lineInFileEvaluationData)
		if !ok || data == nil {
			return nil, domainpipeline.NewInternalError("line_in_file evaluation missing internal data", nil, map[string]interface{}{
				"step_id":     step.ID,
				"plugin_type": string(domainplugin.TypeLineInFile),
			})
		}
		evaluation = converted
	}

	if evaluation != nil && !evaluation.RequiresAction {
		return &domainpipeline.StepResult{
			StepID:  step.ID,
			Status:  domainpipeline.StatusAlreadySatisfied,
			Message: "no changes needed",
		}, nil
	}

	if !data.Changed {
		return &domainpipeline.StepResult{
			StepID:  step.ID,
			Status:  domainpipeline.StatusAlreadySatisfied,
			Message: "no changes needed",
		}, nil
	}

	newContent := joinLines(data.UpdatedLines, data.TrailingNewline)

	if cfg.Backup && data.State.Exists {
		originalBytes, encodeErr := encodeContent(data.CurrentContent, cfg.Encoding)
		if encodeErr != nil {
			return nil, convertError(step.ID, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("failed to encode backup content: %w", encodeErr)))
		}
		if _, err := createBackup(data.State.Path, cfg.BackupDir, originalBytes, data.State.Permissions); err != nil {
			return nil, convertError(step.ID, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("failed to create backup: %w", err)))
		}
	}

	encoded, encodeErr := encodeContent(newContent, cfg.Encoding)
	if encodeErr != nil {
		return nil, convertError(step.ID, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("failed to encode content: %w", encodeErr)))
	}

	if err := writeFileAtomic(data.State.Path, encoded, data.State.Permissions); err != nil {
		return nil, convertError(step.ID, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("failed to write file: %w", err)))
	}

	diff := ""
	if data.ChangeSet != nil {
		diff = data.ChangeSet.Diff
	}

	return &domainpipeline.StepResult{
		StepID:  step.ID,
		Status:  domainpipeline.StatusSuccess,
		Message: fmt.Sprintf("line action completed: %s", data.Action),
		Changed: true,
		Diff:    diff,
	}, nil
}

func convertEvaluationResult(stepID string, result *evaluationResult) (*domainpipeline.EvaluationResult, error) {
	if result == nil {
		return &domainpipeline.EvaluationResult{
			RequiresAction: false,
			CurrentState:   string(domainpipeline.VerificationUnknown),
			DesiredState:   "evaluation failed - no result returned",
			InternalData:   nil,
		}, nil
	}

	currentState := string(domainpipeline.VerificationSatisfied)
	message := "line configuration satisfied"
	diff := ""
	requiresAction := false

	if !result.state.Exists {
		currentState = string(domainpipeline.VerificationFailed)
		message = "file does not exist"
		requiresAction = true
	} else if result.changed {
		currentState = string(domainpipeline.VerificationFailed)
		message = fmt.Sprintf("line action needed: %s", result.action)
		requiresAction = true
		if result.changeSet != nil {
			diff = result.changeSet.Diff
		}
	}

	if !result.changed && result.action == "none" && !result.state.Exists {
		requiresAction = true
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

	return &domainpipeline.EvaluationResult{
		RequiresAction: requiresAction,
		CurrentState:   currentState,
		DesiredState:   message,
		Diff:           diff,
		InternalData:   internalData,
	}, nil
}

func convertError(stepID string, err error) error {
	if err == nil {
		return nil
	}

	ctxDetails := map[string]interface{}{
		"step_id":     stepID,
		"plugin_type": string(domainplugin.TypeLineInFile),
	}

	if errors.Is(err, context.Canceled) {
		return domainpipeline.NewCancelledError("operation cancelled", ctxDetails)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domainpipeline.NewTimeoutError("operation timed out", err, ctxDetails)
	}

	var valErr *streamyerrors.ValidationError
	if errors.As(err, &valErr) {
		ctx := map[string]interface{}{}
		for k, v := range ctxDetails {
			ctx[k] = v
		}
		if valErr.Field != "" {
			ctx["field"] = valErr.Field
		}
		return domainpipeline.NewDomainError(domainpipeline.ErrCodeValidation, valErr.Message, valErr.Err, ctx)
	}

	var execErr *streamyerrors.ExecutionError
	if errors.As(err, &execErr) {
		return domainpipeline.NewExecutionError("execution failure", execErr.Err, ctxDetails)
	}

	return domainpipeline.NewExecutionError("line_in_file error", err, ctxDetails)
}

// evaluationResult captures the outcome of the helper evaluation routine.
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

func evaluateLineInFile(ctx context.Context, stepID string, cfg *LineInFileConfig) (*evaluationResult, error) {
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

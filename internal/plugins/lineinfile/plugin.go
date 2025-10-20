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

	eval, err := convertEvaluationResult(result)
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

	data, convertedEval, evalErr := ensureLineEvaluation(ctx, step, cfg, evaluation)
	if evalErr != nil {
		return nil, evalErr
	}

	evaluation = convertedEval

	if skipLineChange(evaluation, data) {
		return buildNoChangeResult(step.ID), nil
	}

	if backupErr := createLineInFileBackup(step.ID, cfg, data); backupErr != nil {
		return nil, convertError(step.ID, backupErr)
	}

	if writeErr := writeUpdatedLineContent(step.ID, cfg, data); writeErr != nil {
		return nil, convertError(step.ID, writeErr)
	}

	return buildLineChangeResult(step.ID, data), nil
}

func ensureLineEvaluation(ctx context.Context, step domainpipeline.Step, cfg *LineInFileConfig, evaluation *domainpipeline.EvaluationResult) (*lineInFileEvaluationData, *domainpipeline.EvaluationResult, error) {
	if evaluation != nil {
		if typed, ok := evaluation.InternalData.(*lineInFileEvaluationData); ok && typed != nil {
			return typed, evaluation, nil
		}
	}

	fallbackEval, err := evaluateLineInFile(ctx, step.ID, cfg)
	if err != nil {
		return nil, nil, convertError(step.ID, err)
	}

	converted, convErr := convertEvaluationResult(fallbackEval)
	if convErr != nil {
		return nil, nil, convertError(step.ID, convErr)
	}

	data, ok := converted.InternalData.(*lineInFileEvaluationData)
	if !ok || data == nil {
		return nil, nil, domainpipeline.NewInternalError("line_in_file evaluation missing internal data", nil, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeLineInFile),
		})
	}

	return data, converted, nil
}

func skipLineChange(evaluation *domainpipeline.EvaluationResult, data *lineInFileEvaluationData) bool {
	if evaluation != nil && !evaluation.RequiresAction {
		return true
	}

	if data == nil {
		return true
	}

	return !data.Changed
}

func buildNoChangeResult(stepID string) *domainpipeline.StepResult {
	return &domainpipeline.StepResult{
		StepID:  stepID,
		Status:  domainpipeline.StatusAlreadySatisfied,
		Message: "no changes needed",
	}
}

func createLineInFileBackup(stepID string, cfg *LineInFileConfig, data *lineInFileEvaluationData) error {
	if !cfg.Backup || data == nil || data.State == nil || !data.State.Exists {
		return nil
	}

	originalBytes, err := encodeContent(data.CurrentContent, cfg.Encoding)
	if err != nil {
		//nolint:wrapcheck // propagate execution error for caller to convert
		return streamyerrors.NewExecutionError(stepID, fmt.Errorf("failed to encode backup content: %w", err))
	}

	if _, err := createBackup(data.State.Path, cfg.BackupDir, originalBytes, data.State.Permissions); err != nil {
		//nolint:wrapcheck // propagate execution error for caller to convert
		return streamyerrors.NewExecutionError(stepID, fmt.Errorf("failed to create backup: %w", err))
	}

	return nil
}

func writeUpdatedLineContent(stepID string, cfg *LineInFileConfig, data *lineInFileEvaluationData) error {
	if data == nil || data.State == nil {
		//nolint:wrapcheck // propagate execution error for caller to convert
		return streamyerrors.NewExecutionError(stepID, fmt.Errorf("missing file state"))
	}

	newContent := joinLines(data.UpdatedLines, data.TrailingNewline)

	encoded, encodeErr := encodeContent(newContent, cfg.Encoding)
	if encodeErr != nil {
		//nolint:wrapcheck // propagate execution error for caller to convert
		return streamyerrors.NewExecutionError(stepID, fmt.Errorf("failed to encode content: %w", encodeErr))
	}

	if err := writeFileAtomic(data.State.Path, encoded, data.State.Permissions); err != nil {
		//nolint:wrapcheck // propagate execution error for caller to convert
		return streamyerrors.NewExecutionError(stepID, fmt.Errorf("failed to write file: %w", err))
	}

	return nil
}

func buildLineChangeResult(stepID string, data *lineInFileEvaluationData) *domainpipeline.StepResult {
	diff := ""
	if data != nil && data.ChangeSet != nil {
		diff = data.ChangeSet.Diff
	}

	action := "none"
	if data != nil && data.Action != "" {
		action = data.Action
	}

	return &domainpipeline.StepResult{
		StepID:  stepID,
		Status:  domainpipeline.StatusSuccess,
		Message: fmt.Sprintf("line action completed: %s", action),
		Changed: true,
		Diff:    diff,
	}
}

func convertEvaluationResult(result *evaluationResult) (*domainpipeline.EvaluationResult, error) {
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

	if !result.changed && result.action == actionNone && !result.state.Exists {
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
		//nolint:wrapcheck // propagate execution error from helper
		return nil, streamyerrors.NewExecutionError(stepID, err)
	}

	state, err := readFileState(cfg)
	if err != nil {
		//nolint:wrapcheck // propagate execution error from helper
		return nil, streamyerrors.NewExecutionError(stepID, fmt.Errorf("failed to read file: %w", err))
	}

	lines := append([]string{}, state.Lines...)
	trailing := state.TrailingNewline
	action := actionNone
	changed := false

	switch cfg.State {
	case statePresent:
		updatedLines, updatedTrailing, updatedAction, updatedChanged, presentErr := applyPresentState(lines, trailing, cfg, stepID)
		if presentErr != nil {
			//nolint:wrapcheck // propagate execution error from helper
			return nil, presentErr
		}

		lines = updatedLines
		trailing = updatedTrailing
		action = updatedAction
		changed = updatedChanged
	case stateAbsent:
		updatedLines, updatedTrailing, updatedAction, updatedChanged := applyAbsentState(lines, trailing, cfg)
		lines = updatedLines
		trailing = updatedTrailing
		action = updatedAction
		changed = updatedChanged
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

func applyPresentState(lines []string, trailing bool, cfg *LineInFileConfig, stepID string) ([]string, bool, string, bool, error) {
	if cfg.pattern == nil {
		updated, appended := appendLineIfMissing(lines, cfg.Line)
		if appended {
			return updated, true, "append", true, nil
		}

		return lines, trailing, actionNone, false, nil
	}

	matches := findMatches(lines, cfg.pattern)
	if matches.MatchCount == 0 {
		return append(lines, cfg.Line), true, "append", true, nil
	}

	updated, replaced, replaceErr := replaceLines(lines, matches, cfg.Line, cfg.OnMultipleMatches)
	if replaceErr != nil {
		if cfg.OnMultipleMatches == onMultiplePrompt {
			//nolint:wrapcheck // propagate execution error for caller to convert
			return nil, false, "", false, streamyerrors.NewExecutionError(stepID, fmt.Errorf("on_multiple_matches=prompt requires interactive session"))
		}

		//nolint:wrapcheck // propagate execution error for caller to convert
		return nil, false, "", false, streamyerrors.NewExecutionError(stepID, replaceErr)
	}

	if replaced {
		return updated, trailing, "replace", true, nil
	}

	return lines, trailing, actionNone, false, nil
}

func applyAbsentState(lines []string, trailing bool, cfg *LineInFileConfig) ([]string, bool, string, bool) {
	matches := findMatches(lines, cfg.pattern)

	updated, removed := removeMatchedLines(lines, matches)
	if !removed {
		return lines, trailing, actionNone, false
	}

	trailing = len(updated) > 0 && trailing

	return updated, trailing, "remove", true
}

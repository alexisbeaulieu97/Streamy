package lineinfileplugin

import (
	"context"
	"fmt"
	"strings"

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

func init() {
	if err := plugin.RegisterPlugin("line_in_file", New()); err != nil {
		panic(err)
	}
}

func (p *lineInFilePlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "line-in-file",
		Version: "1.0.0",
		Type:    "line_in_file",
	}
}

func (p *lineInFilePlugin) Schema() interface{} {
	return config.LineInFileStep{}
}

func (p *lineInFilePlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	cfg, err := newConfigFromStep(step)
	if err != nil {
		return false, err
	}

	result, err := p.evaluate(ctx, step.ID, cfg)
	if err != nil {
		return false, err
	}

	return !result.changed, nil
}

func (p *lineInFilePlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg, err := newConfigFromStep(step)
	if err != nil {
		return nil, err
	}

	result, err := p.evaluate(ctx, step.ID, cfg)
	if err != nil {
		return nil, err
	}

	res := &model.StepResult{StepID: step.ID}
	if !result.changed {
		res.Status = model.StatusSkipped
		res.Message = "no changes required"
		return res, nil
	}

	newContent := result.content
	encoded, err := encodeContent(newContent, cfg.Encoding)
	if err != nil {
		return nil, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("failed to encode content: %w", err))
	}

	if cfg.Backup && result.state.Exists {
		originalBytes, err := encodeContent(result.original, cfg.Encoding)
		if err != nil {
			return nil, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("failed to encode backup content: %w", err))
		}
		if _, err := createBackup(result.state.Path, cfg.BackupDir, originalBytes, result.state.Permissions); err != nil {
			return nil, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("failed to create backup: %w", err))
		}
	}

	if err := writeFileAtomic(result.state.Path, encoded, result.state.Permissions); err != nil {
		return nil, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("failed to write file: %w", err))
	}

	res.Status = model.StatusSuccess
	res.Message = p.successMessage(cfg, result)
	return res, nil
}

func (p *lineInFilePlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg, err := newConfigFromStep(step)
	if err != nil {
		return nil, err
	}

	result, err := p.evaluate(ctx, step.ID, cfg)
	if err != nil {
		return nil, err
	}

	res := &model.StepResult{StepID: step.ID}
	if !result.changed {
		res.Status = model.StatusSkipped
		res.Message = "no changes required"
		return res, nil
	}

	if !result.state.Exists {
		res.Status = model.StatusWouldCreate
	} else {
		res.Status = model.StatusWouldUpdate
	}

	msg := result.changeSet.Diff
	if strings.TrimSpace(msg) == "" {
		msg = p.successMessage(cfg, result)
	}
	res.Message = msg
	return res, nil
}

func (p *lineInFilePlugin) successMessage(cfg *LineInFileConfig, result *evaluationResult) string {
	switch result.action {
	case "append":
		return fmt.Sprintf("added line to %s", result.state.Path)
	case "replace":
		return fmt.Sprintf("updated line in %s", result.state.Path)
	case "remove":
		return fmt.Sprintf("removed line(s) from %s", result.state.Path)
	default:
		return fmt.Sprintf("updated %s", result.state.Path)
	}
}

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

var _ plugin.Plugin = (*lineInFilePlugin)(nil)

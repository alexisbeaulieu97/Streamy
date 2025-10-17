package lineinfileplugin

import (
	"context"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/portutil"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type portLineInFilePlugin struct {
	legacy *lineInFilePlugin
}

// NewPort constructs a ports.Plugin backed by the legacy line_in_file plugin.
func NewPort() ports.Plugin {
	return &portLineInFilePlugin{legacy: &lineInFilePlugin{}}
}

func (p *portLineInFilePlugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:           "line_in_file",
		Name:         "line_in_file",
		Version:      "1.0.0",
		Type:         domainplugin.TypeLineInFile,
		Description:  "Manages ensuring specific lines exist within files.",
		Dependencies: nil,
		APIVersion:   "1.x",
	}
}

func (p *portLineInFilePlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid line_in_file configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeLineInFile)},
		}
	}

	legacyEval, err := p.legacy.Evaluate(ctx, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypeLineInFile, err)
	}

	result := portutil.LegacyEvaluationToDomain(legacyEval)
	if result == nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeInternal,
			Message: "line_in_file evaluation returned nil result",
			Context: map[string]interface{}{"step_id": step.ID},
		}
	}
	return result, nil
}

func (p *portLineInFilePlugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid line_in_file configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeLineInFile)},
		}
	}

	legacyEval := portutil.DomainEvaluationToLegacy(step.ID, evaluation)
	legacyResult, err := p.legacy.Apply(ctx, legacyEval, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypeLineInFile, err)
	}
	domainResult := portutil.LegacyStepResultToDomain(step.ID, domainplugin.TypeLineInFile, legacyResult)
	return &domainResult, nil
}

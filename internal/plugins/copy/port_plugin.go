package copyplugin

import (
	"context"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/portutil"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type portCopyPlugin struct {
	legacy *copyPlugin
}

// NewPort constructs a ports.Plugin backed by the legacy copy plugin.
func NewPort() ports.Plugin {
	return &portCopyPlugin{legacy: &copyPlugin{}}
}

func (p *portCopyPlugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:           "copy",
		Name:         "copy",
		Version:      "1.0.0",
		Type:         domainplugin.TypeCopy,
		Description:  "Copies files and directories with permission and backup support.",
		Dependencies: nil,
		APIVersion:   "1.x",
	}
}

func (p *portCopyPlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid copy configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCopy)},
		}
	}

	legacyEval, err := p.legacy.Evaluate(ctx, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypeCopy, err)
	}

	result := portutil.LegacyEvaluationToDomain(legacyEval)
	if result == nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeInternal,
			Message: "copy evaluation returned nil result",
			Context: map[string]interface{}{"step_id": step.ID},
		}
	}
	return result, nil
}

func (p *portCopyPlugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid copy configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCopy)},
		}
	}

	legacyEval := portutil.DomainEvaluationToLegacy(step.ID, evaluation)
	legacyResult, err := p.legacy.Apply(ctx, legacyEval, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypeCopy, err)
	}
	domainResult := portutil.LegacyStepResultToDomain(step.ID, domainplugin.TypeCopy, legacyResult)
	return &domainResult, nil
}

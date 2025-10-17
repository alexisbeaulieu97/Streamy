package templateplugin

import (
	"context"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/portutil"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type portTemplatePlugin struct {
	legacy *templatePlugin
}

// NewPort constructs a ports.Plugin backed by the legacy template plugin.
func NewPort() ports.Plugin {
	return &portTemplatePlugin{legacy: &templatePlugin{}}
}

func (p *portTemplatePlugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:           "template",
		Name:         "template",
		Version:      "1.0.0",
		Type:         domainplugin.TypeTemplate,
		Description:  "Renders Go templates to files with variable substitution.",
		Dependencies: nil,
		APIVersion:   "1.x",
	}
}

func (p *portTemplatePlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid template configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeTemplate)},
		}
	}

	legacyEval, err := p.legacy.Evaluate(ctx, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypeTemplate, err)
	}

	result := portutil.LegacyEvaluationToDomain(legacyEval)
	if result == nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeInternal,
			Message: "template evaluation returned nil result",
			Context: map[string]interface{}{"step_id": step.ID},
		}
	}
	return result, nil
}

func (p *portTemplatePlugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid template configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeTemplate)},
		}
	}

	legacyEval := portutil.DomainEvaluationToLegacy(step.ID, evaluation)
	legacyResult, err := p.legacy.Apply(ctx, legacyEval, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypeTemplate, err)
	}
	domainResult := portutil.LegacyStepResultToDomain(step.ID, domainplugin.TypeTemplate, legacyResult)
	return &domainResult, nil
}

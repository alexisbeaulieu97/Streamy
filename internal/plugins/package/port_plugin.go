package packageplugin

import (
	"context"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/portutil"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type portPackagePlugin struct {
	legacy *packagePlugin
}

// NewPort constructs a ports.Plugin implementation backed by the legacy package plugin logic.
func NewPort() ports.Plugin {
	return &portPackagePlugin{legacy: &packagePlugin{}}
}

func (p *portPackagePlugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:           "package",
		Name:         "package",
		Version:      "1.0.0",
		Type:         domainplugin.TypePackage,
		Description:  "Manages system packages using apt package manager.",
		Dependencies: nil,
		APIVersion:   "1.x",
	}
}

func (p *portPackagePlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid package configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypePackage)},
		}
	}

	legacyEval, err := p.legacy.Evaluate(ctx, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypePackage, err)
	}

	result := portutil.LegacyEvaluationToDomain(legacyEval)
	if result == nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeInternal,
			Message: "package evaluation returned nil result",
			Context: map[string]interface{}{"step_id": step.ID},
		}
	}
	return result, nil
}

func (p *portPackagePlugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid package configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypePackage)},
		}
	}

	legacyEval := portutil.DomainEvaluationToLegacy(step.ID, evaluation)
	legacyResult, err := p.legacy.Apply(ctx, legacyEval, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypePackage, err)
	}
	domainResult := portutil.LegacyStepResultToDomain(step.ID, domainplugin.TypePackage, legacyResult)
	return &domainResult, nil
}

package symlinkplugin

import (
	"context"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/portutil"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type portSymlinkPlugin struct {
	legacy *symlinkPlugin
}

// NewPort constructs a ports.Plugin implementation backed by the legacy symlink plugin logic.
func NewPort() ports.Plugin {
	return &portSymlinkPlugin{
		legacy: &symlinkPlugin{},
	}
}

func (p *portSymlinkPlugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:           "symlink",
		Name:         "symlink",
		Version:      "1.0.0",
		Type:         domainplugin.TypeSymlink,
		Description:  "Manages symbolic links with target validation.",
		Dependencies: nil,
		APIVersion:   "1.x",
	}
}

func (p *portSymlinkPlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid symlink configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeSymlink)},
		}
	}

	legacyEval, err := p.legacy.Evaluate(ctx, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypeSymlink, err)
	}

	result := portutil.LegacyEvaluationToDomain(legacyEval)
	if result == nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeInternal,
			Message: "symlink evaluation returned nil result",
			Context: map[string]interface{}{"step_id": step.ID},
		}
	}
	return result, nil
}

func (p *portSymlinkPlugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	cfgStep, err := portutil.DomainStepToConfig(step)
	if err != nil {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: "invalid symlink configuration",
			Cause:   err,
			Context: map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeSymlink)},
		}
	}

	legacyEval := portutil.DomainEvaluationToLegacy(step.ID, evaluation)
	legacyResult, err := p.legacy.Apply(ctx, legacyEval, cfgStep)
	if err != nil {
		return nil, portutil.LegacyErrorToDomain(step.ID, domainplugin.TypeSymlink, err)
	}
	domainResult := portutil.LegacyStepResultToDomain(step.ID, domainplugin.TypeSymlink, legacyResult)
	return &domainResult, nil
}

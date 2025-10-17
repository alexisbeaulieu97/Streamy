package portutil

import (
	"errors"
	"time"

	configpkg "github.com/alexisbeaulieu97/streamy/internal/config"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	legacyplugin "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

// DomainStepToConfig converts a domain step into the legacy config representation.
func DomainStepToConfig(step domainpipeline.Step) (*configpkg.Step, error) {
	cfg := &configpkg.Step{
		ID:            step.ID,
		Name:          step.Name,
		Type:          string(step.Type),
		DependsOn:     append([]string(nil), step.DependsOn...),
		Enabled:       step.Enabled,
		VerifyTimeout: step.VerifyTimeout,
	}
	if err := cfg.SetConfig(step.Config); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LegacyEvaluationToDomain maps a legacy evaluation result into the domain representation.
func LegacyEvaluationToDomain(eval *model.EvaluationResult) *domainpipeline.EvaluationResult {
	if eval == nil {
		return nil
	}
	return &domainpipeline.EvaluationResult{
		RequiresAction: eval.RequiresAction,
		CurrentState:   string(eval.CurrentState),
		DesiredState:   eval.Message,
		Diff:           eval.Diff,
		InternalData:   eval.InternalData,
	}
}

// DomainEvaluationToLegacy maps a domain evaluation result back to the legacy representation.
func DomainEvaluationToLegacy(stepID string, eval *domainpipeline.EvaluationResult) *model.EvaluationResult {
	if eval == nil {
		return nil
	}
	status := model.VerificationStatus(eval.CurrentState)
	if !status.IsValid() {
		status = model.StatusUnknown
	}
	return &model.EvaluationResult{
		StepID:         stepID,
		CurrentState:   status,
		RequiresAction: eval.RequiresAction,
		Message:        eval.DesiredState,
		Diff:           eval.Diff,
		InternalData:   eval.InternalData,
	}
}

// LegacyStepResultToDomain converts a legacy step result into a domain result.
func LegacyStepResultToDomain(stepID string, pluginType domainplugin.Type, res *model.StepResult) domainpipeline.StepResult {
	if res == nil {
		return domainpipeline.StepResult{StepID: stepID}
	}

	status := mapResultStatus(res.Status)
	changed := res.Status == model.StatusWouldCreate || res.Status == model.StatusWouldUpdate

	return domainpipeline.StepResult{
		StepID:   stepID,
		Status:   status,
		Duration: int(res.Duration / time.Millisecond),
		Message:  res.Message,
		Output:   res.Message,
		Error:    LegacyErrorToDomain(stepID, pluginType, res.Error),
		Changed:  changed,
	}
}

func mapResultStatus(status string) domainpipeline.ResultStatus {
	switch status {
	case model.StatusSuccess:
		return domainpipeline.StatusSuccess
	case model.StatusSkipped:
		return domainpipeline.StatusSkipped
	case model.StatusWouldCreate, model.StatusWouldUpdate:
		return domainpipeline.StatusSuccess
	default:
		return domainpipeline.StatusFailure
	}
}

// LegacyErrorToDomain converts legacy plugin errors into domain errors with structured context.
func LegacyErrorToDomain(stepID string, pluginType domainplugin.Type, err error) *domainpipeline.DomainError {
	if err == nil {
		return nil
	}

	var domainErr *domainpipeline.DomainError
	if errors.As(err, &domainErr) {
		return domainErr
	}

	context := map[string]interface{}{
		"step_id":     stepID,
		"plugin_type": string(pluginType),
	}

	switch typed := err.(type) {
	case *legacyplugin.ValidationError:
		return &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeValidation,
			Message: typed.Error(),
			Cause:   typed,
			Context: context,
		}
	case *legacyplugin.ExecutionError:
		return &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeExecution,
			Message: typed.Error(),
			Cause:   typed,
			Context: context,
		}
	case *legacyplugin.StateError:
		return &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeState,
			Message: typed.Error(),
			Cause:   typed,
			Context: context,
		}
	default:
		return &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeExecution,
			Message: err.Error(),
			Cause:   err,
			Context: context,
		}
	}
}

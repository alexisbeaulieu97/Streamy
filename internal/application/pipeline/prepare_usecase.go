package pipeline

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// PrepareUseCase orchestrates configuration loading, validation, and planning.
type PrepareUseCase struct {
	configLoader ports.ConfigLoader
	dagBuilder   ports.DAGBuilder
	logger       ports.Logger
	events       ports.EventPublisher
}

// NewPrepareUseCase constructs a prepare use case with the required ports.
func NewPrepareUseCase(loader ports.ConfigLoader, builder ports.DAGBuilder, logger ports.Logger, events ports.EventPublisher) *PrepareUseCase {
	return &PrepareUseCase{
		configLoader: loader,
		dagBuilder:   builder,
		logger:       logger,
		events:       events,
	}
}

// Prepare loads the pipeline configuration, validates it, builds the DAG, and generates an execution plan.
func (u *PrepareUseCase) Prepare(ctx context.Context, configPath string) (*pipeline.Pipeline, *pipeline.ExecutionPlan, error) {
	if u.logger != nil {
		u.logger.Info(ctx, "preparing pipeline", "config_path", configPath)
	}
	publishEvent(ctx, u.events, u.logger, ports.EventPipelineStarted, map[string]interface{}{
		"config_path": configPath,
		"phase":       "prepare",
	})

	pip, err := u.configLoader.Load(ctx, configPath)
	if err != nil {
		if u.logger != nil {
			u.logger.Error(ctx, "failed to load pipeline configuration", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
			"config_path": configPath,
			"phase":       "prepare",
			"error":       err,
		})
		return nil, nil, err
	}

	if u.logger != nil {
		u.logger.Debug(ctx, "building execution plan", "config_path", configPath, "step_count", len(pip.Steps))
	}
	plan, err := u.dagBuilder.Build(ctx, pip.Steps)
	if err != nil {
		if u.logger != nil {
			u.logger.Error(ctx, "failed to build execution plan", "config_path", configPath, "error", err)
		}
		return pip, nil, err
	}

	if err := plan.Validate(*pip); err != nil {
		if u.logger != nil {
			u.logger.Error(ctx, "execution plan validation failed", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
			"config_path": configPath,
			"phase":       "prepare",
			"error":       err,
		})
		return pip, nil, err
	}

	if u.logger != nil {
		u.logger.Info(ctx, "pipeline prepared", "config_path", configPath, "levels", len(plan.Levels))
	}
	publishEvent(ctx, u.events, u.logger, ports.EventPipelineCompleted, map[string]interface{}{
		"config_path": configPath,
		"phase":       "prepare",
		"levels":      len(plan.Levels),
		"step_count":  len(pip.Steps),
	})
	return pip, plan, nil
}

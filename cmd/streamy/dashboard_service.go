package main

import (
	"context"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/pipelineconv"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/tui/dashboard"
)

type dashboardPipelineAdapter struct {
	applyUseCase  *applicationpipeline.ApplyUseCase
	verifyUseCase *applicationpipeline.VerifyUseCase
}

func newDashboardPipelineAdapter(app *AppContext) dashboard.PipelineService {
	return &dashboardPipelineAdapter{
		applyUseCase:  app.ApplyUseCase,
		verifyUseCase: app.VerifyUseCase,
	}
}

func (a *dashboardPipelineAdapter) Verify(ctx context.Context, opts dashboard.VerifyOptions) (*registry.ExecutionResult, error) {
	pipeline, results, err := a.verifyUseCase.Verify(ctx, opts.ConfigPath)
	if err != nil {
		return nil, err
	}

	summary := pipelineconv.BuildVerificationSummary(pipeline, results)
	result := pipelineconv.SummaryToExecutionResult(summary, opts.ConfigPath)
	if pipeline != nil {
		result.PipelineID = pipeline.Name
		if result.PipelineID == "" {
			result.PipelineID = opts.ConfigPath
		}
	}
	return result, nil
}

func (a *dashboardPipelineAdapter) Apply(ctx context.Context, opts dashboard.ApplyOptions) (*registry.ExecutionResult, error) {
	pipeline, stepResults, _, err := a.applyUseCase.Apply(ctx, opts.ConfigPath, opts.DryRun)
	if err != nil {
		return nil, err
	}

	modelResults := make([]model.StepResult, len(stepResults))
	for i, res := range stepResults {
		modelResults[i] = pipelineconv.ConvertStepResult(res, opts.DryRun)
	}

	execResult := pipelineconv.ConvertApplyResults(modelResults, opts.ConfigPath, nil, nil)
	if pipeline != nil {
		execResult.PipelineID = pipeline.Name
		if execResult.PipelineID == "" {
			execResult.PipelineID = opts.ConfigPath
		}
	}
	return execResult, nil
}

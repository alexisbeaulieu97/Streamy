package pipeline

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// VerifyUseCase coordinates preparing and verifying pipelines without applying changes.
type VerifyUseCase struct {
	prepareUseCase *PrepareUseCase
	executor       ports.PluginExecutor
	logger         ports.Logger
	events         ports.EventPublisher
}

// NewVerifyUseCase constructs a VerifyUseCase with dependencies injected.
func NewVerifyUseCase(prepare *PrepareUseCase, executor ports.PluginExecutor, logger ports.Logger, events ports.EventPublisher) *VerifyUseCase {
	return &VerifyUseCase{
		prepareUseCase: prepare,
		executor:       executor,
		logger:         logger,
		events:         events,
	}
}

// Verify loads the pipeline and runs verification via the plugin executor.
func (u *VerifyUseCase) Verify(ctx context.Context, configPath string) (*pipeline.Pipeline, []pipeline.VerificationResult, error) {
	if u.logger != nil {
		u.logger.Info(ctx, "verifying pipeline", "config_path", configPath)
	}
	publishEvent(ctx, u.events, u.logger, ports.EventValidationStarted, map[string]interface{}{
		"config_path": configPath,
	})

	pip, _, err := u.prepareUseCase.Prepare(ctx, configPath)
	if err != nil {
		if u.logger != nil {
			u.logger.Error(ctx, "failed to prepare pipeline for verification", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
			"config_path": configPath,
			"error":       err,
		})
		return nil, nil, err
	}

	results, err := u.executor.Verify(ctx, pip)
	if err != nil {
		if u.logger != nil {
			u.logger.Error(ctx, "pipeline verification failed", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"error":       err,
		})
		return pip, results, err
	}

	if u.logger != nil {
		u.logger.Info(ctx, "pipeline verification complete", "config_path", configPath)
	}
	passed, failed, unknown := summarizeVerificationResults(results)
	publishEvent(ctx, u.events, u.logger, ports.EventValidationCompleted, map[string]interface{}{
		"config_path": configPath,
		"pipeline":    pip.Name,
		"passed":      passed,
		"failed":      failed,
		"unknown":     unknown,
	})
	return pip, results, nil
}

func summarizeVerificationResults(results []pipeline.VerificationResult) (int, int, int) {
	var passed, failed, unknown int
	for _, result := range results {
		switch result.Status {
		case pipeline.VerificationSatisfied:
			passed++
		case pipeline.VerificationFailed:
			failed++
		default:
			unknown++
		}
	}
	return passed, failed, unknown
}

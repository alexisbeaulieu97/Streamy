package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/pipelineconv"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/tui/dashboard"
)

type dashboardPipelineAdapter struct {
	applyUseCase  *applicationpipeline.ApplyUseCase
	verifyUseCase *applicationpipeline.VerifyUseCase
	events        ports.EventPublisher
	progress      *stepProgress
	progressCh    chan dashboard.StepProgressMsg
}

func newDashboardPipelineService(apply *applicationpipeline.ApplyUseCase, verify *applicationpipeline.VerifyUseCase, publisher ports.EventPublisher) (dashboard.PipelineService, error) {
	adapter := &dashboardPipelineAdapter{
		applyUseCase:  apply,
		verifyUseCase: verify,
		events:        publisher,
		progress:      newStepProgress(),
	}
	if adapter.events != nil {
		adapter.progressCh = make(chan dashboard.StepProgressMsg, 32)
		if err := adapter.subscribeToEvents(); err != nil {
			return nil, err
		}
	}

	return adapter, nil
}

func (a *dashboardPipelineAdapter) Verify(ctx context.Context, opts dashboard.VerifyOptions) (*registry.ExecutionResult, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	pipeline, results, err := a.verifyUseCase.Verify(ctx, opts.ConfigPath)
	if err != nil {
		return nil, wrapDashboardError("verify", opts.ConfigPath, err)
	}

	summary := pipelineconv.BuildVerificationSummary(pipeline, results)
	result := pipelineconv.SummaryToExecutionResult(summary, opts.ConfigPath)

	result.PipelineID = opts.ConfigPath
	if pipeline != nil && strings.TrimSpace(pipeline.Name) != "" {
		result.PipelineID = pipeline.Name
	}

	return result, nil
}

func (a *dashboardPipelineAdapter) Apply(ctx context.Context, opts dashboard.ApplyOptions) (*registry.ExecutionResult, error) {
	pipeline, stepResults, _, err := a.applyUseCase.Apply(ctx, opts.ConfigPath, opts.DryRun)
	if err != nil {
		return nil, wrapDashboardError("apply", opts.ConfigPath, err)
	}

	execResult := pipelineconv.ConvertApplyResults(stepResults, opts.ConfigPath, opts.DryRun, nil, nil)

	execResult.PipelineID = opts.ConfigPath
	if pipeline != nil && strings.TrimSpace(pipeline.Name) != "" {
		execResult.PipelineID = pipeline.Name
	}

	return execResult, nil
}

func (a *dashboardPipelineAdapter) StepProgressCmd() tea.Cmd {
	if a.progressCh == nil {
		return nil
	}

	return func() tea.Msg {
		select {
		case msg, ok := <-a.progressCh:
			if !ok {
				a.progressCh = nil
				return dashboard.StepProgressChannelClosedMsg{}
			}

			return msg
		case <-time.After(250 * time.Millisecond):
			return dashboard.StepProgressTimeoutMsg{}
		}
	}
}

func (a *dashboardPipelineAdapter) subscribeToEvents() error {
	if a.events == nil {
		return nil
	}

	handler := func(_ context.Context, event ports.DomainEvent) error {
		payload, ok := event.Payload().(map[string]interface{})
		if !ok {
			return nil
		}

		pipelineID, _ := payload["pipeline"].(string)
		if pipelineID == "" {
			pipelineID, _ = payload["pipeline_id"].(string)
		}

		stepID, _ := payload["step_id"].(string)

		message := ""
		if errVal, ok := payload["error"]; ok {
			message = fmt.Sprint(errVal)
		}

		status := strings.TrimPrefix(event.EventType(), "step.")

		a.progress.Set(pipelineID, dashboard.StepProgress{
			StepID:   stepID,
			Status:   status,
			Message:  message,
			Recorded: time.Now(),
		})

		msg := dashboard.StepProgressMsg{
			PipelineID: pipelineID,
			StepID:     stepID,
			Status:     status,
			Message:    message,
		}
		select {
		case a.progressCh <- msg:
		default:
		}

		return nil
	}

	var errs []error

	for _, eventType := range []string{
		ports.EventStepStarted,
		ports.EventStepCompleted,
		ports.EventStepFailed,
		ports.EventStepSkipped,
	} {
		if _, err := a.events.Subscribe(eventType, handler); err != nil {
			errs = append(errs, fmt.Errorf("subscribe to %s: %w", eventType, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type stepProgress struct {
	mu   sync.RWMutex
	data map[string]dashboard.StepProgress
}

func newStepProgress() *stepProgress {
	return &stepProgress{data: make(map[string]dashboard.StepProgress)}
}

func (s *stepProgress) Set(pipelineID string, progress dashboard.StepProgress) {
	s.mu.Lock()
	s.data[pipelineID] = progress
	s.mu.Unlock()
}

func (s *stepProgress) Get(pipelineID string) (dashboard.StepProgress, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	progress, ok := s.data[pipelineID]

	return progress, ok
}

func wrapDashboardError(operation, configPath string, err error) error {
	return &commandError{
		operation:  operation,
		context:    fmt.Sprintf("processing pipeline %q", configPath),
		cause:      err,
		suggestion: "Inspect the detailed error output above, resolve the issue, then retry.",
	}
}

package dashboard

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// loadInitialStatusCmd loads cached statuses for pipelines
func loadInitialStatusCmd(pipelines []registry.Pipeline, cache *registry.StatusCache) tea.Cmd {
	return func() tea.Msg {
		statuses := make(map[string]registry.CachedStatus)
		for _, p := range pipelines {
			if cached, ok := cache.Get(p.ID); ok {
				statuses[p.ID] = cached
			}
		}
		return InitialStatusLoadedMsg{Statuses: statuses}
	}
}

// verifyCmd runs verification for a pipeline asynchronously
func verifyCmd(ctx context.Context, pipelineID string, configPath string, svc PipelineService) tea.Cmd {
	return func() tea.Msg {
		result, err := svc.Verify(ctx, VerifyOptions{
			ConfigPath: configPath,
			Timeout:    30 * time.Second,
		})

		if err != nil {
			// Context cancellation
			if ctx.Err() != nil {
				return VerifyCancelledMsg{PipelineID: pipelineID}
			}

			return VerifyErrorMsg{
				PipelineID: pipelineID,
				Error:      err,
			}
		}

		if result == nil {
			return VerifyErrorMsg{
				PipelineID: pipelineID,
				Error:      fmt.Errorf("verification produced no result"),
			}
		}

		return VerifyCompleteMsg{
			PipelineID: pipelineID,
			Result:     result,
		}
	}
}

// applyCmd runs apply for a pipeline asynchronously
func applyCmd(ctx context.Context, pipelineID string, configPath string, svc PipelineService) tea.Cmd {
	return func() tea.Msg {
		result, err := svc.Apply(ctx, ApplyOptions{
			ConfigPath:      configPath,
			ContinueOnError: false,
		})

		if err != nil {
			// Context cancellation
			if ctx.Err() != nil {
				return ApplyCancelledMsg{PipelineID: pipelineID}
			}

			return ApplyErrorMsg{
				PipelineID: pipelineID,
				Error:      err,
			}
		}

		if result == nil {
			return ApplyErrorMsg{
				PipelineID: pipelineID,
				Error:      fmt.Errorf("apply produced no result"),
			}
		}

		return ApplyCompleteMsg{
			PipelineID: pipelineID,
			Result:     result,
		}
	}
}

// refreshAllCmd runs verification for all pipelines in parallel
func refreshAllCmd(ctx context.Context, pipelines []registry.Pipeline, _ PipelineService) tea.Cmd {
	return func() tea.Msg {
		return RefreshStartedMsg{Total: len(pipelines)}
	}
}

// refreshSingleCmd runs verification for a single pipeline during refresh all
func refreshSingleCmd(ctx context.Context, pl registry.Pipeline, svc PipelineService, index int, total int) tea.Cmd {
	return func() tea.Msg {
		result, err := svc.Verify(ctx, VerifyOptions{
			ConfigPath: pl.Path,
			Timeout:    30 * time.Second,
		})

		if err != nil {
			if ctx.Err() != nil {
				return RefreshCancelledMsg{}
			}

			return RefreshPipelineCompleteMsg{
				PipelineID: pl.ID,
				Index:      index,
				Total:      total,
				Result:     nil,
				Error:      err,
			}
		}

		if result == nil {
			return RefreshPipelineCompleteMsg{
				PipelineID: pl.ID,
				Index:      index,
				Total:      total,
				Result:     nil,
				Error:      fmt.Errorf("verification produced no result"),
			}
		}

		return RefreshPipelineCompleteMsg{
			PipelineID: pl.ID,
			Index:      index,
			Total:      total,
			Result:     result,
			Error:      nil,
		}
	}
}

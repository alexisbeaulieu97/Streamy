package dashboard

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
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
func verifyCmd(ctx context.Context, pipelineID string, configPath string, pluginReg *plugin.PluginRegistry) tea.Cmd {
	return func() tea.Msg {
		// Run verification
		result, err := engine.VerifyPipeline(ctx, configPath, pluginReg)
		
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
		
		return VerifyCompleteMsg{
			PipelineID: pipelineID,
			Result:     result,
		}
	}
}

// applyCmd runs apply for a pipeline asynchronously
func applyCmd(ctx context.Context, pipelineID string, configPath string, pluginReg *plugin.PluginRegistry) tea.Cmd {
	return func() tea.Msg {
		// Run apply
		result, err := engine.ApplyPipeline(ctx, configPath, pluginReg)
		
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
		
		return ApplyCompleteMsg{
			PipelineID: pipelineID,
			Result:     result,
		}
	}
}

// refreshAllCmd runs verification for all pipelines in parallel
func refreshAllCmd(ctx context.Context, pipelines []registry.Pipeline, pluginReg *plugin.PluginRegistry) tea.Cmd {
	return func() tea.Msg {
		return RefreshStartedMsg{Total: len(pipelines)}
	}
}

// refreshSingleCmd runs verification for a single pipeline during refresh all
func refreshSingleCmd(ctx context.Context, pipeline registry.Pipeline, pluginReg *plugin.PluginRegistry, index int, total int) tea.Cmd {
	return func() tea.Msg {
		result, err := engine.VerifyPipeline(ctx, pipeline.Path, pluginReg)
		
		if err != nil {
			if ctx.Err() != nil {
				return RefreshCancelledMsg{}
			}
			
			return RefreshPipelineCompleteMsg{
				PipelineID: pipeline.ID,
				Index:      index,
				Total:      total,
				Result:     nil,
				Error:      err,
			}
		}
		
		return RefreshPipelineCompleteMsg{
			PipelineID: pipeline.ID,
			Index:      index,
			Total:      total,
			Result:     result,
			Error:      nil,
		}
	}
}

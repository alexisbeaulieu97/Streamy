package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type removeOptions struct {
	force bool
}

func newRemoveCmd(_ *rootFlags, app *AppContext) *cobra.Command {
	opts := &removeOptions{}

	cmd := &cobra.Command{
		Use:   "remove <pipeline-id>",
		Short: "Remove a pipeline from the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, logger := app.CommandContext(cmd, "command.registry.remove")
			if logger != nil {
				logger.Info(ctx, "removing pipeline", "pipeline_id", args[0], "force", opts.force)
			}

			err := runRemove(ctx, logger, cmd, args[0], opts)
			if err != nil && logger != nil {
				logger.Error(ctx, "remove command failed", "pipeline_id", args[0], "error", err)
			}

			return err
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Remove without confirmation")

	return cmd
}

func runRemove(ctx context.Context, logger ports.Logger, cmd *cobra.Command, pipelineID string, opts *removeOptions) error {
	if err := validatePipelineID(pipelineID); err != nil {
		return err
	}

	paths, err := resolveRegistryPaths()
	if err != nil {
		return err
	}

	reg, err := registry.NewRegistry(paths.registry)
	if err != nil {
		return newCommandError("remove", "loading registry", err, "Check registry file permissions and try again.")
	}

	pipeline, err := reg.Get(pipelineID)
	if err != nil {
		return newCommandError("remove", fmt.Sprintf("looking up pipeline %q", pipelineID), err, "Run 'streamy registry list' to view registered pipelines.")
	}

	if proceed, err := confirmRemovalIfNeeded(ctx, logger, cmd, pipelineID, pipeline.Name, opts.force); err != nil || !proceed {
		return err
	}

	if err := removePipelineFromRegistry(ctx, logger, reg, pipelineID); err != nil {
		return err
	}

	invalidateStatusCache(ctx, logger, paths.statusCache, pipelineID)

	announceRemoval(cmd, pipelineID, pipeline.Path)

	if logger != nil {
		logger.Info(ctx, "pipeline removed", "pipeline_id", pipelineID, "config_path", pipeline.Path)
	}

	return nil
}

type registryPaths struct {
	registry    string
	statusCache string
}

func validatePipelineID(pipelineID string) error {
	if strings.TrimSpace(pipelineID) != "" {
		return nil
	}

	return newCommandError("remove", "validating pipeline ID", errors.New("pipeline ID cannot be empty"), "Provide the pipeline ID you wish to remove.")
}

func resolveRegistryPaths() (registryPaths, error) {
	registryPath, err := defaultRegistryPath()
	if err != nil {
		return registryPaths{}, newCommandError("remove", "determining registry path", err, "Ensure your HOME directory is set correctly.")
	}

	statusPath, err := defaultStatusCachePath()
	if err != nil {
		return registryPaths{}, newCommandError("remove", "determining status cache path", err, "Ensure your HOME directory is set correctly.")
	}

	return registryPaths{registry: registryPath, statusCache: statusPath}, nil
}

func confirmRemovalIfNeeded(ctx context.Context, logger ports.Logger, cmd *cobra.Command, pipelineID, pipelineName string, force bool) (bool, error) {
	if force {
		return true, nil
	}

	confirmed, err := confirmRemoval(cmd, pipelineID, pipelineName)
	if err != nil {
		return false, err
	}

	if confirmed {
		return true, nil
	}

	if logger != nil {
		logger.Info(ctx, "pipeline removal cancelled", "pipeline_id", pipelineID)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")

	return false, nil
}

func removePipelineFromRegistry(ctx context.Context, logger ports.Logger, reg *registry.Registry, pipelineID string) error {
	if err := reg.Remove(pipelineID); err != nil {
		if logger != nil {
			logger.Error(ctx, "failed to remove pipeline", "pipeline_id", pipelineID, "error", err)
		}

		return newCommandError("remove", fmt.Sprintf("removing pipeline %q", pipelineID), err, "Verify the pipeline still exists using 'streamy registry list'.")
	}

	if err := reg.Save(); err != nil {
		if logger != nil {
			logger.Error(ctx, "failed to save registry after removal", "pipeline_id", pipelineID, "error", err)
		}

		return newCommandError("remove", "saving registry", err, "Check disk space and file permissions, then retry.")
	}

	return nil
}

func invalidateStatusCache(ctx context.Context, logger ports.Logger, statusPath, pipelineID string) {
	statusCache, err := registry.NewStatusCache(statusPath)
	if err != nil {
		if logger != nil {
			logger.Warn(ctx, "failed to load status cache", "path", statusPath, "error", err)
		}

		return
	}

	if err := statusCache.Invalidate(pipelineID); err != nil {
		if logger != nil {
			logger.Warn(ctx, "failed to update status cache", "pipeline_id", pipelineID, "error", err)
		}
	}
}

func announceRemoval(cmd *cobra.Command, pipelineID, configPath string) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed pipeline '%s'\n", pipelineID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nThe configuration file at %s was not deleted.\n", configPath)
}

func confirmRemoval(cmd *cobra.Command, pipelineID, pipelineName string) (bool, error) {
	if !isTerminal(cmd.InOrStdin()) {
		return false, newCommandError("remove", "prompting for confirmation", errors.New("not a terminal"), "Use --force when running in non-interactive environments.")
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Remove pipeline '%s' (%s) from registry? [y/N]: ", pipelineID, pipelineName)

	scanner := bufio.NewScanner(cmd.InOrStdin())
	if !scanner.Scan() {
		if scanErr := scanner.Err(); scanErr != nil {
			return false, newCommandError(
				"remove",
				"reading confirmation response",
				scanErr,
				"Use --force to skip the confirmation prompt when running non-interactively.",
			)
		}

		return false, nil
	}

	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))

	return answer == "y" || answer == "yes", nil
}

func isTerminal(reader any) bool {
	if file, ok := reader.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}

	return false
}

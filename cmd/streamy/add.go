package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type addOptions struct {
	id          string
	name        string
	description string
	verbose     bool
}

func newAddCmd(rootFlags *rootFlags, app *AppContext) *cobra.Command {
	opts := &addOptions{}

	cmd := &cobra.Command{
		Use:   "add <config-path>",
		Short: "Add a Streamy pipeline to the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.verbose = rootFlags.verbose
			ctx, logger := app.CommandContext(cmd, "command.registry.add")
			if logger != nil {
				logger.Info(ctx, "adding pipeline", "config_path", args[0])
			}
			err := runAdd(ctx, logger, cmd, args[0], opts)
			if err != nil && logger != nil {
				logger.Error(ctx, "add command failed", "config_path", args[0], "error", err)
			}
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.id, "id", "i", "", "Pipeline ID (auto-generated if omitted)")
	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "Pipeline name (defaults to filename)")
	cmd.Flags().StringVarP(&opts.description, "description", "d", "", "Optional description")

	return cmd
}

func runAdd(ctx context.Context, logger ports.Logger, cmd *cobra.Command, configPath string, opts *addOptions) error {
	absPath, err := validateAndNormalizePath(configPath)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "invalid config path", "config_path", configPath, "error", err)
		}
		return newCommandError("add", fmt.Sprintf("resolving config path %q", configPath), err, "Check that the file exists and you have permission to read it.")
	}

	if opts.name == "" {
		opts.name = deriveNameFromPath(absPath)
	}

	if opts.id == "" {
		opts.id = registry.GeneratePipelineID(absPath)
	}

	if err := registry.ValidatePipelineID(opts.id); err != nil {
		if logger != nil {
			logger.Error(ctx, "invalid pipeline id", "pipeline_id", opts.id, "error", err)
		}
		return newCommandError("add", "validating pipeline ID", err, "Provide an ID using lowercase letters, numbers, and hyphens. IDs must start and end with alphanumeric characters.")
	}

	if opts.verbose {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "→ Validating config file: %s\n", absPath)
	}

	cfg, err := config.ParseConfig(absPath)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "configuration validation failed", "config_path", absPath, "error", err)
		}
		return newCommandError("add", "validating configuration", err, "Fix the configuration errors shown above and try again.")
	}

	registryPath, err := defaultRegistryPath()
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "registry path resolution failed", "error", err)
		}
		return newCommandError("add", "determining registry path", err, "Ensure your HOME directory is set correctly.")
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "registry load failed", "path", registryPath, "error", err)
		}
		return newCommandError("add", "loading registry", err, "Check that you have write access to the registry directory.")
	}

	newPipeline := registry.Pipeline{
		ID:           opts.id,
		Name:         opts.name,
		Path:         absPath,
		Description:  opts.description,
		RegisteredAt: time.Now().UTC(),
	}

	if err := reg.Add(newPipeline); err != nil {
		if logger != nil {
			logger.Error(ctx, "registry add failed", "pipeline_id", opts.id, "error", err)
		}
		return newCommandError("add", fmt.Sprintf("adding pipeline %q", opts.id), err, "Use a different ID or remove the existing pipeline first.")
	}

	if err := reg.Save(); err != nil {
		if logger != nil {
			logger.Error(ctx, "registry save failed", "pipeline_id", opts.id, "error", err)
		}
		return newCommandError("add", "saving registry", err, "Check disk space and file permissions, then retry.")
	}

	if opts.verbose {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "✓ Added pipeline %q (%s)\n", newPipeline.ID, newPipeline.Name)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Added pipeline '%s' (%s)\n", newPipeline.ID, newPipeline.Name)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Path: %s\n", newPipeline.Path)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  ID:   %s\n", newPipeline.ID)

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nRun 'streamy registry refresh "+newPipeline.ID+"' to verify its current status.")

	if logger != nil {
		logger.Info(ctx, "pipeline registered", "pipeline_id", newPipeline.ID, "config_path", absPath)
	}

	_ = cfg // Ensures validation executed

	return nil
}

func validateAndNormalizePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("config path cannot be empty")
	}

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, not a file", absPath)
	}

	return absPath, nil
}

func deriveNameFromPath(path string) string {
	base := filepath.Base(path)
	if idx := strings.LastIndex(base, "."); idx > 0 {
		base = base[:idx]
	}
	return strings.TrimSpace(base)
}

func newCommandError(operation, context string, cause error, suggestion string) error {
	return &commandError{operation: operation, context: context, cause: cause, suggestion: suggestion}
}

type commandError struct {
	operation  string
	context    string
	cause      error
	suggestion string
}

func (e *commandError) Error() string {
	return fmt.Sprintf("Failed to %s: %s\n\nError: %v\n\nSuggestion: %s", e.operation, e.context, e.cause, e.suggestion)
}

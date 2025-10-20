// Package main wires the Streamy CLI add command.
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

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type addOptions struct {
	id          string
	name        string
	description string
	verbose     bool
}

type addOperation struct {
	ctx      context.Context
	logger   ports.Logger
	app      *AppContext
	cmd      *cobra.Command
	opts     *addOptions
	absPath  string
	prepared *domainpipeline.Pipeline
	registry *registry.Registry
}

func newAddOperation(ctx context.Context, logger ports.Logger, app *AppContext, cmd *cobra.Command, opts *addOptions) *addOperation {
	return &addOperation{
		ctx:    ctx,
		logger: logger,
		app:    app,
		cmd:    cmd,
		opts:   opts,
	}
}

func (op *addOperation) fail(logMsg, commandContext string, err error, suggestion string, fields ...interface{}) error {
	if op.logger != nil {
		op.logger.Error(op.ctx, logMsg, append(fields, "error", err)...)
	}

	return newCommandError("add", commandContext, err, suggestion)
}

func (op *addOperation) resolvePath(configPath string) error {
	absPath, err := validateAndNormalizePath(configPath)
	if err != nil {
		return op.fail(
			"invalid config path",
			fmt.Sprintf("resolving config path %q", configPath),
			err,
			"Check that the file exists and you have permission to read it.",
			"config_path", configPath,
		)
	}

	op.absPath = absPath

	return nil
}

func (op *addOperation) ensurePipelineID() error {
	if op.opts.id == "" {
		op.opts.id = registry.GeneratePipelineID(op.absPath)
	}

	if err := registry.ValidatePipelineID(op.opts.id); err != nil {
		return op.fail(
			"invalid pipeline id",
			"validating pipeline ID",
			err,
			"Provide an ID using lowercase letters, numbers, and hyphens. IDs must start and end with alphanumeric characters.",
			"pipeline_id", op.opts.id,
		)
	}

	return nil
}

func (op *addOperation) preparePipeline() error {
	if op.opts.verbose {
		_, _ = fmt.Fprintf(op.cmd.ErrOrStderr(), "→ Validating config file: %s\n", op.absPath)
	}

	pipeline, _, err := op.app.PrepareUseCase.Prepare(op.ctx, op.absPath)
	if err != nil {
		return op.fail(
			"configuration validation failed",
			"validating configuration",
			err,
			"Fix the configuration errors shown above and try again.",
			"config_path", op.absPath,
		)
	}

	op.prepared = pipeline

	return nil
}

func (op *addOperation) ensureName() {
	if strings.TrimSpace(op.opts.name) != "" {
		return
	}

	if op.prepared != nil && strings.TrimSpace(op.prepared.Name) != "" {
		op.opts.name = op.prepared.Name
		return
	}

	op.opts.name = deriveNameFromPath(op.absPath)
}

func (op *addOperation) loadRegistry() error {
	path, err := defaultRegistryPath()
	if err != nil {
		return op.fail(
			"registry path resolution failed",
			"determining registry path",
			err,
			"Ensure your HOME directory is set correctly.",
		)
	}

	reg, err := registry.NewRegistry(path)
	if err != nil {
		return op.fail(
			"registry load failed",
			"loading registry",
			err,
			"Check that you have write access to the registry directory.",
			"path", path,
		)
	}

	op.registry = reg

	return nil
}

func (op *addOperation) registerPipeline() (registry.Pipeline, error) {
	newPipeline := registry.Pipeline{
		ID:           op.opts.id,
		Name:         op.opts.name,
		Path:         op.absPath,
		Description:  op.opts.description,
		RegisteredAt: time.Now().UTC(),
	}

	if err := op.registry.Add(newPipeline); err != nil {
		return registry.Pipeline{}, op.fail(
			"registry add failed",
			fmt.Sprintf("adding pipeline %q", op.opts.id),
			err,
			"Use a different ID or remove the existing pipeline first.",
			"pipeline_id", op.opts.id,
		)
	}

	if err := op.registry.Save(); err != nil {
		return registry.Pipeline{}, op.fail(
			"registry save failed",
			"saving registry",
			err,
			"Check disk space and file permissions, then retry.",
			"pipeline_id", op.opts.id,
		)
	}

	return newPipeline, nil
}

func (op *addOperation) printSuccess(newPipeline registry.Pipeline) {
	if op.opts.verbose {
		_, _ = fmt.Fprintf(op.cmd.ErrOrStderr(), "✓ Added pipeline %q (%s)\n", newPipeline.ID, newPipeline.Name)
	}

	stdout := op.cmd.OutOrStdout()
	_, _ = fmt.Fprintf(stdout, "✓ Added pipeline '%s' (%s)\n", newPipeline.ID, newPipeline.Name)
	_, _ = fmt.Fprintf(stdout, "  Path: %s\n", newPipeline.Path)
	_, _ = fmt.Fprintf(stdout, "  ID:   %s\n", newPipeline.ID)
	_, _ = fmt.Fprintln(stdout, "\nRun 'streamy registry refresh "+newPipeline.ID+"' to verify its current status.")

	if op.logger != nil {
		op.logger.Info(op.ctx, "pipeline registered", "pipeline_id", newPipeline.ID, "config_path", newPipeline.Path)
	}
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

			err := runAdd(ctx, logger, app, cmd, args[0], opts)
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

func runAdd(ctx context.Context, logger ports.Logger, app *AppContext, cmd *cobra.Command, configPath string, opts *addOptions) error {
	op := newAddOperation(ctx, logger, app, cmd, opts)

	if err := op.resolvePath(configPath); err != nil {
		return err
	}

	if err := op.ensurePipelineID(); err != nil {
		return err
	}

	if err := op.preparePipeline(); err != nil {
		return err
	}

	op.ensureName()

	if err := op.loadRegistry(); err != nil {
		return err
	}

	newPipeline, err := op.registerPipeline()
	if err != nil {
		return err
	}

	op.printSuccess(newPipeline)

	return nil
}

func validateAndNormalizePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("config path cannot be empty")
	}

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}

		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat path %q: %w", absPath, err)
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

func (e *commandError) Unwrap() error {
	return e.cause
}

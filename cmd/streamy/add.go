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

	"github.com/goccy/go-yaml"
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
	ctx         context.Context
	logger      ports.Logger
	app         *AppContext
	cmd         *cobra.Command
	opts        *addOptions
	absPath     string
	prepared    *domainpipeline.Pipeline
	registry    *registry.Registry
	statusCache *registry.StatusCache
	baseID      string
	version     string
	deps        []string
	canonicalID string
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
	override := strings.TrimSpace(op.opts.id)
	if override == "" {
		return nil
	}

	if op.baseID == "" {
		return nil
	}

	if override != op.baseID {
		return op.fail(
			"mismatched pipeline id",
			"validating --id flag",
			fmt.Errorf("flag id %q does not match config id %q", override, op.baseID),
			"Update the --id flag to match the pipeline's YAML id or omit the flag entirely.",
			"flag_id", override,
			"config_id", op.baseID,
		)
	}

	if err := registry.ValidatePipelineID(override); err != nil {
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

func (op *addOperation) loadStatusCache() error {
	path, err := defaultStatusCachePath()
	if err != nil {
		return op.fail(
			"status cache path resolution failed",
			"determining status cache path",
			err,
			"Ensure your HOME directory is set correctly.",
		)
	}

	cache, err := registry.NewStatusCache(path)
	if err != nil {
		return op.fail(
			"status cache load failed",
			"loading status cache",
			err,
			"Check that you have write access to the status cache directory.",
			"path", path,
		)
	}

	op.statusCache = cache

	return nil
}

func (op *addOperation) registerPipeline() (registry.Pipeline, []string, error) {
	dependencies := append([]string(nil), op.deps...)

	if err := op.registry.ValidateDependencyCycles(op.canonicalID, dependencies); err != nil {
		return registry.Pipeline{}, nil, op.fail(
			"invalid dependencies",
			fmt.Sprintf("validating dependencies for pipeline %q", op.canonicalID),
			err,
			"Remove circular references from the dependency list before retrying.",
			"pipeline_id", op.canonicalID,
			"dependencies", dependencies,
		)
	}

	unregistered := op.registry.FindUnregisteredDependencies(dependencies)

	newPipeline := registry.Pipeline{
		ID:           op.canonicalID,
		Name:         op.opts.name,
		Path:         op.absPath,
		Description:  op.opts.description,
		RegisteredAt: time.Now().UTC(),
		Dependencies: dependencies,
	}

	if len(unregistered) > 0 {
		newPipeline.BlockedBy = append([]string(nil), unregistered...)
		newPipeline.Status = registry.StatusBlocked

		stderr := op.cmd.ErrOrStderr()
		_, _ = fmt.Fprintf(stderr, "⚠ Warning: unregistered dependencies: %v\n", unregistered)
		_, _ = fmt.Fprintln(stderr, "  Pipeline will remain blocked until each dependency is added to the registry.")
	} else {
		newPipeline.Status = registry.StatusReady
	}

	if err := op.registry.Add(newPipeline); err != nil {
		return registry.Pipeline{}, nil, op.fail(
			"registry add failed",
			fmt.Sprintf("adding pipeline %q", op.canonicalID),
			err,
			"Use a different ID or remove the existing pipeline first.",
			"pipeline_id", op.canonicalID,
		)
	}

	unblocked := op.registry.ReconcileStatuses(op.statusCache)

	updatedPipeline, err := op.registry.Get(op.canonicalID)
	if err != nil {
		return registry.Pipeline{}, nil, op.fail(
			"registry lookup failed",
			fmt.Sprintf("retrieving pipeline %q", op.canonicalID),
			err,
			"Retry the operation; if the issue persists, inspect the registry file for corruption.",
			"pipeline_id", op.canonicalID,
		)
	}

	if err := op.registry.Save(); err != nil {
		return registry.Pipeline{}, nil, op.fail(
			"registry save failed",
			"saving registry",
			err,
			"Check disk space and file permissions, then retry.",
			"pipeline_id", op.opts.id,
		)
	}

	if op.statusCache != nil {
		if err := op.statusCache.Save(); err != nil {
			return registry.Pipeline{}, nil, op.fail(
				"status cache save failed",
				"saving status cache",
				err,
				"Check disk space and file permissions, then retry.",
				"pipeline_id", op.canonicalID,
			)
		}
	}

	return updatedPipeline, unblocked, nil
}

func (op *addOperation) printSuccess(newPipeline registry.Pipeline, unblocked []string) {
	if op.opts.verbose {
		_, _ = fmt.Fprintf(op.cmd.ErrOrStderr(), "✓ Added pipeline %q (%s)\n", newPipeline.ID, newPipeline.Name)
	}

	stdout := op.cmd.OutOrStdout()
	_, _ = fmt.Fprintf(stdout, "✓ Added pipeline '%s' (%s)\n", newPipeline.ID, newPipeline.Name)
	_, _ = fmt.Fprintf(stdout, "  Path: %s\n", newPipeline.Path)
	_, _ = fmt.Fprintf(stdout, "  Status: %s %s\n", newPipeline.Status.Icon(), strings.ToUpper(newPipeline.Status.String()))

	if len(newPipeline.BlockedBy) > 0 {
		_, _ = fmt.Fprintln(stdout, "  Missing dependencies:")
		for _, dep := range newPipeline.BlockedBy {
			_, _ = fmt.Fprintf(stdout, "    - %s\n", dep)
		}
	}

	if len(unblocked) > 0 {
		_, _ = fmt.Fprintf(stdout, "✓ Unblocked pipelines: %s\n", strings.Join(unblocked, ", "))
	}

	_, _ = fmt.Fprintln(stdout, "\nRun 'streamy registry refresh "+newPipeline.ID+"' to verify its current status.")

	if op.logger != nil {
		args := []interface{}{"pipeline_id", newPipeline.ID, "config_path", newPipeline.Path}
		if len(unblocked) > 0 {
			args = append(args, "unblocked", unblocked)
		}

		op.logger.Info(op.ctx, "pipeline registered", args...)
	}
}

func (op *addOperation) loadConfigMetadata() error {
	data, err := os.ReadFile(op.absPath)
	if err != nil {
		return op.fail(
			"read pipeline config",
			fmt.Sprintf("reading config %q", op.absPath),
			err,
			"Ensure the pipeline file exists and is readable.",
			"config_path", op.absPath,
		)
	}

	var meta pipelineMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return op.fail(
			"parse pipeline config",
			fmt.Sprintf("parsing config %q", op.absPath),
			err,
			"Fix the YAML syntax issues noted above and try again.",
			"config_path", op.absPath,
		)
	}

	meta.ID = strings.TrimSpace(meta.ID)
	if meta.ID == "" {
		return op.fail(
			"missing pipeline id",
			fmt.Sprintf("validating config %q", op.absPath),
			fmt.Errorf("pipeline id missing"),
			"Add an 'id' field to your pipeline YAML (e.g., id: backend-deploy).",
			"config_path", op.absPath,
		)
	}

	meta.Version = strings.TrimSpace(meta.Version)
	if meta.Version == "" {
		return op.fail(
			"missing pipeline version",
			fmt.Sprintf("validating config %q", op.absPath),
			fmt.Errorf("pipeline version missing"),
			"Add a 'version' field to your pipeline YAML (e.g., version: \"1.0\").",
			"pipeline_id", meta.ID,
			"config_path", op.absPath,
		)
	}

	canonicalID, err := registry.BuildCanonicalPipelineID(meta.ID, meta.Version)
	if err != nil {
		return op.fail(
			"invalid pipeline identifier",
			fmt.Sprintf("validating config %q", op.absPath),
			err,
			"Ensure the 'id' uses lowercase letters, numbers, or hyphens and the 'version' does not contain whitespace or '@'.",
			"pipeline_id", meta.ID,
			"pipeline_version", meta.Version,
		)
	}

	dependencies := make([]string, 0, len(meta.Dependencies))
	for _, raw := range meta.Dependencies {
		dep := strings.TrimSpace(raw)
		if dep == "" {
			return op.fail(
				"invalid dependency identifier",
				fmt.Sprintf("validating dependencies in %q", op.absPath),
				fmt.Errorf("dependency entries cannot be blank"),
				"List each dependency using the canonical <id>@<version> format.",
				"pipeline_id", meta.ID,
			)
		}

		if err := registry.ValidateCanonicalPipelineID(dep); err != nil {
			return op.fail(
				"invalid dependency identifier",
				fmt.Sprintf("validating dependencies in %q", op.absPath),
				err,
				"Declare each dependency in canonical <id>@<version> format (e.g., api@1.2).",
				"dependency", dep,
			)
		}

		dependencies = append(dependencies, dep)
	}

	if strings.TrimSpace(op.opts.description) == "" && strings.TrimSpace(meta.Description) != "" {
		op.opts.description = meta.Description
	}

	op.baseID = meta.ID
	op.version = meta.Version
	op.deps = dependencies
	op.canonicalID = canonicalID

	return nil
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

	if err := op.loadConfigMetadata(); err != nil {
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

	if err := op.loadStatusCache(); err != nil {
		return err
	}

	newPipeline, unblocked, err := op.registerPipeline()
	if err != nil {
		return err
	}

	op.printSuccess(newPipeline, unblocked)

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

type pipelineMetadata struct {
	ID           string   `yaml:"id"`
	Version      string   `yaml:"version"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Dependencies []string `yaml:"dependencies"`
}

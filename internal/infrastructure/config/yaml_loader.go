package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cfgpkg "github.com/alexisbeaulieu97/streamy/internal/config"
	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	apperrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// YAMLLoader implements the ConfigLoader port by reading YAML files from disk.
type YAMLLoader struct {
	logger ports.Logger
}

func NewYAMLLoader(logger ports.Logger) *YAMLLoader {
	return &YAMLLoader{logger: logger}
}

func (l *YAMLLoader) Load(ctx context.Context, path string) (*domain.Pipeline, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, domainError(domain.ErrCodeCancelled, "load cancelled", ctxErr, nil)
	}

	l.logDebug(ctx, "loading pipeline configuration", map[string]interface{}{"path": path})

	cfg, err := cfgpkg.ParseConfig(path)
	if err != nil {
		l.logError(ctx, "failed to parse configuration", err, map[string]interface{}{"path": path})
		return nil, convertError(err, path)
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, domainError(domain.ErrCodeCancelled, "load cancelled", ctxErr, nil)
	}

	domainPipeline := mapToDomain(cfg)
	if err := domainPipeline.Validate(); err != nil {
		l.logError(ctx, "configuration failed domain validation", err, map[string]interface{}{"path": path})
		return nil, err
	}

	l.logInfo(ctx, "pipeline configuration loaded", map[string]interface{}{"path": path, "steps": len(domainPipeline.Steps)})
	return domainPipeline, nil
}

func (l *YAMLLoader) Validate(ctx context.Context, path string) error {
	if err := contextCheck(ctx); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		l.logError(ctx, "configuration path stat failed", err, map[string]interface{}{"path": path})
		return convertError(err, path)
	}
	if info.IsDir() {
		return domainError(domain.ErrCodeValidation, "configuration path is a directory", nil, map[string]interface{}{"path": path})
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		l.logDebug(ctx, "validating pipeline configuration", map[string]interface{}{"path": path})
		_, err = l.Load(ctx, path)
	default:
		err = domainError(domain.ErrCodeValidation, "unsupported configuration file extension", nil, map[string]interface{}{"path": path, "extension": ext})
	}

	return err
}

var _ ports.ConfigLoader = (*YAMLLoader)(nil)

func convertError(err error, path string) error {
	if err == nil {
		return nil
	}
	var parseErr *apperrors.ParseError
	if errors.As(err, &parseErr) {
		if errors.Is(parseErr.Err, os.ErrNotExist) {
			return domainError(domain.ErrCodeNotFound, "configuration not found", parseErr.Err, map[string]interface{}{"path": path})
		}
		return domainError(domain.ErrCodeValidation, "invalid configuration syntax", err, map[string]interface{}{"path": parseErr.Path, "line": parseErr.Line})
	}
	var valErr *apperrors.ValidationError
	if errors.As(err, &valErr) {
		context := map[string]interface{}{"path": path}
		if valErr.Field != "" {
			context["field"] = valErr.Field
		}
		code := domain.ErrCodeValidation
		msg := strings.ToLower(valErr.Message)
		switch {
		case strings.Contains(msg, "duplicate"):
			code = domain.ErrCodeDuplicate
		case strings.Contains(msg, "depends on"):
			code = domain.ErrCodeDependency
		}
		return domainError(code, valErr.Message, valErr.Err, context)
	}
	if os.IsNotExist(err) {
		return domainError(domain.ErrCodeNotFound, "configuration not found", err, map[string]interface{}{"path": path})
	}
	return domainError(domain.ErrCodeInternal, "configuration load failed", err, map[string]interface{}{"path": path})
}

func contextCheck(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return domainError(domain.ErrCodeCancelled, "operation cancelled", err, nil)
	}
	return nil
}

func domainError(code domain.ErrorCode, message string, cause error, ctx map[string]interface{}) *domain.DomainError {
	return &domain.DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: ctx,
	}
}

func (l *YAMLLoader) logDebug(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.logger == nil {
		return
	}
	l.logger.Debug(ctx, msg, flattenFields(fields)...)
}

func (l *YAMLLoader) logInfo(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.logger == nil {
		return
	}
	l.logger.Info(ctx, msg, flattenFields(fields)...)
}

func (l *YAMLLoader) logError(ctx context.Context, msg string, err error, fields map[string]interface{}) {
	if l.logger == nil {
		return
	}
	payload := make(map[string]interface{}, len(fields)+2)
	for k, v := range fields {
		payload[k] = v
	}
	payload["error"] = err
	l.logger.Error(ctx, msg, flattenFields(payload)...)
}

func flattenFields(fields map[string]interface{}) []interface{} {
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	args := make([]interface{}, 0, len(fields)*2)
	for _, k := range keys {
		args = append(args, k, fields[k])
	}
	return args
}

func mapToDomain(cfg *cfgpkg.Config) *domain.Pipeline {
	if cfg == nil {
		return &domain.Pipeline{}
	}

	steps := make([]domain.Step, len(cfg.Steps))
	for i, step := range cfg.Steps {
		steps[i] = domain.Step{
			ID:            step.ID,
			Name:          step.Name,
			Type:          domain.StepType(step.Type),
			DependsOn:     append([]string(nil), step.DependsOn...),
			Enabled:       step.Enabled,
			VerifyTimeout: step.VerifyTimeout,
			Config:        cloneMap(step.RawConfig()),
		}
	}

	validations := make([]domain.Validation, len(cfg.Validations))
	for i, val := range cfg.Validations {
		validations[i] = domain.Validation{
			Type:   domain.ValidationType(val.Type),
			Config: extractValidationConfig(val),
		}
	}

	settings := domain.Settings{
		Parallel:        cfg.Settings.Parallel,
		Timeout:         cfg.Settings.Timeout,
		ContinueOnError: cfg.Settings.ContinueOnError,
		DryRun:          cfg.Settings.DryRun,
		Verbose:         cfg.Settings.Verbose,
	}

	return &domain.Pipeline{
		Version:     cfg.Version,
		Name:        cfg.Name,
		Description: cfg.Description,
		Settings:    settings,
		Steps:       steps,
		Validations: validations,
	}
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	clone := make(map[string]any, len(src))
	for k, v := range src {
		clone[k] = v
	}
	return clone
}

func extractValidationConfig(val cfgpkg.Validation) map[string]any {
	switch val.Type {
	case "command_exists":
		if val.CommandExists == nil {
			return map[string]any{}
		}
		return cloneMap(map[string]any{"command": val.CommandExists.Command})
	case "file_exists":
		if val.FileExists == nil {
			return map[string]any{}
		}
		return cloneMap(map[string]any{"path": val.FileExists.Path})
	case "path_contains":
		if val.PathContains == nil {
			return map[string]any{}
		}
		return cloneMap(map[string]any{"file": val.PathContains.File, "text": val.PathContains.Text})
	default:
		return map[string]any{}
	}
}

package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	apperrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
	"go.yaml.in/yaml/v3"
)

// YAMLLoader implements the ConfigLoader port by reading YAML files from disk.
type YAMLLoader struct {
	logger ports.Logger
	open   func(string) (io.ReadCloser, error)
}

// NewYAMLLoader constructs a loader that reads pipeline configs from YAML files.
func NewYAMLLoader(logger ports.Logger) *YAMLLoader {
	return &YAMLLoader{
		logger: logger,
		open: func(path string) (io.ReadCloser, error) {
			return os.Open(path) // #nosec G304 -- path originates from user configuration input
		},
	}
}

// Load parses a pipeline configuration from the provided path.
func (l *YAMLLoader) Load(ctx context.Context, path string) (*domain.Pipeline, error) {
	if err := contextDomainError(ctx, "load cancelled", "load timed out", map[string]interface{}{"path": path}); err != nil {
		return nil, err
	}

	l.logDebug(ctx, "loading pipeline configuration", map[string]interface{}{"path": path})

	file, err := l.open(path)
	if err != nil {
		l.logError(ctx, "failed to open configuration", err, map[string]interface{}{"path": path})
		return nil, convertError(err, path)
	}

	defer func() {
		_ = file.Close()
	}()

	pipelineConfig, err := l.parseConfig(ctx, file, path)
	if err != nil {
		return nil, err
	}

	if err := contextDomainError(ctx, "load cancelled", "load timed out", map[string]interface{}{"path": path}); err != nil {
		return nil, err
	}

	l.logInfo(ctx, "pipeline configuration loaded", map[string]interface{}{"path": path, "steps": len(pipelineConfig.Steps)})

	return pipelineConfig, nil
}

// Validate verifies the configuration exists and is a supported YAML file.
func (l *YAMLLoader) Validate(ctx context.Context, path string) error {
	if err := contextCheck(ctx); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		l.logError(ctx, "configuration path stat failed", err, map[string]interface{}{"path": path})

		if errors.Is(err, os.ErrNotExist) {
			return domain.NewNotFoundError("configuration", map[string]interface{}{"path": path})
		}

		return domain.NewValidationError("configuration path inaccessible", map[string]interface{}{
			"path":  path,
			"error": err.Error(),
		})
	}

	if info.IsDir() {
		return domain.NewValidationError("configuration path is a directory", map[string]interface{}{"path": path})
	}

	ext := filepath.Ext(path)
	if ext != ".yaml" && ext != ".yml" {
		return domain.NewValidationError("unsupported configuration file extension", map[string]interface{}{"path": path, "extension": ext})
	}

	// Perform a lightweight syntax check so validation only surfaces permitted error codes.
	l.logDebug(ctx, "validating pipeline configuration", map[string]interface{}{"path": path})

	file, err := l.open(path)
	if err != nil {
		l.logError(ctx, "failed to open configuration for validation", err, map[string]interface{}{"path": path})

		if errors.Is(err, os.ErrNotExist) {
			return domain.NewNotFoundError("configuration", map[string]interface{}{"path": path})
		}

		return domain.NewValidationError("configuration unreadable", map[string]interface{}{
			"path":  path,
			"error": err.Error(),
		})
	}

	defer func() {
		_ = file.Close()
	}()

	reader := io.Reader(file)
	if ctx != nil {
		reader = &ctxAwareReader{ctx: ctx, reader: file}
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return mapValidationReadError(err, path)
	}

	if len(data) == 0 {
		return domain.NewValidationError("configuration file is empty", map[string]interface{}{"path": path})
	}

	if err := yaml.Unmarshal(data, &struct{}{}); err != nil {
		ctxFields := map[string]interface{}{"path": path}
		if line := extractLine(err); line > 0 {
			ctxFields["line"] = line
		}

		return domain.NewValidationError("invalid configuration syntax", ctxFields)
	}

	return nil
}

var _ ports.ConfigLoader = (*YAMLLoader)(nil)

// mapValidationReadError normalizes I/O failures into the limited set of
// DomainError codes that the ConfigLoader.Validate contract permits.
func mapValidationReadError(err error, path string) error {
	var domainErr *domain.DomainError
	if errors.As(err, &domainErr) {
		return domainErr
	}

	if errors.Is(err, context.Canceled) {
		return domain.NewDomainError(domain.ErrCodeCancelled, "validation cancelled", err, map[string]interface{}{"path": path})
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return domain.NewTimeoutError("validation timed out", err, map[string]interface{}{"path": path})
	}

	if os.IsNotExist(err) {
		return domain.NewNotFoundError("configuration", map[string]interface{}{"path": path})
	}

	return domain.NewValidationError("failed to read configuration", map[string]interface{}{
		"path":  path,
		"error": err.Error(),
	})
}

func convertError(err error, path string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return domain.NewTimeoutError("operation timed out", err, map[string]interface{}{"path": path})
	}

	if errors.Is(err, context.Canceled) {
		return domain.NewDomainError(domain.ErrCodeCancelled, "operation cancelled", err, map[string]interface{}{"path": path})
	}

	var parseErr *apperrors.ParseError
	if errors.As(err, &parseErr) {
		ctx := map[string]interface{}{"path": parseErr.Path, "line": parseErr.Line}
		if parseErr.Line == 0 {
			delete(ctx, "line")
		}

		if errors.Is(parseErr.Err, os.ErrNotExist) {
			return domain.NewNotFoundError("configuration", ctx)
		}

		return domain.NewConfigError("invalid configuration syntax", parseErr.Err, ctx)
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

		return domain.NewDomainError(code, valErr.Message, valErr.Err, context)
	}

	if os.IsNotExist(err) {
		return domain.NewNotFoundError("configuration", map[string]interface{}{"path": path})
	}

	return domain.NewInternalError("configuration load failed", err, map[string]interface{}{"path": path})
}

func contextCheck(ctx context.Context) error {
	return contextDomainError(ctx, "operation cancelled", "operation timed out", nil)
}

func contextDomainError(ctx context.Context, cancelMsg, timeoutMsg string, fields map[string]interface{}) error {
	if ctx == nil {
		return nil
	}

	return domainErrorFromContextErr(ctx.Err(), cancelMsg, timeoutMsg, fields)
}

func domainErrorFromContextErr(err error, cancelMsg, timeoutMsg string, fields map[string]interface{}) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return domain.NewTimeoutError(timeoutMsg, err, fields)
	}

	if errors.Is(err, context.Canceled) {
		return domain.NewDomainError(domain.ErrCodeCancelled, cancelMsg, err, fields)
	}

	return domain.NewDomainError(domain.ErrCodeCancelled, cancelMsg, err, fields)
}

func (l *YAMLLoader) parseConfig(ctx context.Context, r io.Reader, path string) (*domain.Pipeline, error) {
	reader := r
	if ctx != nil {
		reader = &ctxAwareReader{ctx: ctx, reader: r}
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		if ctxErr := domainErrorFromContextErr(err, "load cancelled", "load timed out", map[string]interface{}{"path": path}); ctxErr != nil {
			return nil, ctxErr
		}

		l.logError(ctx, "failed to read configuration", err, map[string]interface{}{"path": path})

		return nil, convertError(err, path)
	}

	if err := contextDomainError(ctx, "load cancelled", "load timed out", map[string]interface{}{"path": path}); err != nil {
		return nil, err
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		parseErr := apperrors.NewParseError(path, extractLine(err), err)
		l.logError(ctx, "failed to parse configuration", parseErr, map[string]interface{}{"path": path})

		return nil, convertError(parseErr, path)
	}

	pipelineConfig, err := cfg.toPipeline()
	if err != nil {
		l.logError(ctx, "configuration failed domain validation", err, map[string]interface{}{"path": path})
		return nil, err
	}

	return pipelineConfig, nil
}

type ctxAwareReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *ctxAwareReader) Read(p []byte) (int, error) {
	if err := domainErrorFromContextErr(r.ctx.Err(), "stream read cancelled", "stream read timed out", nil); err != nil {
		return 0, err
	}

	n, readErr := r.reader.Read(p)
	if readErr != nil {
		if errors.Is(readErr, io.EOF) {
			return n, io.EOF
		}

		return n, fmt.Errorf("read configuration stream: %w", readErr)
	}

	return n, nil
}

var yamlLineRegex = regexp.MustCompile(`line (\d+)`)

func extractLine(err error) int {
	if err == nil {
		return 0
	}

	matches := yamlLineRegex.FindStringSubmatch(err.Error())
	if len(matches) != 2 {
		return 0
	}

	var line int
	if _, scanErr := fmt.Sscanf(matches[1], "%d", &line); scanErr != nil {
		return 0
	}

	return line
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

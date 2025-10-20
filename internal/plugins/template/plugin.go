// Package templateplugin renders Go templates to reconcile filesystem content.
package templateplugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/pkg/diff"
)

// Plugin renders Go templates to destination files.
type Plugin struct{}

var _ ports.Plugin = (*Plugin)(nil)

// New constructs a ports-native template plugin.
func New() ports.Plugin {
	return &Plugin{}
}

// Metadata describes the template plugin for registry registration.
func (Plugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "template",
		Name:        "template",
		Version:     "1.0.0",
		Type:        domainplugin.TypeTemplate,
		Description: "Renders Go templates to files with variable substitution.",
	}
}

type evaluationData struct {
	RenderedContent string
	RenderedHash    string
	DesiredMode     os.FileMode
	ExistingHash    string
	ExistingMode    os.FileMode
	ExistingExists  bool
	SourceExists    bool
}

// Evaluate determines whether the rendered template matches the destination.
func (Plugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("template evaluation cancelled", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
		})
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	if _, statErr := os.Stat(cfg.Source); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return &domainpipeline.EvaluationResult{
				RequiresAction: true,
				CurrentState:   string(domainpipeline.VerificationFailed),
				DesiredState:   "template source not found",
				Diff:           fmt.Sprintf("would create from source: %s", cfg.Source),
				InternalData: &evaluationData{
					DesiredMode:    cfg.mode(),
					SourceExists:   false,
					ExistingExists: false,
				},
			}, nil
		}

		return nil, domainpipeline.NewExecutionError("stat template source", statErr, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
			"source":      cfg.Source,
		})
	}

	rendered, renderedHash, renderErr := renderTemplate(cfg)
	if renderErr != nil {
		return nil, domainpipeline.NewExecutionError("render template", renderErr, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
			"source":      cfg.Source,
		})
	}

	desiredMode := cfg.mode()

	existingHash, existingMode, exists, stateErr := destinationState(cfg.Destination)
	if stateErr != nil {
		return nil, domainpipeline.NewExecutionError("inspect destination", stateErr, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
			"destination": cfg.Destination,
		})
	}

	data := &evaluationData{
		RenderedContent: rendered,
		RenderedHash:    renderedHash,
		DesiredMode:     desiredMode,
		ExistingHash:    existingHash,
		ExistingMode:    existingMode,
		ExistingExists:  exists,
		SourceExists:    true,
	}

	if !exists {
		return &domainpipeline.EvaluationResult{
			RequiresAction: true,
			CurrentState:   string(domainpipeline.VerificationFailed),
			DesiredState:   fmt.Sprintf("destination %s does not exist", cfg.Destination),
			Diff:           fmt.Sprintf("would create: %s", cfg.Destination),
			InternalData:   data,
		}, nil
	}

	contentMatches := renderedHash == existingHash
	modeMatches := desiredMode.Perm() == existingMode.Perm()

	if contentMatches && modeMatches {
		return &domainpipeline.EvaluationResult{
			RequiresAction: false,
			CurrentState:   string(domainpipeline.VerificationSatisfied),
			DesiredState:   fmt.Sprintf("template output is up to date: %s", cfg.Destination),
			InternalData:   data,
		}, nil
	}

	diffStr := ""

	if !contentMatches {
		currentContent, readErr := os.ReadFile(cfg.Destination)
		if readErr != nil {
			diffStr = fmt.Sprintf("unable to read existing file: %v", readErr)
		} else {
			diffStr = diff.GenerateUnifiedDiff([]byte(rendered), currentContent, "desired", "current")
		}
	}

	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   string(domainpipeline.VerificationFailed),
		DesiredState:   fmt.Sprintf("template output differs for %s", cfg.Destination),
		Diff:           diffStr,
		InternalData:   data,
	}, nil
}

// Apply renders the template to disk when drift is detected.
func (Plugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("template apply cancelled", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
		})
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data, err := ensureEvaluationData(ctx, evaluation, step)
	if err != nil {
		return nil, err
	}

	if evaluation != nil && !evaluation.RequiresAction {
		return &domainpipeline.StepResult{
			StepID:  step.ID,
			Status:  domainpipeline.StatusAlreadySatisfied,
			Message: "no changes required",
		}, nil
	}

	dir := filepath.Dir(cfg.Destination)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, domainpipeline.NewExecutionError("create destination directory", err, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
			"directory":   dir,
		})
	}

	mode := data.DesiredMode
	if mode == 0 {
		mode = 0o600
	}

	if mode.Perm() > 0o600 {
		mode = (mode &^ os.FileMode(0o777)) | 0o600
	}

	if writeErr := os.WriteFile(cfg.Destination, []byte(data.RenderedContent), mode); writeErr != nil {
		return nil, domainpipeline.NewExecutionError("write template output", writeErr, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
			"destination": cfg.Destination,
		})
	}

	return &domainpipeline.StepResult{
		StepID:  step.ID,
		Status:  domainpipeline.StatusSuccess,
		Message: fmt.Sprintf("rendered template to %s", cfg.Destination),
		Changed: true,
	}, nil
}

type templateConfig struct {
	Source       string
	Destination  string
	Vars         map[string]string
	Env          bool
	AllowMissing bool
	Mode         *uint32
}

func (c templateConfig) mode() os.FileMode {
	if c.Mode != nil && *c.Mode != 0 {
		return os.FileMode(*c.Mode)
	}

	return 0o600
}

func decodeConfig(step domainpipeline.Step) (templateConfig, error) {
	if strings.TrimSpace(step.ID) == "" {
		return templateConfig{}, domainpipeline.NewValidationError("template step missing id", map[string]interface{}{
			"plugin_type": string(domainplugin.TypeTemplate),
		})
	}

	if step.Config == nil {
		return templateConfig{}, domainpipeline.NewValidationError("template configuration missing", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
		})
	}

	source, ok := getString(step.Config, "source")
	if !ok || strings.TrimSpace(source) == "" {
		return templateConfig{}, domainpipeline.NewValidationError("template source is required", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
		})
	}

	dest, ok := getString(step.Config, "destination")
	if !ok || strings.TrimSpace(dest) == "" {
		return templateConfig{}, domainpipeline.NewValidationError("template destination is required", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
		})
	}

	cfg := templateConfig{
		Source:      strings.TrimSpace(source),
		Destination: strings.TrimSpace(dest),
		Env:         true,
		Vars:        map[string]string{},
	}

	if vars, exists := step.Config["vars"]; exists {
		parsed, err := parseVars(vars)
		if err != nil {
			return templateConfig{}, domainpipeline.NewValidationError("template vars must be a map of strings", map[string]interface{}{
				"step_id":     step.ID,
				"plugin_type": string(domainplugin.TypeTemplate),
				"value":       vars,
			})
		}

		cfg.Vars = parsed
	}

	if rawEnv, exists := step.Config["env"]; exists {
		val, err := parseBool(rawEnv)
		if err != nil {
			return templateConfig{}, domainpipeline.NewValidationError("template env must be boolean", map[string]interface{}{
				"step_id":     step.ID,
				"plugin_type": string(domainplugin.TypeTemplate),
				"value":       rawEnv,
			})
		}

		cfg.Env = val
	}

	if rawAllowMissing, exists := step.Config["allow_missing"]; exists {
		val, err := parseBool(rawAllowMissing)
		if err != nil {
			return templateConfig{}, domainpipeline.NewValidationError("template allow_missing must be boolean", map[string]interface{}{
				"step_id":     step.ID,
				"plugin_type": string(domainplugin.TypeTemplate),
				"value":       rawAllowMissing,
			})
		}

		cfg.AllowMissing = val
	}

	if rawMode, exists := step.Config["mode"]; exists {
		mode, err := parseMode(rawMode)
		if err != nil {
			return templateConfig{}, domainpipeline.NewValidationError("template mode must be an integer", map[string]interface{}{
				"step_id":     step.ID,
				"plugin_type": string(domainplugin.TypeTemplate),
				"value":       rawMode,
			})
		}

		cfg.Mode = &mode
	}

	return cfg, nil
}

func ensureEvaluationData(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*evaluationData, error) {
	if evaluation != nil {
		if data, ok := evaluation.InternalData.(*evaluationData); ok && data != nil {
			return data, nil
		}
	}

	eval, err := (&Plugin{}).Evaluate(ctx, step)
	if err != nil {
		return nil, err
	}

	data, ok := eval.InternalData.(*evaluationData)
	if !ok || data == nil {
		return nil, domainpipeline.NewInternalError("template evaluation missing internal data", nil, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeTemplate),
		})
	}

	return data, nil
}

func renderTemplate(cfg templateConfig) (string, string, error) {
	content, err := os.ReadFile(cfg.Source)
	if err != nil {
		return "", "", fmt.Errorf("read template file %q: %w", cfg.Source, err)
	}

	if len(cfg.Vars) == 0 {
		if _, err := template.New(cfg.Source).Parse(string(content)); err != nil {
			return "", "", fmt.Errorf("parse template %q: %w", cfg.Source, err)
		}

		str := string(content)

		return str, hash(str), nil
	}

	tmpl, err := template.New(cfg.Source).Parse(string(content))
	if err != nil {
		return "", "", fmt.Errorf("parse template %q: %w", cfg.Source, err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, cfg.Vars); err != nil {
		return "", "", fmt.Errorf("render template %q: %w", cfg.Source, err)
	}

	out := rendered.String()

	return out, hash(out), nil
}

func destinationState(path string) (string, os.FileMode, bool, error) {
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", 0, false, nil
		}

		return "", 0, false, wrapTemplatePathError("stat destination", path, err)
	}

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return "", info.Mode(), true, wrapTemplatePathError("read destination", path, readErr)
	}

	return hash(string(content)), info.Mode(), true, nil
}

func hash(value string) string {
	h := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", h[:])
}

func parseVars(raw interface{}) (map[string]string, error) {
	switch v := raw.(type) {
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, val := range v {
			out[k] = val
		}

		return out, nil
	case map[string]interface{}:
		out := make(map[string]string, len(v))
		for k, val := range v {
			switch typed := val.(type) {
			case string:
				out[k] = typed
			case fmt.Stringer:
				out[k] = typed.String()
			default:
				return nil, fmt.Errorf("invalid var type %T for key %s", val, k)
			}
		}

		return out, nil
	default:
		return nil, fmt.Errorf("invalid vars type %T", raw)
	}
}

func parseBool(raw interface{}) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no", "":
			return false, nil
		default:
			return false, fmt.Errorf("invalid boolean string %q", v)
		}
	default:
		return false, fmt.Errorf("invalid boolean type %T", raw)
	}
}

func parseMode(raw interface{}) (uint32, error) {
	switch v := raw.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative mode value %d", v)
		}

		return normalizeMode(uint64(v))
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("negative mode value %d", v)
		}

		return normalizeMode(uint64(v))
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative mode value %d", v)
		}

		return normalizeMode(uint64(v))
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("negative mode value %f", v)
		}

		frac, _ := math.Modf(v)
		if frac != 0 {
			return 0, fmt.Errorf("mode value %f must be a whole number", v)
		}

		return normalizeMode(uint64(v))
	case string:
		val := strings.TrimSpace(v)
		if val == "" {
			return 0, nil
		}

		if strings.HasPrefix(val, "0") {
			parsed, err := strconv.ParseUint(val, 8, 32)
			if err != nil {
				return 0, fmt.Errorf("parse octal mode %q: %w", val, err)
			}

			return normalizeMode(parsed)
		}

		parsed, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse mode %q: %w", val, err)
		}

		return normalizeMode(parsed)
	default:
		return 0, fmt.Errorf("invalid mode type %T", raw)
	}
}

func getString(values map[string]interface{}, key string) (string, bool) {
	raw, ok := values[key]
	if !ok || raw == nil {
		return "", false
	}

	switch v := raw.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func normalizeMode(val uint64) (uint32, error) {
	const maxMode = 0o777
	if val > maxMode {
		return 0, fmt.Errorf("mode %o exceeds maximum %o", val, maxMode)
	}

	return uint32(val), nil
}

func wrapTemplatePathError(action, path string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s %q: %w", action, path, err)
}

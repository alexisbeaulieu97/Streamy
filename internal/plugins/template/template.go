package templateplugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/pkg/diff"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

const templatePluginType = "template"

type templatePlugin struct{}

// New creates a new instance of the template plugin.
func New() plugin.Plugin {
	return &templatePlugin{}
}

var _ plugin.Plugin = (*templatePlugin)(nil)

func (p *templatePlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "template-renderer",
		Version: "1.0.0",
		Type:    templatePluginType,
	}
}

func (p *templatePlugin) Schema() interface{} {
	return config.TemplateStep{}
}

func (p *templatePlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	cfg := step.Template
	if cfg == nil {
		return false, streamyerrors.NewValidationError(step.ID, "template configuration missing", nil)
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	rendered, err := p.renderTemplate(ctx, cfg)
	if err != nil {
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	renderedHash := hashContent(rendered)
	desiredMode, err := determineFileMode(cfg)
	if err != nil {
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	existingHash, existingMode, exists, err := existingDestinationState(cfg.Destination)
	if err != nil {
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	if !exists {
		return false, nil
	}

	if renderedHash != existingHash {
		return false, nil
	}

	if desiredMode.Perm() != existingMode.Perm() {
		return false, nil
	}

	return true, nil
}

func (p *templatePlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg := step.Template
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "template configuration missing", nil)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if _, err := os.Stat(cfg.Source); err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, fmt.Errorf("stat source %q: %w", cfg.Source, err))
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: execErr.Error(), Error: execErr}, execErr
	}

	rendered, err := p.renderTemplate(ctx, cfg)
	if err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, err)
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: execErr}, execErr
	}

	renderedHash := hashContent(rendered)
	desiredMode, err := determineFileMode(cfg)
	if err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, err)
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: execErr}, execErr
	}

	dstHash, dstMode, dstExists, err := existingDestinationState(cfg.Destination)
	if err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, err)
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: execErr}, execErr
	}

	if dstExists {
		contentMatches := renderedHash == dstHash
		modeMatches := desiredMode.Perm() == dstMode.Perm()

		if contentMatches && modeMatches {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusSkipped,
				Message: "template output already up to date",
			}, nil
		}

		if contentMatches && !modeMatches {
			if err := os.Chmod(cfg.Destination, desiredMode); err != nil {
				execErr := streamyerrors.NewExecutionError(step.ID, fmt.Errorf("chmod destination %q: %w", cfg.Destination, err))
				return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: execErr.Error(), Error: execErr}, execErr
			}
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusSuccess,
				Message: fmt.Sprintf("updated permissions for %s", cfg.Destination),
			}, nil
		}
	}

	dir := filepath.Dir(cfg.Destination)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, fmt.Errorf("create destination directory %q: %w", dir, err))
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: execErr.Error(), Error: execErr}, execErr
	}

	if err := os.WriteFile(cfg.Destination, rendered, desiredMode); err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, fmt.Errorf("write destination %q: %w", cfg.Destination, err))
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: execErr.Error(), Error: execErr}, execErr
	}

	if err := os.Chmod(cfg.Destination, desiredMode); err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, fmt.Errorf("chmod destination %q: %w", cfg.Destination, err))
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: execErr.Error(), Error: execErr}, execErr
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("rendered template to %s", cfg.Destination),
	}, nil
}

func (p *templatePlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg := step.Template
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "template configuration missing", nil)
	}

	rendered, err := p.renderTemplate(ctx, cfg)
	if err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, err)
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: execErr}, execErr
	}

	renderedHash := hashContent(rendered)
	desiredMode, err := determineFileMode(cfg)
	if err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, err)
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: execErr}, execErr
	}

	dstHash, dstMode, dstExists, err := existingDestinationState(cfg.Destination)
	if err != nil {
		execErr := streamyerrors.NewExecutionError(step.ID, err)
		return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: execErr}, execErr
	}

	if !dstExists {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusWouldCreate,
			Message: fmt.Sprintf("would create %s", cfg.Destination),
		}, nil
	}

	contentMatches := renderedHash == dstHash
	modeMatches := desiredMode.Perm() == dstMode.Perm()

	if contentMatches && modeMatches {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusSkipped,
			Message: "template output already up to date",
		}, nil
	}

	message := fmt.Sprintf("would update %s", cfg.Destination)
	if contentMatches && !modeMatches {
		message = fmt.Sprintf("would update permissions for %s", cfg.Destination)
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusWouldUpdate,
		Message: message,
	}, nil
}

func (p *templatePlugin) renderTemplate(ctx context.Context, cfg *config.TemplateStep) ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("template configuration is nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	contents, err := os.ReadFile(cfg.Source)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", cfg.Source, err)
	}

	name := filepath.Base(cfg.Source)
	option := "missingkey=error"
	if cfg.AllowMissing {
		option = "missingkey=zero"
	}

	tmpl, err := template.New(name).Option(option).Parse(string(contents))
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", cfg.Source, err)
	}

	data := p.buildContext(cfg)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %q: %w", cfg.Source, err)
	}

	return buf.Bytes(), nil
}

func (p *templatePlugin) buildContext(cfg *config.TemplateStep) map[string]string {
	values := make(map[string]string, len(cfg.Vars))

	if cfg.Env {
		for _, entry := range os.Environ() {
			parts := strings.SplitN(entry, "=", 2)
			key := parts[0]
			value := ""
			if len(parts) == 2 {
				value = parts[1]
			}
			values[key] = value
		}
	}

	for key, value := range cfg.Vars {
		values[key] = value
	}

	return values
}

func hashContent(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func hashFile(path string) ([32]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	return hashContent(data), nil
}

func existingDestinationState(path string) ([32]byte, os.FileMode, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return [32]byte{}, 0, false, nil
		}
		return [32]byte{}, 0, false, fmt.Errorf("stat destination %q: %w", path, err)
	}

	if info.IsDir() {
		return [32]byte{}, 0, false, fmt.Errorf("destination %q is a directory", path)
	}

	hash, err := hashFile(path)
	if err != nil {
		return [32]byte{}, 0, false, fmt.Errorf("hash destination %q: %w", path, err)
	}

	return hash, info.Mode(), true, nil
}

func determineFileMode(cfg *config.TemplateStep) (os.FileMode, error) {
	if cfg.Mode != nil {
		return os.FileMode(*cfg.Mode), nil
	}

	info, err := os.Stat(cfg.Source)
	if err != nil {
		return 0, fmt.Errorf("stat source %q: %w", cfg.Source, err)
	}

	return info.Mode(), nil
}

func (p *templatePlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
	start := time.Now()
	cfg := step.Template
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "template configuration missing", nil)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   "verification cancelled",
			Error:     ctx.Err(),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	default:
	}

	// Render template in-memory
	rendered, err := p.renderTemplate(ctx, cfg)
	if err != nil {
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   fmt.Sprintf("cannot render template: %v", err),
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// Check if destination exists
	destContent, err := os.ReadFile(cfg.Destination)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.VerificationResult{
				StepID:    step.ID,
				Status:    model.StatusMissing,
				Message:   fmt.Sprintf("destination file %s does not exist", cfg.Destination),
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}, nil
		}
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   fmt.Sprintf("cannot read destination %s: %v", cfg.Destination, err),
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// Compare rendered content with destination
	if !bytes.Equal(rendered, destContent) {
		// Generate diff
		diffOutput := diff.GenerateUnifiedDiff(destContent, rendered, cfg.Destination, "rendered template")

		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusDrifted,
			Message:   fmt.Sprintf("rendered template differs from %s", cfg.Destination),
			Details:   diffOutput,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	return &model.VerificationResult{
		StepID:    step.ID,
		Status:    model.StatusSatisfied,
		Message:   fmt.Sprintf("template output matches %s", cfg.Destination),
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, nil
}

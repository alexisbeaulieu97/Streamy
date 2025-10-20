// Package copyplugin reconciles files and directories for pipeline steps.
package copyplugin

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/pkg/diff"
)

// Plugin implements the copy step using the ports.Plugin interface.
type Plugin struct{}

// New constructs a ports.Plugin implementation.
func New() ports.Plugin {
	return &Plugin{}
}

// Metadata returns the domain metadata for the copy plugin.
func (Plugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "copy",
		Name:        "copy",
		Version:     "1.0.0",
		Type:        domainplugin.TypeCopy,
		Description: "Copies files and directories with permission and backup support.",
	}
}

type copyConfig struct {
	Source          string
	Destination     string
	Recursive       bool
	Overwrite       bool
	PreserveMode    bool
	PreserveModeSet bool
}

type evaluationData struct {
	isDirectory       bool
	isFile            bool
	sourceInfo        os.FileInfo
	sourceHash        string
	destinationExists bool
	destinationInfo   os.FileInfo
	destinationHash   string
	needsRecursive    bool
	preserveMode      bool
	overwrite         bool
}

// Evaluate inspects the current filesystem state.
func (p *Plugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, cancelError(step.ID, err)
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data, err := gatherEvaluationData(ctx, step.ID, cfg)
	if err != nil {
		return nil, err
	}

	if !data.destinationExists {
		return &domainpipeline.EvaluationResult{
			RequiresAction: true,
			CurrentState:   string(domainpipeline.VerificationFailed),
			DesiredState:   fmt.Sprintf("destination %s does not exist", cfg.Destination),
			Diff:           fmt.Sprintf("Would copy: %s -> %s", cfg.Source, cfg.Destination),
			InternalData:   data,
		}, nil
	}

	if data.isDirectory && !data.destinationInfo.IsDir() {
		return &domainpipeline.EvaluationResult{
			RequiresAction: true,
			CurrentState:   string(domainpipeline.VerificationFailed),
			DesiredState:   fmt.Sprintf("destination %s is a file", cfg.Destination),
			Diff:           fmt.Sprintf("Would replace file with directory: %s -> %s", cfg.Source, cfg.Destination),
			InternalData:   data,
		}, nil
	}

	if data.isFile && data.destinationInfo.IsDir() {
		return &domainpipeline.EvaluationResult{
			RequiresAction: true,
			CurrentState:   string(domainpipeline.VerificationFailed),
			DesiredState:   fmt.Sprintf("destination %s is a directory", cfg.Destination),
			Diff:           fmt.Sprintf("Would replace directory with file: %s -> %s", cfg.Source, cfg.Destination),
			InternalData:   data,
		}, nil
	}

	if data.isDirectory && !cfg.Recursive {
		return &domainpipeline.EvaluationResult{
			RequiresAction: false,
			CurrentState:   string(domainpipeline.VerificationUnknown),
			DesiredState:   "source is a directory; enable recursive copy to proceed",
			InternalData:   data,
		}, nil
	}

	if data.isFile {
		if data.sourceHash == data.destinationHash {
			return &domainpipeline.EvaluationResult{
				RequiresAction: false,
				CurrentState:   string(domainpipeline.VerificationSatisfied),
				DesiredState:   "files are identical",
				InternalData:   data,
			}, nil
		}

		if !data.overwrite {
			return &domainpipeline.EvaluationResult{
				RequiresAction: false,
				CurrentState:   string(domainpipeline.VerificationUnknown),
				DesiredState:   "destination exists and overwrite is disabled",
				InternalData:   data,
			}, nil
		}

		diffStr := generateFileDiff(cfg.Source, cfg.Destination)

		return &domainpipeline.EvaluationResult{
			RequiresAction: true,
			CurrentState:   string(domainpipeline.VerificationFailed),
			DesiredState:   "files differ",
			Diff:           diffStr,
			InternalData:   data,
		}, nil
	}

	// Directories can only be approximated; assume drift if recursive copy allowed.
	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   string(domainpipeline.VerificationFailed),
		DesiredState:   "directory contents may differ",
		Diff:           fmt.Sprintf("Would copy directory recursively: %s -> %s", cfg.Source, cfg.Destination),
		InternalData:   data,
	}, nil
}

// Apply performs the actual copy when action is required.
func (p *Plugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, cancelError(step.ID, err)
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data, err := p.ensureEvaluationData(ctx, evaluation, step)
	if err != nil {
		return nil, err
	}

	if evaluation != nil && !evaluation.RequiresAction {
		return &domainpipeline.StepResult{
			StepID:  step.ID,
			Status:  domainpipeline.StatusAlreadySatisfied,
			Message: "destination already matches source",
		}, nil
	}

	if data.isDirectory {
		if !cfg.Recursive {
			return failureResult(step.ID, "recursive copy required for directory contents")
		}

		if err := copyDirectory(ctx, cfg.Source, cfg.Destination, data.preserveMode); err != nil {
			return failureWithError(step.ID, "copy directory", err)
		}
	} else {
		if err := copyFile(ctx, cfg.Source, cfg.Destination, data.preserveMode, data.overwrite); err != nil {
			return failureWithError(step.ID, "copy file", err)
		}
	}

	return &domainpipeline.StepResult{
		StepID:  step.ID,
		Status:  domainpipeline.StatusSuccess,
		Message: fmt.Sprintf("copied %s -> %s", cfg.Source, cfg.Destination),
		Changed: true,
	}, nil
}

func (p *Plugin) ensureEvaluationData(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*evaluationData, error) {
	if evaluation != nil {
		if data, ok := evaluation.InternalData.(*evaluationData); ok && data != nil {
			return data, nil
		}
	}

	evalResult, err := p.Evaluate(ctx, step)
	if err != nil {
		return nil, err
	}

	data, ok := evalResult.InternalData.(*evaluationData)
	if !ok || data == nil {
		return nil, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeExecution,
			"evaluation missing internal data",
			nil,
			map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCopy)},
		)
	}

	return data, nil
}

func decodeConfig(step domainpipeline.Step) (copyConfig, error) {
	if strings.TrimSpace(step.ID) == "" {
		return copyConfig{}, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeValidation,
			"copy step missing id",
			nil,
			map[string]interface{}{"step": step},
		)
	}

	if len(step.Config) == 0 {
		return copyConfig{}, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeValidation,
			"copy configuration missing",
			nil,
			map[string]interface{}{"step_id": step.ID},
		)
	}

	cfg := copyConfig{}

	if value, ok := getString(step.Config, "source"); ok && strings.TrimSpace(value) != "" {
		cfg.Source = filepath.Clean(value)
	} else {
		return copyConfig{}, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeValidation,
			"source is required",
			nil,
			map[string]interface{}{"step_id": step.ID},
		)
	}

	if value, ok := getString(step.Config, "destination"); ok && strings.TrimSpace(value) != "" {
		cfg.Destination = filepath.Clean(value)
	} else {
		return copyConfig{}, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeValidation,
			"destination is required",
			nil,
			map[string]interface{}{"step_id": step.ID},
		)
	}

	if value, ok := getBool(step.Config, "recursive"); ok {
		cfg.Recursive = value
	}

	if value, ok := getBool(step.Config, "overwrite"); ok {
		cfg.Overwrite = value
	}

	if value, ok := getBool(step.Config, "preserve_mode"); ok {
		cfg.PreserveMode = value
		cfg.PreserveModeSet = true
	} else {
		cfg.PreserveMode = true
	}

	return cfg, nil
}

func gatherEvaluationData(ctx context.Context, stepID string, cfg copyConfig) (*evaluationData, error) {
	data := &evaluationData{
		needsRecursive: cfg.Recursive,
		overwrite:      cfg.Overwrite,
		preserveMode:   cfg.PreserveMode,
	}

	srcInfo, err := os.Stat(cfg.Source)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}

		return nil, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeExecution,
			"stat source",
			err,
			map[string]interface{}{"step_id": stepID, "plugin_type": string(domainplugin.TypeCopy), "source": cfg.Source},
		)
	}

	data.sourceInfo = srcInfo
	data.isDirectory = srcInfo.IsDir()
	data.isFile = !srcInfo.IsDir()

	dstInfo, err := os.Stat(cfg.Destination)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, domainpipeline.NewDomainError(
				domainpipeline.ErrCodeExecution,
				"stat destination",
				err,
				map[string]interface{}{"step_id": stepID, "plugin_type": string(domainplugin.TypeCopy), "destination": cfg.Destination},
			)
		}

		data.destinationExists = false
	} else {
		data.destinationExists = true
		data.destinationInfo = dstInfo
	}

	if data.isFile {
		hash, err := hashFile(ctx, cfg.Source)
		if err != nil {
			return nil, domainpipeline.NewDomainError(
				domainpipeline.ErrCodeExecution,
				"hash source",
				err,
				map[string]interface{}{"step_id": stepID, "plugin_type": string(domainplugin.TypeCopy), "source": cfg.Source},
			)
		}

		data.sourceHash = hash

		if data.destinationExists && !data.destinationInfo.IsDir() {
			dstHash, err := hashFile(ctx, cfg.Destination)
			if err != nil {
				return nil, domainpipeline.NewDomainError(
					domainpipeline.ErrCodeExecution,
					"hash destination",
					err,
					map[string]interface{}{"step_id": stepID, "plugin_type": string(domainplugin.TypeCopy), "destination": cfg.Destination},
				)
			}

			data.destinationHash = dstHash
		}
	}

	return data, nil
}

func copyDirectory(ctx context.Context, src, dst string, preserveMode bool) error {
	if err := ctx.Err(); err != nil {
		return wrapCopyCtxError("copy directory cancelled", err)
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return wrapCopyPathError("stat source", src, err)
	}

	perm := srcInfo.Mode().Perm()
	if perm > 0o750 {
		perm = 0o750
	}

	if err := os.MkdirAll(dst, perm); err != nil {
		return wrapCopyPathError("ensure destination directory", dst, err)
	}

	walkErr := filepath.Walk(src, func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return wrapCopyPathError("walk source", path, walkErr)
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return wrapCopyCtxError("copy directory cancelled", ctxErr)
		}

		if path == src {
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return wrapCopyPathError("determine relative path", path, err)
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			mode := info.Mode().Perm()
			if mode > 0o750 {
				mode = 0o750
			}

			if err := os.MkdirAll(dstPath, mode); err != nil {
				return wrapCopyPathError("ensure nested directory", dstPath, err)
			}

			return nil
		}

		return copyFile(ctx, path, dstPath, preserveMode, true)
	})
	if walkErr != nil {
		return wrapCopyPathError("walk source tree", src, walkErr)
	}

	return nil
}

func copyFile(ctx context.Context, src, dst string, preserveMode, overwrite bool) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	if err := ctx.Err(); err != nil {
		return wrapCopyCtxError("copy file cancelled", err)
	}

	if err := ensureDestinationWritable(dst, overwrite); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return wrapCopyPathError("open source", src, err)
	}

	defer func() { _ = srcFile.Close() }()

	if err := ensureDestinationDirectory(dst); err != nil {
		return err
	}

	dstFile, err := openDestinationFile(dst)
	if err != nil {
		return err
	}

	defer func() { _ = dstFile.Close() }()

	if err := streamFile(ctx, src, dst, srcFile, dstFile); err != nil {
		return err
	}

	if preserveMode {
		if err := mirrorFileMode(src, dst); err != nil {
			return err
		}
	}

	return nil
}

func ensureDestinationWritable(dst string, overwrite bool) error {
	if overwrite {
		return nil
	}

	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("destination exists and overwrite is false")
	} else if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return wrapCopyPathError("stat destination", dst, err)
	}

	return nil
}

func ensureDestinationDirectory(dst string) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return wrapCopyPathError("ensure destination directory", dir, err)
	}

	return nil
}

func openDestinationFile(dst string) (*os.File, error) {
	file, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- dst is cleaned and rooted by caller
	if err != nil {
		return nil, wrapCopyPathError("open destination", dst, err)
	}

	return file, nil
}

func streamFile(ctx context.Context, srcPath, dstPath string, srcFile, dstFile *os.File) error {
	buf := make([]byte, 32*1024)

	for {
		if err := ctx.Err(); err != nil {
			_ = dstFile.Close()
			_ = os.Remove(dstPath)

			return wrapCopyCtxError("copy file cancelled", err)
		}

		n, readErr := srcFile.Read(buf)
		if n > 0 {
			if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
				return wrapCopyPathError("write destination", dstPath, writeErr)
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}

			return wrapCopyPathError("read source", srcPath, readErr)
		}
	}

	return nil
}

func mirrorFileMode(src, dst string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return wrapCopyPathError("stat source for mode", src, err)
	}

	_ = os.Chmod(dst, stat.Mode())

	return nil
}

func hashFile(ctx context.Context, path string) (string, error) {
	path = filepath.Clean(path)

	file, err := os.Open(path)
	if err != nil {
		return "", wrapCopyPathError("open file", path, err)
	}

	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	buf := make([]byte, 32*1024)

	for {
		if err := ctx.Err(); err != nil {
			return "", wrapCopyCtxError("hash file cancelled", err)
		}

		n, readErr := file.Read(buf)
		if n > 0 {
			if _, writeErr := hasher.Write(buf[:n]); writeErr != nil {
				return "", fmt.Errorf("hash file write: %w", writeErr)
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}

			return "", wrapCopyPathError("read file", path, readErr)
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func wrapCopyCtxError(action string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", action, err)
}

func wrapCopyPathError(action, path string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s %q: %w", action, path, err)
}

func generateFileDiff(src, dst string) string {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	srcContent, err := os.ReadFile(src)
	if err != nil {
		return fmt.Sprintf("cannot read source file: %v", err)
	}

	dstContent, err := os.ReadFile(dst)
	if err != nil {
		return fmt.Sprintf("cannot read destination file: %v", err)
	}

	diffStr := diff.GenerateUnifiedDiff(srcContent, dstContent, src, dst)
	if diffStr == "" {
		return "files differ"
	}

	return diffStr
}

func getString(values map[string]interface{}, key string) (string, bool) {
	if values == nil {
		return "", false
	}

	val, ok := values[key]
	if !ok {
		return "", false
	}

	switch typed := val.(type) {
	case string:
		return typed, true
	case fmt.Stringer:
		return typed.String(), true
	default:
		return fmt.Sprintf("%v", typed), true
	}
}

func getBool(values map[string]interface{}, key string) (bool, bool) {
	if values == nil {
		return false, false
	}

	val, ok := values[key]
	if !ok {
		return false, false
	}

	switch typed := val.(type) {
	case bool:
		return typed, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		switch lower {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func cancelError(stepID string, err error) *domainpipeline.DomainError {
	return domainpipeline.NewDomainError(
		domainpipeline.ErrCodeCancelled,
		"operation cancelled",
		err,
		map[string]interface{}{"step_id": stepID, "plugin_type": string(domainplugin.TypeCopy)},
	)
}

func failureResult(stepID, message string) (*domainpipeline.StepResult, error) {
	domainErr := domainpipeline.NewDomainError(
		domainpipeline.ErrCodeExecution,
		message,
		nil,
		map[string]interface{}{"step_id": stepID, "plugin_type": string(domainplugin.TypeCopy)},
	)

	return &domainpipeline.StepResult{
		StepID:  stepID,
		Status:  domainpipeline.StatusFailure,
		Error:   domainErr,
		Message: message,
	}, domainErr
}

func failureWithError(stepID, action string, cause error) (*domainpipeline.StepResult, error) {
	domainErr := domainpipeline.NewDomainError(
		domainpipeline.ErrCodeExecution,
		fmt.Sprintf("%s failed", action),
		cause,
		map[string]interface{}{"step_id": stepID, "plugin_type": string(domainplugin.TypeCopy)},
	)

	return &domainpipeline.StepResult{
		StepID:  stepID,
		Status:  domainpipeline.StatusFailure,
		Error:   domainErr,
		Message: domainErr.Error(),
	}, domainErr
}

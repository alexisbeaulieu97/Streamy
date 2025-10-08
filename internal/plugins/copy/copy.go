package copyplugin

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/pkg/diff"
)

// Internal data types for copy operations
type copyEvaluationData struct {
	IsDirectory       bool
	IsFile            bool
	SourceInfo        os.FileInfo
	SourceHash        string
	DestinationExists bool
	DestinationInfo   os.FileInfo
	DestinationHash   string
	NeedsRecursive    bool
	PreserveMode      bool
	Overwrite         bool
}

type copyPlugin struct{}

// New creates a new copy plugin instance.
func New() plugin.Plugin {
	return &copyPlugin{}
}

var _ plugin.Plugin = (*copyPlugin)(nil)

// PluginMetadata describes the plugin for the dependency registry.
//
// The empty Dependencies slice documents that copy does not require other plugins.
// APIVersion pins compatibility with other plugins using the registry-provided interface.
func (p *copyPlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:         "copy",
		Type:         "copy",
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: []plugin.Dependency{},
		Stateful:     false,
		Description:  "Copies files and directories with permission and backup support.",
	}
}

func (p *copyPlugin) Schema() any {
	return config.CopyStep{}
}

func (p *copyPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	// Check context first (only if context is provided)
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	cfg := step.Copy
	if cfg == nil {
		return nil, plugin.NewValidationError(step.ID, fmt.Errorf("copy configuration missing"))
	}

	// Gather evaluation data (read-only)
	data := &copyEvaluationData{
		NeedsRecursive: cfg.Recursive,
		Overwrite:      cfg.Overwrite,
	}

	if cfg.PreserveModeSet {
		data.PreserveMode = cfg.PreserveMode
	} else {
		data.PreserveMode = true // default
	}

	// Check source
	srcInfo, err := os.Stat(cfg.Source)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.EvaluationResult{
				StepID:         step.ID,
				CurrentState:   model.StatusMissing,
				RequiresAction: true,
				Message:        fmt.Sprintf("source %s does not exist", cfg.Source),
				Diff:           fmt.Sprintf("Would copy: %s -> %s", cfg.Source, cfg.Destination),
			}, nil
		}
		return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot stat source: %w", err))
	}
	data.SourceInfo = srcInfo
	data.IsDirectory = srcInfo.IsDir()
	data.IsFile = !srcInfo.IsDir()

	// Check destination
	dstInfo, err := os.Stat(cfg.Destination)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot stat destination: %w", err))
		}
		data.DestinationExists = false
	} else {
		data.DestinationExists = true
		data.DestinationInfo = dstInfo
	}

	// For file operations, compute hashes to compare content
	if data.IsFile {
		srcHash, err := hashFile(cfg.Source)
		if err != nil {
			return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot hash source file: %w", err))
		}
		data.SourceHash = srcHash

		if data.DestinationExists && !data.DestinationInfo.IsDir() {
			dstHash, err := hashFile(cfg.Destination)
			if err != nil {
				return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot hash destination file: %w", err))
			}
			data.DestinationHash = dstHash
		}
	}

	// Determine state and action needed
	if !data.DestinationExists {
		// Destination doesn't exist - need to copy
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusMissing,
			RequiresAction: true,
			Message:        fmt.Sprintf("destination %s does not exist", cfg.Destination),
			Diff:           fmt.Sprintf("Would copy: %s -> %s", cfg.Source, cfg.Destination),
			InternalData:   data,
		}, nil
	}

	// Check if source and destination types match
	if data.IsDirectory && !data.DestinationInfo.IsDir() {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusDrifted,
			RequiresAction: true,
			Message:        fmt.Sprintf("destination %s exists but is a file (source is directory)", cfg.Destination),
			Diff:           fmt.Sprintf("Would replace file with directory: %s -> %s", cfg.Source, cfg.Destination),
			InternalData:   data,
		}, nil
	}

	if data.IsFile && data.DestinationInfo.IsDir() {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusDrifted,
			RequiresAction: true,
			Message:        fmt.Sprintf("destination %s exists but is a directory (source is file)", cfg.Destination),
			Diff:           fmt.Sprintf("Would replace directory with file: %s -> %s", cfg.Source, cfg.Destination),
			InternalData:   data,
		}, nil
	}

	// For directories, check recursive flag
	if data.IsDirectory && !cfg.Recursive {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusBlocked,
			RequiresAction: false,
			Message:        fmt.Sprintf("source %s is a directory; enable recursive copy", cfg.Source),
			InternalData:   data,
		}, nil
	}

	// For files, compare content
	if data.IsFile {
		if data.SourceHash == data.DestinationHash {
			return &model.EvaluationResult{
				StepID:         step.ID,
				CurrentState:   model.StatusSatisfied,
				RequiresAction: false,
				Message:        fmt.Sprintf("files are identical: %s -> %s", cfg.Source, cfg.Destination),
				InternalData:   data,
			}, nil
		}

		if !data.Overwrite {
			return &model.EvaluationResult{
				StepID:         step.ID,
				CurrentState:   model.StatusBlocked,
				RequiresAction: false,
				Message:        "destination exists and overwrite is disabled",
				InternalData:   data,
			}, nil
		}

		// Files differ - need to copy
		diffStr := generateFileDiff(cfg.Source, cfg.Destination)
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusDrifted,
			RequiresAction: true,
			Message:        fmt.Sprintf("files differ: %s -> %s", cfg.Source, cfg.Destination),
			Diff:           diffStr,
			InternalData:   data,
		}, nil
	}

	// For directories, assume they differ (simplified for now)
	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusDrifted,
		RequiresAction: true,
		Message:        fmt.Sprintf("directories may differ: %s -> %s", cfg.Source, cfg.Destination),
		Diff:           fmt.Sprintf("Would copy directory recursively: %s -> %s", cfg.Source, cfg.Destination),
		InternalData:   data,
	}, nil
}

func (p *copyPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	// Use evaluation data to avoid recomputation
	var data *copyEvaluationData
	if evalResult != nil {
		if typed, ok := evalResult.InternalData.(*copyEvaluationData); ok {
			data = typed
		}
	}
	if data == nil {
		// Fallback to recomputing evaluation data
		cfg := step.Copy
		if cfg == nil {
			return nil, plugin.NewValidationError(step.ID, fmt.Errorf("copy configuration missing"))
		}

		// Check source to determine if it's a directory
		srcInfo, err := os.Stat(cfg.Source)
		if err != nil {
			return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("cannot stat source: %w", err))
		}

		data = &copyEvaluationData{
			IsDirectory:    srcInfo.IsDir(),
			IsFile:         !srcInfo.IsDir(),
			SourceInfo:     srcInfo,
			NeedsRecursive: cfg.Recursive,
			Overwrite:      cfg.Overwrite,
			PreserveMode:   true, // default
		}
		if cfg.PreserveModeSet {
			data.PreserveMode = cfg.PreserveMode
		}
	}

	cfg := step.Copy

	// Perform the copy operation
	if data.IsDirectory {
		if !data.NeedsRecursive {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: fmt.Sprintf("source %s is a directory; enable recursive copy", cfg.Source),
				Error:   fmt.Errorf("directory copy requires recursive flag"),
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("directory copy requires recursive flag"))
		}
		if err := copyDirectory(cfg.Source, cfg.Destination, data.PreserveMode); err != nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: fmt.Sprintf("failed to copy directory %s to %s: %v", cfg.Source, cfg.Destination, err),
				Error:   err,
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to copy directory: %w", err))
		}
	} else {
		if err := copyFile(cfg.Source, cfg.Destination, data.PreserveMode, data.Overwrite); err != nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: fmt.Sprintf("failed to copy file %s to %s: %v", cfg.Source, cfg.Destination, err),
				Error:   err,
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to copy file: %w", err))
		}
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("copied %s to %s", cfg.Source, cfg.Destination),
	}, nil
}

// Helper functions

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func generateFileDiff(src, dst string) string {
	// Use the existing diff package for text files
	srcContent, err := os.ReadFile(src)
	if err != nil {
		return fmt.Sprintf("Cannot read source file: %v", err)
	}

	dstContent, err := os.ReadFile(dst)
	if err != nil {
		return fmt.Sprintf("Cannot read destination file: %v", err)
	}

	// Generate unified diff
	diffStr := diff.GenerateUnifiedDiff(srcContent, dstContent, src, dst)
	if diffStr == "" {
		return "Files are identical"
	}

	return diffStr
}

func copyDirectory(src, dst string, preserveMode bool) error {
	// Ensure destination directory exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root source directory to avoid copying it as a subdirectory
		if path == src {
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath, preserveMode, true)
	})
}

func copyFile(src, dst string, preserveMode bool, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("destination file exists and overwrite is false")
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve file mode if requested
	if preserveMode {
		srcInfo, err := os.Stat(src)
		if err != nil {
			return err
		}
		return os.Chmod(dst, srcInfo.Mode())
	}

	return nil
}

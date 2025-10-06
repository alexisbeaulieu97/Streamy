package copyplugin

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/pkg/diff"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type copyPlugin struct{}

// New creates a new copy plugin instance.
func New() plugin.Plugin {
	return &copyPlugin{}
}

func (p *copyPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "file-copy",
		Version: "1.0.0",
		Type:    "copy",
	}
}

func (p *copyPlugin) Schema() interface{} {
	return config.CopyStep{}
}

func (p *copyPlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	cfg := step.Copy
	if cfg == nil {
		return false, streamyerrors.NewValidationError(step.ID, "copy configuration missing", nil)
	}

	srcInfo, err := os.Stat(cfg.Source)
	if err != nil {
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	dstInfo, err := os.Stat(cfg.Destination)
	if err != nil {
		return false, nil
	}

	if srcInfo.IsDir() || dstInfo.IsDir() {
		return false, nil
	}

	srcHash, err := hashFile(cfg.Source)
	if err != nil {
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	dstHash, err := hashFile(cfg.Destination)
	if err != nil {
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	return srcHash == dstHash, nil
}

func (p *copyPlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg := step.Copy
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "copy configuration missing", nil)
	}

	srcInfo, err := os.Stat(cfg.Source)
	if err != nil {
		return nil, streamyerrors.NewExecutionError(step.ID, err)
	}

	preserve := true
	if cfg.PreserveModeSet {
		preserve = cfg.PreserveMode
	}

	if srcInfo.IsDir() {
		if !cfg.Recursive {
			err := fmt.Errorf("source %s is a directory; enable recursive copy", cfg.Source)
			return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: err}, streamyerrors.NewExecutionError(step.ID, err)
		}
		if err := copyDirectory(cfg.Source, cfg.Destination, preserve); err != nil {
			return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: err}, streamyerrors.NewExecutionError(step.ID, err)
		}
	} else {
		if err := copyFile(cfg.Source, cfg.Destination, preserve, cfg.Overwrite); err != nil {
			return &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: err}, streamyerrors.NewExecutionError(step.ID, err)
		}
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: "copy completed",
	}, nil
}

func (p *copyPlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSkipped,
		Message: "dry-run: files not copied",
	}, nil
}

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

func copyFile(src, dst string, preserveMode bool, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("destination %s exists", dst)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	mode := os.FileMode(0o644)
	if preserveMode {
		mode = srcInfo.Mode()
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Ensure mode is set correctly (OpenFile uses umask)
	if preserveMode {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return err
		}
	}

	return nil
}

func copyDirectory(src, dst string, preserveMode bool) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			mode := os.FileMode(0o755)
			if preserveMode {
				mode = info.Mode()
			}
			if err := os.MkdirAll(target, mode); err != nil {
				return err
			}
			if preserveMode {
				if err := os.Chmod(target, info.Mode()); err != nil {
					return err
				}
			}
			return nil
		}

		return copyFile(path, target, preserveMode, true)
	})
}

func (p *copyPlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
	start := time.Now()
	cfg := step.Copy
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "copy configuration missing", nil)
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

	// Check if source exists
	srcInfo, err := os.Stat(cfg.Source)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.VerificationResult{
				StepID:    step.ID,
				Status:    model.StatusBlocked,
				Message:   fmt.Sprintf("source file %s does not exist", cfg.Source),
				Error:     err,
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}, nil
		}
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   fmt.Sprintf("cannot access source %s: %v", cfg.Source, err),
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// Check if destination exists
	dstInfo, err := os.Stat(cfg.Destination)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.VerificationResult{
				StepID:    step.ID,
				Status:    model.StatusMissing,
				Message:   fmt.Sprintf("destination %s does not exist", cfg.Destination),
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}, nil
		}
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   fmt.Sprintf("cannot access destination %s: %v", cfg.Destination, err),
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// Check type mismatch
	if srcInfo.IsDir() != dstInfo.IsDir() {
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusDrifted,
			Message:   "source and destination types do not match (file vs directory)",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// For directories, only check existence (detailed recursive verification is expensive)
	if srcInfo.IsDir() {
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusSatisfied,
			Message:   fmt.Sprintf("directory %s exists at destination", cfg.Destination),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// For files, compare checksums
	srcHash, err := hashFile(cfg.Source)
	if err != nil {
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   fmt.Sprintf("cannot read source file %s: %v", cfg.Source, err),
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	dstHash, err := hashFile(cfg.Destination)
	if err != nil {
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   fmt.Sprintf("cannot read destination file %s: %v", cfg.Destination, err),
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	if srcHash != dstHash {
		// Files differ - generate diff if both are text files and reasonably sized
		var details string
		if srcInfo.Size() < 1024*1024 && dstInfo.Size() < 1024*1024 { // < 1MB
			srcContent, err1 := os.ReadFile(cfg.Source)
			dstContent, err2 := os.ReadFile(cfg.Destination)
			if err1 == nil && err2 == nil {
				details = diff.GenerateUnifiedDiff(dstContent, srcContent, cfg.Destination, cfg.Source)
			}
		}

		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusDrifted,
			Message:   fmt.Sprintf("file content differs (source: %s, destination: %s)", srcHash[:8], dstHash[:8]),
			Details:   details,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	return &model.VerificationResult{
		StepID:    step.ID,
		Status:    model.StatusSatisfied,
		Message:   fmt.Sprintf("file %s matches source %s", cfg.Destination, cfg.Source),
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, nil
}

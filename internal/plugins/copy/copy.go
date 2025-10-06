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
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type copyPlugin struct{}

// New creates a new copy plugin instance.
func New() plugin.Plugin {
	return &copyPlugin{}
}

func init() {
	if err := plugin.RegisterPlugin("copy", New()); err != nil {
		panic(err)
	}
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

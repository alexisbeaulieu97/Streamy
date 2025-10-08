package symlinkplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestSymlinkPlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.PluginMetadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "symlink", meta.Name)
}

func TestSymlinkPlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.SymlinkStep)
	require.True(t, ok, "schema should be of type SymlinkStep")
}

func TestSymlinkPlugin_ApplyCreatesLink(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "file.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))

	step := &config.Step{
		ID:   "link_file",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetDir,
		},
	}

	p := New()

	evalResult := &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusMissing,
		RequiresAction: true,
		Message:        "Symlink needs to be created",
	}

	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	info, err := os.Lstat(targetDir)
	require.NoError(t, err)
	require.True(t, info.Mode()&os.ModeSymlink != 0)

	target, err := os.Readlink(targetDir)
	require.NoError(t, err)
	require.Equal(t, sourceFile, target)
}

func TestSymlinkPlugin_EvaluateCorrectLink(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "file.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))

	// Create correct symlink
	require.NoError(t, os.Symlink(sourceFile, targetDir))

	step := &config.Step{
		ID:   "check_correct_link",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetDir,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "symlink exists and points to correct target")
}

func TestSymlinkPlugin_EvaluateMissingLink(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "file.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))

	step := &config.Step{
		ID:   "check_missing_link",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetDir,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "symlink does not exist")
}

func TestSymlinkPlugin_ApplyOverwritesExisting(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "file.txt")
	otherFile := filepath.Join(sourceDir, "other.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(otherFile, []byte("other"), 0o644))

	// Create existing symlink pointing to wrong target
	require.NoError(t, os.Symlink(otherFile, targetDir))

	step := &config.Step{
		ID:   "overwrite_link",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetDir,
			Force:  true,
		},
	}

	p := New()

	evalResult := &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusDrifted,
		RequiresAction: true,
		Message:        "Symlink points to wrong target",
	}

	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	target, err := os.Readlink(targetDir)
	require.NoError(t, err)
	require.Equal(t, sourceFile, target)
}

func TestSymlinkPlugin_EvaluateForDryRun(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "file.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))

	step := &config.Step{
		ID:   "dry_run_link",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetDir,
		},
	}

	p := New()

	// Test dry run when symlink doesn't exist
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)

	// Test dry run when symlink exists and is correct
	require.NoError(t, os.Symlink(sourceFile, targetDir))
	evalResult, err = p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
}

func TestSymlinkPlugin_ApplyBrokenLink(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "nonexistent.txt")

	step := &config.Step{
		ID:   "broken_link",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetDir,
		},
	}

	p := New()

	evalResult := &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusMissing,
		RequiresAction: true,
		Message:        "Symlink needs to be created",
	}

	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	// Link should exist even if target doesn't
	info, err := os.Lstat(targetDir)
	require.NoError(t, err)
	require.True(t, info.Mode()&os.ModeSymlink != 0)
}

func TestSymlinkPlugin_EvaluateLinkPointsToWrongTarget(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "file.txt")
	otherFile := filepath.Join(sourceDir, "other.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(otherFile, []byte("other"), 0o644))

	// Create symlink pointing to wrong target
	require.NoError(t, os.Symlink(otherFile, targetDir))

	step := &config.Step{
		ID:   "wrong_target_link",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetDir,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "symlink exists but points to wrong target")
}

func TestSymlinkPlugin_EvaluateExistingFileWithoutForce(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "file.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(targetDir, []byte("existing"), 0o644))

	step := &config.Step{
		ID:   "existing_file_no_force",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetDir,
			Force:  false,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusBlocked, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "target exists and is not a symlink")
}

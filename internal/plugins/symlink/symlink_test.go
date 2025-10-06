package symlinkplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestSymlinkPlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.Metadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "symlink", meta.Type)
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
	require.Implements(t, (*pluginpkg.Plugin)(nil), p)

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, res.StepID)
	require.Equal(t, "success", res.Status)

	linkTarget, err := os.Readlink(targetDir)
	require.NoError(t, err)
	require.Equal(t, sourceFile, linkTarget)
}

func TestSymlinkPlugin_CheckDetectsExistingLink(t *testing.T) {
	sourceFile := filepath.Join(t.TempDir(), "source.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("content"), 0o644))
	targetFile := filepath.Join(t.TempDir(), "target.txt")
	require.NoError(t, os.Symlink(sourceFile, targetFile))

	step := &config.Step{
		ID:   "link_file",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: sourceFile,
			Target: targetFile,
		},
	}

	p := New()

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestSymlinkPlugin_ApplyWithForceReplacesExisting(t *testing.T) {
	original := filepath.Join(t.TempDir(), "original.txt")
	require.NoError(t, os.WriteFile(original, []byte("original"), 0o644))

	replacementSource := filepath.Join(t.TempDir(), "new.txt")
	require.NoError(t, os.WriteFile(replacementSource, []byte("new"), 0o644))

	target := filepath.Join(t.TempDir(), "link.txt")
	require.NoError(t, os.WriteFile(target, []byte("stale"), 0o644))

	step := &config.Step{
		ID:   "link_file",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: replacementSource,
			Target: target,
			Force:  true,
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", res.Status)

	linkTarget, err := os.Readlink(target)
	require.NoError(t, err)
	require.Equal(t, replacementSource, linkTarget)
}

func TestSymlinkPlugin_CheckReturnsFalseForNonSymlink(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "source.txt")
	require.NoError(t, os.WriteFile(source, []byte("data"), 0o644))

	target := filepath.Join(t.TempDir(), "target.txt")
	// Create a regular file, not a symlink
	require.NoError(t, os.WriteFile(target, []byte("not a symlink"), 0o644))

	step := &config.Step{
		ID:   "check_link",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: source,
			Target: target,
		},
	}

	p := New()

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok, "expected Check to return false for non-symlink file")
}

func TestSymlinkPlugin_CheckReturnsFalseWhenTargetMissing(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "source.txt")
	target := filepath.Join(t.TempDir(), "missing.txt")

	step := &config.Step{
		ID:   "check_link",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: source,
			Target: target,
		},
	}

	p := New()

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok, "expected Check to return false when target missing")
}

func TestSymlinkPlugin_ApplyWithNestedTargetDirectory(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "source.txt")
	require.NoError(t, os.WriteFile(source, []byte("test"), 0o644))

	// Target in a nested directory that doesn't exist yet
	target := filepath.Join(t.TempDir(), "nested/dir/link.txt")

	step := &config.Step{
		ID:   "link_file",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: source,
			Target: target,
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", res.Status)

	linkTarget, err := os.Readlink(target)
	require.NoError(t, err)
	require.Equal(t, source, linkTarget)
}

func TestSymlinkPlugin_DryRunReportsSkip(t *testing.T) {
	source := filepath.Join(t.TempDir(), "file.txt")
	target := filepath.Join(t.TempDir(), "link.txt")

	step := &config.Step{
		ID:   "link_file",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: source,
			Target: target,
		},
	}

	p := New()

	res, err := p.DryRun(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, res.StepID)
	require.Equal(t, "skipped", res.Status)

	_, err = os.Lstat(target)
	require.Error(t, err)
}

func TestSymlinkPlugin_ApplyFailsWhenTargetExistsWithoutForce(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source.txt")
	require.NoError(t, os.WriteFile(src, []byte("content"), 0o644))

	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	require.NoError(t, os.WriteFile(target, []byte("existing"), 0o644))

	step := &config.Step{
		ID:   "link_file",
		Type: "symlink",
		Symlink: &config.SymlinkStep{
			Source: src,
			Target: target,
			Force:  false,
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.Error(t, err)
	require.Equal(t, "link_file", res.StepID)
	require.Equal(t, "failed", res.Status)
}

func TestSymlinkPlugin_Verify(t *testing.T) {
	t.Run("returns satisfied when symlink exists and matches", func(t *testing.T) {
		source := filepath.Join(t.TempDir(), "source.txt")
		require.NoError(t, os.WriteFile(source, []byte("content"), 0o644))
		target := filepath.Join(t.TempDir(), "target.txt")
		require.NoError(t, os.Symlink(source, target))

		p := New()

		step := &config.Step{
			ID:   "link_file",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: source,
				Target: target,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
		require.Contains(t, result.Message, "correctly points to")
	})

	t.Run("returns missing when symlink does not exist", func(t *testing.T) {
		source := filepath.Join(t.TempDir(), "source.txt")
		target := filepath.Join(t.TempDir(), "target.txt")

		p := New()

		step := &config.Step{
			ID:   "link_file",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: source,
				Target: target,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "missing", string(result.Status))
		require.Contains(t, result.Message, "does not exist")
	})

	t.Run("returns drifted when target is not a symlink", func(t *testing.T) {
		source := filepath.Join(t.TempDir(), "source.txt")
		target := filepath.Join(t.TempDir(), "target.txt")
		require.NoError(t, os.WriteFile(target, []byte("regular file"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "link_file",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: source,
				Target: target,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "drifted", string(result.Status))
		require.Contains(t, result.Message, "is not a symlink")
	})

	t.Run("returns drifted when symlink points to wrong target", func(t *testing.T) {
		source := filepath.Join(t.TempDir(), "source.txt")
		wrongSource := filepath.Join(t.TempDir(), "wrong.txt")
		target := filepath.Join(t.TempDir(), "target.txt")
		require.NoError(t, os.Symlink(wrongSource, target))

		p := New()

		step := &config.Step{
			ID:   "link_file",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: source,
				Target: target,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "drifted", string(result.Status))
		require.Contains(t, result.Message, "points to")
	})

	t.Run("returns blocked when context is cancelled", func(t *testing.T) {
		source := filepath.Join(t.TempDir(), "source.txt")
		target := filepath.Join(t.TempDir(), "target.txt")

		p := New()

		step := &config.Step{
			ID:   "link_file",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: source,
				Target: target,
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := p.Verify(ctx, step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "blocked", string(result.Status))
		require.Contains(t, result.Message, "cancelled")
		require.NotNil(t, result.Error)
	})

	t.Run("returns error when symlink config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:      "link_file",
			Type:    "symlink",
			Symlink: nil,
		}

		_, err := p.Verify(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "symlink configuration missing")
	})
}

func TestSymlinkPlugin_Check_Errors(t *testing.T) {
	t.Run("returns error when symlink config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:      "check_link",
			Type:    "symlink",
			Symlink: nil,
		}

		_, err := p.Check(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "symlink configuration missing")
	})
}

func TestSymlinkPlugin_Apply_Errors(t *testing.T) {
	t.Run("returns error when symlink config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:      "link_file",
			Type:    "symlink",
			Symlink: nil,
		}

		_, err := p.Apply(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "symlink configuration missing")
	})
}

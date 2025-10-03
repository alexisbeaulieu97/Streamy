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

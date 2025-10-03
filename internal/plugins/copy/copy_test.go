package copyplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestCopyPlugin_CheckUsesHashForIdempotency(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "file.txt")
	dstFile := filepath.Join(dstDir, "file.txt")

	require.NoError(t, os.WriteFile(srcFile, []byte("same"), 0o644))
	require.NoError(t, os.WriteFile(dstFile, []byte("same"), 0o644))

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
		},
	}

	p := New()
	require.Implements(t, (*pluginpkg.Plugin)(nil), p)

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, os.WriteFile(dstFile, []byte("different"), 0o644))
	ok, err = p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCopyPlugin_ApplyCopiesFileAndPermissions(t *testing.T) {
	srcDir := t.TempDir()
	dstFile := filepath.Join(t.TempDir(), "copied.txt")
	srcFile := filepath.Join(srcDir, "original.txt")

	require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o750))

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
			Overwrite:   true,
		},
	}

	p := New()

	result, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	data, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	require.Equal(t, "content", string(data))

	info, err := os.Stat(dstFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o750), info.Mode().Perm())
}

func TestCopyPlugin_ApplyRecursiveCopy(t *testing.T) {
	srcDir := t.TempDir()
	nested := filepath.Join(srcDir, "nested")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "file.txt"), []byte("recursive"), 0o644))

	destDir := filepath.Join(t.TempDir(), "dest")

	step := &config.Step{
		ID:   "copy_dir",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcDir,
			Destination: destDir,
			Recursive:   true,
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", res.Status)

	data, err := os.ReadFile(filepath.Join(destDir, "nested", "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "recursive", string(data))
}

func TestCopyPlugin_DryRunSkipsCopy(t *testing.T) {
	srcFile := filepath.Join(t.TempDir(), "source.txt")
	dstFile := filepath.Join(t.TempDir(), "dest.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o644))

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
		},
	}

	p := New()

	res, err := p.DryRun(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "skipped", res.Status)

	_, err = os.Stat(dstFile)
	require.Error(t, err)
}

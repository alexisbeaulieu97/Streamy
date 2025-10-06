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

func TestCopyPlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.Metadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "copy", meta.Type)
}

func TestCopyPlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.CopyStep)
	require.True(t, ok, "schema should be of type CopyStep")
}

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

func TestCopyPlugin_CheckReturnsFalseForDirectory(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	destDir := t.TempDir()

	step := &config.Step{
		ID:   "copy_dir",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcDir,
			Destination: destDir,
		},
	}

	p := New()

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok, "expected Check to return false for directory")
}

func TestCopyPlugin_CheckReturnsFalseWhenDestinationMissing(t *testing.T) {
	t.Parallel()

	srcFile := filepath.Join(t.TempDir(), "src.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("data"), 0o644))

	destFile := filepath.Join(t.TempDir(), "missing.txt")

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: destFile,
		},
	}

	p := New()

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok, "expected Check to return false when destination missing")
}

func TestCopyPlugin_ApplyWithOverwrite(t *testing.T) {
	srcFile := filepath.Join(t.TempDir(), "src.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("new content"), 0o644))

	destFile := filepath.Join(t.TempDir(), "dest.txt")
	require.NoError(t, os.WriteFile(destFile, []byte("old content"), 0o644))

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: destFile,
			Overwrite:   true,
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", res.Status)

	data, err := os.ReadFile(destFile)
	require.NoError(t, err)
	require.Equal(t, "new content", string(data))
}

func TestCopyPlugin_ApplyCreatesParentDirectory(t *testing.T) {
	srcFile := filepath.Join(t.TempDir(), "src.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("data"), 0o644))

	destFile := filepath.Join(t.TempDir(), "nested/dir/dest.txt")

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: destFile,
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", res.Status)

	data, err := os.ReadFile(destFile)
	require.NoError(t, err)
	require.Equal(t, "data", string(data))
}

func TestCopyPlugin_ApplyFailsForDirectoryWithoutRecursive(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("data"), 0o644))
	destDir := filepath.Join(t.TempDir(), "dest")

	step := &config.Step{
		ID:   "copy_dir",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcDir,
			Destination: destDir,
			Recursive:   false, // This should fail
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.Error(t, err)
	require.NotNil(t, res)
	require.Equal(t, "failed", res.Status)
	require.Contains(t, res.Message, "directory")
	require.Contains(t, res.Message, "recursive")
}

func TestCopyPlugin_ApplyWithoutPreserveMode(t *testing.T) {
	srcFile := filepath.Join(t.TempDir(), "src.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("data"), 0o700))

	destFile := filepath.Join(t.TempDir(), "dest.txt")

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:          srcFile,
			Destination:     destFile,
			PreserveMode:    false,
			PreserveModeSet: true,
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", res.Status)

	// File should have been copied with default permissions
	info, err := os.Stat(destFile)
	require.NoError(t, err)
	// Should not preserve the 0o700 mode
	require.NotEqual(t, os.FileMode(0o700), info.Mode().Perm())
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

func TestCopyPlugin_Verify(t *testing.T) {
	t.Run("returns satisfied when file exists and matches", func(t *testing.T) {
		srcFile := filepath.Join(t.TempDir(), "source.txt")
		dstFile := filepath.Join(t.TempDir(), "dest.txt")
		require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o644))
		require.NoError(t, os.WriteFile(dstFile, []byte("content"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: &config.CopyStep{
				Source:      srcFile,
				Destination: dstFile,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
		require.Contains(t, result.Message, "matches source")
	})

	t.Run("returns missing when destination does not exist", func(t *testing.T) {
		srcFile := filepath.Join(t.TempDir(), "source.txt")
		dstFile := filepath.Join(t.TempDir(), "dest.txt")
		require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: &config.CopyStep{
				Source:      srcFile,
				Destination: dstFile,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "missing", string(result.Status))
		require.Contains(t, result.Message, "does not exist")
	})

	t.Run("returns drifted when file content differs", func(t *testing.T) {
		srcFile := filepath.Join(t.TempDir(), "source.txt")
		dstFile := filepath.Join(t.TempDir(), "dest.txt")
		require.NoError(t, os.WriteFile(srcFile, []byte("new content"), 0o644))
		require.NoError(t, os.WriteFile(dstFile, []byte("old content"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: &config.CopyStep{
				Source:      srcFile,
				Destination: dstFile,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "drifted", string(result.Status))
		require.Contains(t, result.Message, "content differs")
	})

	t.Run("returns satisfied when directory exists", func(t *testing.T) {
		srcDir := filepath.Join(t.TempDir(), "source")
		dstDir := filepath.Join(t.TempDir(), "dest")
		require.NoError(t, os.MkdirAll(srcDir, 0o755))
		require.NoError(t, os.MkdirAll(dstDir, 0o755))

		p := New()

		step := &config.Step{
			ID:   "copy_dir",
			Type: "copy",
			Copy: &config.CopyStep{
				Source:      srcDir,
				Destination: dstDir,
				Recursive:   true,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
	})

	t.Run("returns drifted when source is dir but destination is file", func(t *testing.T) {
		srcDir := filepath.Join(t.TempDir(), "source")
		dstFile := filepath.Join(t.TempDir(), "dest")
		require.NoError(t, os.MkdirAll(srcDir, 0o755))
		require.NoError(t, os.WriteFile(dstFile, []byte("content"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "copy_dir",
			Type: "copy",
			Copy: &config.CopyStep{
				Source:      srcDir,
				Destination: dstFile,
				Recursive:   true,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "drifted", string(result.Status))
		require.Contains(t, result.Message, "types do not match")
	})

	t.Run("returns blocked when source does not exist", func(t *testing.T) {
		srcFile := filepath.Join(t.TempDir(), "nonexistent.txt")
		dstFile := filepath.Join(t.TempDir(), "dest.txt")

		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: &config.CopyStep{
				Source:      srcFile,
				Destination: dstFile,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "blocked", string(result.Status))
		require.Contains(t, result.Message, "source file")
	})

	t.Run("returns blocked when context is cancelled", func(t *testing.T) {
		srcFile := filepath.Join(t.TempDir(), "source.txt")
		dstFile := filepath.Join(t.TempDir(), "dest.txt")

		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: &config.CopyStep{
				Source:      srcFile,
				Destination: dstFile,
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

	t.Run("returns error when copy config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: nil,
		}

		_, err := p.Verify(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "copy configuration missing")
	})
}

func TestCopyPlugin_Check_Errors(t *testing.T) {
	t.Run("returns error when copy config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: nil,
		}

		_, err := p.Check(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "copy configuration missing")
	})
}

func TestCopyPlugin_Apply_Errors(t *testing.T) {
	t.Run("returns error when copy config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: nil,
		}

		_, err := p.Apply(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "copy configuration missing")
	})

	t.Run("returns error when source does not exist", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "copy_file",
			Type: "copy",
			Copy: &config.CopyStep{
				Source:      "/nonexistent/file.txt",
				Destination: filepath.Join(t.TempDir(), "dest.txt"),
			},
		}

		result, err := p.Apply(context.Background(), step)
		require.Error(t, err)
		require.Nil(t, result)
	})
}

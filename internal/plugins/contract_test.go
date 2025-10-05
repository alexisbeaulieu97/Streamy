package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	commandplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	copyplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	lineinfileplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"
	packageplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	repoplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	symlinkplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
	templateplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/template"
)

// getAllPlugins returns all plugin implementations for contract testing
func getAllPlugins() []plugin.Plugin {
	return []plugin.Plugin{
		symlinkplugin.New(),
		packageplugin.New(),
		templateplugin.New(),
		commandplugin.New(),
		repoplugin.New(),
		lineinfileplugin.New(),
		copyplugin.New(),
	}
}

// TestVerifyContract_ReadOnly tests BR-001: Verify() must not modify system state
func TestVerifyContract_ReadOnly(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.Metadata().Type, func(t *testing.T) {
			t.Parallel()

			// Create test fixtures
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			err := os.WriteFile(testFile, []byte("original content"), 0644)
			require.NoError(t, err)

			// Get file info before verification
			beforeStat, err := os.Stat(testFile)
			require.NoError(t, err)
			beforeContent, err := os.ReadFile(testFile)
			require.NoError(t, err)

			// Create appropriate step for plugin type
			step := createTestStep(p.Metadata().Type, tmpDir, testFile)

			// Run verification
			ctx := context.Background()
			result, err := p.Verify(ctx, step)

			// Verification should not return error
			require.NoError(t, err)
			require.NotNil(t, result)

			// Check that file hasn't been modified
			afterStat, err := os.Stat(testFile)
			require.NoError(t, err)
			afterContent, err := os.ReadFile(testFile)
			require.NoError(t, err)

			require.Equal(t, beforeContent, afterContent, "file content should not change")
			require.Equal(t, beforeStat.ModTime().Unix(), afterStat.ModTime().Unix(), "file modification time should not change")
			require.Equal(t, beforeStat.Mode(), afterStat.Mode(), "file permissions should not change")
		})
	}
}

// TestVerifyContract_ContextCancellation tests BR-002: Verify() must respect context cancellation
func TestVerifyContract_ContextCancellation(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.Metadata().Type, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			err := os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)

			step := createTestStep(p.Metadata().Type, tmpDir, testFile)

			// Create cancelled context
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			// Run verification
			result, err := p.Verify(ctx, step)

			// Should either return context.Canceled error or handle gracefully
			if err != nil {
				require.ErrorIs(t, err, context.Canceled, "should return context.Canceled error")
			} else {
				// If no error, result should indicate blocked status
				require.NotNil(t, result)
				if result.Status == model.StatusBlocked {
					require.NotNil(t, result.Error, "blocked status should have error")
				}
			}
		})
	}
}

// TestVerifyContract_TimeoutHandling tests BR-002: Verify() must respect context deadline
func TestVerifyContract_TimeoutHandling(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.Metadata().Type, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			err := os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)

			step := createTestStep(p.Metadata().Type, tmpDir, testFile)

			// Create context with very short deadline
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			defer cancel()

			// Give context time to expire
			time.Sleep(5 * time.Millisecond)

			// Run verification
			result, err := p.Verify(ctx, step)

			// Should either return deadline exceeded or handle gracefully
			if err != nil {
				require.ErrorIs(t, err, context.DeadlineExceeded, "should return context.DeadlineExceeded error")
			} else {
				require.NotNil(t, result)
				// Timeout should result in blocked status
				if result.Status == model.StatusBlocked {
					require.NotNil(t, result.Error, "blocked status should have error")
				}
			}
		})
	}
}

// TestVerifyContract_StatusAccuracy tests BR-003: Status must accurately reflect state
func TestVerifyContract_StatusAccuracy(t *testing.T) {
	t.Parallel()

	t.Run("symlink_satisfied", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		sourceFile := filepath.Join(tmpDir, "source.txt")
		targetLink := filepath.Join(tmpDir, "link")

		err := os.WriteFile(sourceFile, []byte("test"), 0644)
		require.NoError(t, err)
		err = os.Symlink(sourceFile, targetLink)
		require.NoError(t, err)

		p := symlinkplugin.New()
		step := &config.Step{
			ID:   "test",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: sourceFile,
				Target: targetLink,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, model.StatusSatisfied, result.Status, "symlink should be satisfied when it exists and matches")
	})

	t.Run("symlink_missing", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		sourceFile := filepath.Join(tmpDir, "source.txt")
		targetLink := filepath.Join(tmpDir, "nonexistent_link")

		err := os.WriteFile(sourceFile, []byte("test"), 0644)
		require.NoError(t, err)

		p := symlinkplugin.New()
		step := &config.Step{
			ID:   "test",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: sourceFile,
				Target: targetLink,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, model.StatusMissing, result.Status, "symlink should be missing when it doesn't exist")
	})

	t.Run("symlink_drifted", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		sourceFile := filepath.Join(tmpDir, "source.txt")
		wrongSource := filepath.Join(tmpDir, "wrong.txt")
		targetLink := filepath.Join(tmpDir, "link")

		err := os.WriteFile(sourceFile, []byte("test"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(wrongSource, []byte("wrong"), 0644)
		require.NoError(t, err)
		err = os.Symlink(wrongSource, targetLink)
		require.NoError(t, err)

		p := symlinkplugin.New()
		step := &config.Step{
			ID:   "test",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: sourceFile,
				Target: targetLink,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, model.StatusDrifted, result.Status, "symlink should be drifted when it points to wrong target")
	})

	t.Run("command_unknown", func(t *testing.T) {
		t.Parallel()

		p := commandplugin.New()
		step := &config.Step{
			ID:   "test",
			Type: "command",
			Command: &config.CommandStep{
				Command: "echo 'test'",
				// No Verify field specified
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, model.StatusUnknown, result.Status, "command should be unknown when no verify clause specified")
	})
}

// TestVerifyContract_MessageClarity tests BR-004: Message must be clear and descriptive
func TestVerifyContract_MessageClarity(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.Metadata().Type, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			err := os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)

			step := createTestStep(p.Metadata().Type, tmpDir, testFile)

			result, err := p.Verify(context.Background(), step)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Message must not be empty
			require.NotEmpty(t, result.Message, "message must not be empty")

			// Message should be at least 10 characters (meaningful description)
			require.GreaterOrEqual(t, len(result.Message), 10, "message should be at least 10 characters long")

			// Message should not be generic
			genericMessages := []string{"OK", "ok", "failed", "error", "success"}
			for _, generic := range genericMessages {
				require.NotEqual(t, generic, result.Message, "message should not be generic: %s", generic)
			}
		})
	}
}

// TestVerifyContract_Idempotency tests BR-008: Multiple calls should produce same result
func TestVerifyContract_Idempotency(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.Metadata().Type, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			err := os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)

			step := createTestStep(p.Metadata().Type, tmpDir, testFile)

			// First verification
			ctx := context.Background()
			result1, err := p.Verify(ctx, step)
			require.NoError(t, err)
			require.NotNil(t, result1)

			// Second verification (immediate)
			result2, err := p.Verify(ctx, step)
			require.NoError(t, err)
			require.NotNil(t, result2)

			// Results should be identical (assuming no state change)
			require.Equal(t, result1.Status, result2.Status, "status should be identical")
			require.Equal(t, result1.Message, result2.Message, "message should be identical")
			require.Equal(t, result1.Details, result2.Details, "details should be identical")
		})
	}
}

// createTestStep creates appropriate test step based on plugin type
func createTestStep(pluginType, tmpDir, testFile string) *config.Step {
	switch pluginType {
	case "symlink":
		sourceFile := filepath.Join(tmpDir, "source.txt")
		os.WriteFile(sourceFile, []byte("source"), 0644)
		return &config.Step{
			ID:   "test",
			Type: "symlink",
			Symlink: &config.SymlinkStep{
				Source: sourceFile,
				Target: testFile,
			},
		}

	case "copy":
		sourceFile := filepath.Join(tmpDir, "copy_source.txt")
		os.WriteFile(sourceFile, []byte("copy content"), 0644)
		return &config.Step{
			ID:   "test",
			Type: "copy",
			Copy: &config.CopyStep{
				Source: sourceFile,
				Destination:   testFile,
			},
		}

	case "template":
		templateFile := filepath.Join(tmpDir, "template.tmpl")
		os.WriteFile(templateFile, []byte("Hello {{.Name}}"), 0644)
		return &config.Step{
			ID:   "test",
			Type: "template",
			Template: &config.TemplateStep{
				Source:      templateFile,
				Destination: testFile,
				Vars: map[string]string{
					"Name": "World",
				},
			},
		}

	case "command":
		return &config.Step{
			ID:   "test",
			Type: "command",
			Command: &config.CommandStep{
				Command:    "echo test",
				Check: "test -f " + testFile,
			},
		}

	case "line_in_file":
		return &config.Step{
			ID:   "test",
			Type: "line_in_file",
			LineInFile: &config.LineInFileStep{
				File: testFile,
				Line: "test content",
			},
		}

	case "repo":
		repoPath := filepath.Join(tmpDir, "test_repo")
		os.MkdirAll(repoPath, 0755)
		return &config.Step{
			ID:   "test",
			Type: "repo",
			Repo: &config.RepoStep{
				URL:    "https://github.com/test/repo.git",
				Destination:   repoPath,
				Branch: "main",
			},
		}

	case "package":
		return &config.Step{
			ID:   "test",
			Type: "package",
			Package: &config.PackageStep{
				Packages: []string{"git"},
			},
		}

	default:
		return &config.Step{
			ID:   "test",
			Type: pluginType,
		}
	}
}

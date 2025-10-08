package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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

// TestEvaluateContract_ReadOnly tests BR-001: Evaluate() must not modify system state
func TestEvaluateContract_ReadOnly(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.PluginMetadata().Type, func(t *testing.T) {
			t.Parallel()

			// Create test fixtures
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			initialContent := "initial content"
			require.NoError(t, os.WriteFile(testFile, []byte(initialContent), 0644))

			// Create a step that would modify the file if applied
			step := createTestStep(t, p.PluginMetadata().Type, tmpDir, testFile)

			// Get initial state
			initialStat, err := os.Stat(testFile)
			require.NoError(t, err)

			// Run Evaluate multiple times
			for i := 0; i < 3; i++ {
				evalResult, err := p.Evaluate(context.Background(), step)
				require.NoError(t, err)
				require.NotNil(t, evalResult)

				// Verify file hasn't changed
				currentStat, err := os.Stat(testFile)
				require.NoError(t, err)
				require.Equal(t, initialStat.ModTime(), currentStat.ModTime(),
					"Evaluate() modified file system state on iteration %d", i)
				require.Equal(t, initialStat.Size(), currentStat.Size(),
					"Evaluate() changed file size on iteration %d", i)

				// Verify content hasn't changed
				content, err := os.ReadFile(testFile)
				require.NoError(t, err)
				require.Equal(t, initialContent, string(content),
					"Evaluate() changed file content on iteration %d", i)
			}
		})
	}
}

// TestEvaluateContract_Deterministic tests BR-002: Evaluate() must return consistent results
func TestEvaluateContract_Deterministic(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.PluginMetadata().Type, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

			step := createTestStep(t, p.PluginMetadata().Type, tmpDir, testFile)

			// Run Evaluate multiple times and compare results
			var firstResult *model.EvaluationResult
			for i := 0; i < 5; i++ {
				evalResult, err := p.Evaluate(context.Background(), step)
				require.NoError(t, err)
				require.NotNil(t, evalResult)

				if firstResult == nil {
					firstResult = evalResult
				} else {
					require.Equal(t, firstResult.StepID, evalResult.StepID)
					require.Equal(t, firstResult.CurrentState, evalResult.CurrentState)
					require.Equal(t, firstResult.RequiresAction, evalResult.RequiresAction)
					require.Equal(t, firstResult.Message, evalResult.Message)
				}
			}
		})
	}
}

// TestEvaluateContract_ResultsValid tests BR-003: Evaluate() must return valid EvaluationResult
func TestEvaluateContract_ResultsValid(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.PluginMetadata().Type, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

			step := createTestStep(t, p.PluginMetadata().Type, tmpDir, testFile)

			evalResult, err := p.Evaluate(context.Background(), step)
			require.NoError(t, err)
			require.NotNil(t, evalResult)

			// Validate EvaluationResult structure
			require.NotEmpty(t, evalResult.StepID)
			require.Contains(t, []model.VerificationStatus{
				model.StatusSatisfied,
				model.StatusMissing,
				model.StatusDrifted,
				model.StatusBlocked,
				model.StatusUnknown,
			}, evalResult.CurrentState)

			// RequiresAction should be true for Missing/Drifted, false for others
			expectedRequiresAction := evalResult.CurrentState == model.StatusMissing ||
				evalResult.CurrentState == model.StatusDrifted
			require.Equal(t, expectedRequiresAction, evalResult.RequiresAction)

			// Message should be non-empty
			require.NotEmpty(t, evalResult.Message)
		})
	}
}

// TestEvaluateContract_Timeout tests BR-004: Evaluate() must respect context cancellation
func TestEvaluateContract_Timeout(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.PluginMetadata().Type, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

			step := createTestStep(t, p.PluginMetadata().Type, tmpDir, testFile)

			// Test with cancelled context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			evalResult, err := p.Evaluate(ctx, step)
			require.Error(t, err)
			require.Nil(t, evalResult)
			require.ErrorIs(t, err, context.Canceled)

			// Test with timeout
			ctx, cancel = context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			evalResult, err = p.Evaluate(ctx, step)
			require.Error(t, err)
			require.Nil(t, evalResult)
			require.ErrorIs(t, err, context.DeadlineExceeded)
		})
	}
}

// TestApplyContract_UsesEvaluationResult tests that Apply() properly uses EvaluationResult
func TestApplyContract_UsesEvaluationResult(t *testing.T) {
	t.Parallel()

	plugins := getAllPlugins()

	for _, p := range plugins {
		p := p
		t.Run(p.PluginMetadata().Type, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			initialContent := "initial content"
			require.NoError(t, os.WriteFile(testFile, []byte(initialContent), 0644))

			step := createTestStep(t, p.PluginMetadata().Type, tmpDir, testFile)

			// First evaluate to get result
			evalResult, err := p.Evaluate(context.Background(), step)
			require.NoError(t, err)

			// Apply should work with valid evaluation result
			if evalResult.RequiresAction {
				result, err := p.Apply(context.Background(), evalResult, step)
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, step.ID, result.StepID)
			}
		})
	}
}

// createTestStep creates a test step for the given plugin type
func createTestStep(t *testing.T, pluginType, tmpDir, testFile string) *config.Step {
	switch pluginType {
	case "symlink":
		return &config.Step{
			ID:   "test-symlink",
			Type: pluginType,
			Symlink: &config.SymlinkStep{
				Source: testFile,
				Target: filepath.Join(tmpDir, "link"),
			},
		}
	case "copy":
		destFile := filepath.Join(tmpDir, "copy.txt")
		return &config.Step{
			ID:   "test-copy",
			Type: pluginType,
			Copy: &config.CopyStep{
				Source:      testFile,
				Destination: destFile,
			},
		}
	case "line_in_file":
		return &config.Step{
			ID:   "test-lineinfile",
			Type: pluginType,
			LineInFile: &config.LineInFileStep{
				File:  testFile,
				Line:  "new line",
				State: "present",
			},
		}
	case "template":
		templateFile := filepath.Join(tmpDir, "template.tmpl")
		require.NoError(t, os.WriteFile(templateFile, []byte("template content: {{ .Var }}"), 0644))
		outputFile := filepath.Join(tmpDir, "output.txt")
		mode := uint32(0644)
		return &config.Step{
			ID:   "test-template",
			Type: pluginType,
			Template: &config.TemplateStep{
				Source:      templateFile,
				Destination: outputFile,
				Vars:        map[string]string{"Var": "value"},
				Mode:        &mode,
			},
		}
	case "package":
		// Use a package that's guaranteed to be present so Apply is not invoked.
		return &config.Step{
			ID:   "test-package",
			Type: pluginType,
			Package: &config.PackageStep{
				Packages: []string{"bash"},
				Manager:  "apt",
			},
		}
	case "repo":
		source := initContractRepo(t)
		return &config.Step{
			ID:   "test-repo",
			Type: pluginType,
			Repo: &config.RepoStep{
				URL:         source,
				Destination: filepath.Join(tmpDir, "repo"),
			},
		}
	case "command":
		return &config.Step{
			ID:   "test-command",
			Type: pluginType,
			Command: &config.CommandStep{
				Command: "echo 'test command'",
				Check:   "echo 'check command'",
			},
		}
	default:
		t.Fatalf("unknown plugin type: %s", pluginType)
		return nil
	}
}

func initContractRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("contract repo"), 0o644))

	_, err = worktree.Add("README.md")
	require.NoError(t, err)

	_, err = worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Contract Test",
			Email: "contract@test",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	return dir
}

package lineinfileplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestLineInFile_Name(t *testing.T) {
	t.Parallel()

	plugin := New()
	meta := plugin.Metadata()

	assert.Equal(t, "line_in_file", meta.Type)
}

func TestLineInFile_Validate_Valid(t *testing.T) {
	t.Parallel()

	plugin := New()
	ctx := context.Background()

	tests := []struct {
		name string
		step *config.Step
	}{
		{
			name: "present without match",
			step: newLineInFileStep(
				"test-present",
				&config.LineInFileStep{
					File:  "/tmp/example.txt",
					Line:  "export PATH=\"$PATH:/opt/bin\"",
					State: "present",
				},
			),
		},
		{
			name: "present with match",
			step: newLineInFileStep(
				"test-match",
				&config.LineInFileStep{
					File:              "/tmp/example.txt",
					Line:              "debug=false",
					State:             "present",
					Match:             "^debug=",
					OnMultipleMatches: "first",
				},
			),
		},
		{
			name: "absent with match",
			step: newLineInFileStep(
				"test-absent",
				&config.LineInFileStep{
					File:  "/tmp/example.txt",
					Line:  "export OLD_VAR=value",
					State: "absent",
					Match: "^export OLD_VAR=",
				},
			),
		},
		{
			name: "all optional fields",
			step: newLineInFileStep(
				"test-optional",
				&config.LineInFileStep{
					File:              "/tmp/example.txt",
					Line:              "export EDITOR=vim",
					State:             "present",
					Match:             "^export EDITOR=",
					OnMultipleMatches: "all",
					Backup:            true,
					BackupDir:         "/tmp/backups",
					Encoding:          "latin-1",
				},
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := plugin.DryRun(ctx, tt.step)
			require.NoError(t, err)
		})
	}
}

func TestLineInFile_Validate_Errors(t *testing.T) {
	t.Parallel()

	plugin := New()
	ctx := context.Background()

	tests := []struct {
		name     string
		step     *config.Step
		errField string
	}{
		{
			name: "missing file",
			step: newLineInFileStep(
				"missing-file",
				&config.LineInFileStep{
					Line: "export PATH=\"$PATH:/opt/bin\"",
				},
			),
			errField: "file",
		},
		{
			name: "missing line",
			step: newLineInFileStep(
				"missing-line",
				&config.LineInFileStep{
					File: "/tmp/example.txt",
				},
			),
			errField: "line",
		},
		{
			name: "invalid state",
			step: newLineInFileStep(
				"invalid-state",
				&config.LineInFileStep{
					File:  "/tmp/example.txt",
					Line:  "value",
					State: "maybe",
				},
			),
			errField: "state",
		},
		{
			name: "absent without match",
			step: newLineInFileStep(
				"absent-match",
				&config.LineInFileStep{
					File:  "/tmp/example.txt",
					Line:  "value",
					State: "absent",
				},
			),
			errField: "match",
		},
		{
			name: "invalid regex",
			step: newLineInFileStep(
				"invalid-regex",
				&config.LineInFileStep{
					File:  "/tmp/example.txt",
					Line:  "value",
					State: "present",
					Match: "[invalid",
				},
			),
			errField: "match",
		},
		{
			name: "invalid multiple matches option",
			step: newLineInFileStep(
				"invalid-on-multiple",
				&config.LineInFileStep{
					File:              "/tmp/example.txt",
					Line:              "value",
					State:             "present",
					Match:             "^value$",
					OnMultipleMatches: "prompt-each",
				},
			),
			errField: "on_multiple_matches",
		},
		{
			name: "unsupported encoding",
			step: newLineInFileStep(
				"unsupported-encoding",
				&config.LineInFileStep{
					File:     "/tmp/example.txt",
					Line:     "value",
					State:    "present",
					Encoding: "utf-32",
				},
			),
			errField: "encoding",
		},
		{
			name: "empty file path",
			step: newLineInFileStep(
				"empty-file",
				&config.LineInFileStep{
					File: " ",
					Line: "value",
				},
			),
			errField: "file",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := plugin.DryRun(ctx, tt.step)
			require.Error(t, err)

			var validationErr *streamyerrors.ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tt.errField, validationErr.Field)
		})
	}
}

func newLineInFileStep(id string, cfg *config.LineInFileStep) *config.Step {
	return &config.Step{
		ID:         id,
		Type:       "line_in_file",
		LineInFile: cfg,
	}
}

func TestLineInFilePlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.LineInFileStep)
	require.True(t, ok, "schema should be of type LineInFileStep")
}

func TestLineInFilePlugin_Check(t *testing.T) {
	t.Run("returns true when line already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("existing line\nother content\n"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "ensure_line",
			Type: "line_in_file",
			LineInFile: &config.LineInFileStep{
				File:  testFile,
				Line:  "existing line",
				State: "present",
			},
		}

		ok, err := p.Check(context.Background(), step)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("returns false when line needs to be added", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("other content\n"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "ensure_line",
			Type: "line_in_file",
			LineInFile: &config.LineInFileStep{
				File:  testFile,
				Line:  "new line",
				State: "present",
			},
		}

		ok, err := p.Check(context.Background(), step)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("returns error when line_in_file config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:         "ensure_line",
			Type:       "line_in_file",
			LineInFile: nil,
		}

		_, err := p.Check(context.Background(), step)
		require.Error(t, err)
	})
}

func TestLineInFilePlugin_Verify(t *testing.T) {
	t.Run("returns satisfied when line is present as expected", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("existing line\n"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "ensure_line",
			Type: "line_in_file",
			LineInFile: &config.LineInFileStep{
				File:  testFile,
				Line:  "existing line",
				State: "present",
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
	})

	t.Run("returns drifted when line needs to be added", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("other content\n"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "ensure_line",
			Type: "line_in_file",
			LineInFile: &config.LineInFileStep{
				File:  testFile,
				Line:  "new line",
				State: "present",
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "drifted", string(result.Status))
	})

	t.Run("returns missing when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "nonexistent.txt")

		p := New()

		step := &config.Step{
			ID:   "ensure_line",
			Type: "line_in_file",
			LineInFile: &config.LineInFileStep{
				File:  testFile,
				Line:  "new line",
				State: "present",
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "missing", string(result.Status))
	})

	t.Run("returns satisfied when line correctly absent", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("other content\n"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "ensure_line_absent",
			Type: "line_in_file",
			LineInFile: &config.LineInFileStep{
				File:  testFile,
				Line:  "unwanted line",
				State: "absent",
				Match: "unwanted line",
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
	})

	t.Run("returns blocked when context is cancelled", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		p := New()

		step := &config.Step{
			ID:   "ensure_line",
			Type: "line_in_file",
			LineInFile: &config.LineInFileStep{
				File:  testFile,
				Line:  "new line",
				State: "present",
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

	t.Run("returns error when line_in_file config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:         "ensure_line",
			Type:       "line_in_file",
			LineInFile: nil,
		}

		_, err := p.Verify(context.Background(), step)
		require.Error(t, err)
	})
}

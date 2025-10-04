package lineinfileplugin

import (
	"context"
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

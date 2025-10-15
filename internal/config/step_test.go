package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type stepTestCase struct {
	name      string
	step      Step
	wantError error
}

func TestValidateStep(t *testing.T) {
	t.Parallel()

	cases := []stepTestCase{
		{
			name: "package step with packages",
			step: func() Step {
				var s Step
				s.ID = "install_git"
				s.Type = "package"
				require.NoError(t, s.SetConfig(PackageStep{Packages: []string{"git"}, Manager: "apt"}))
				return s
			}(),
		},
		{
			name: "package step missing packages",
			step: func() Step {
				var s Step
				s.ID = "install_none"
				s.Type = "package"
				require.NoError(t, s.SetConfig(PackageStep{}))
				return s
			}(),
			wantError: &streamyerrors.ValidationError{},
		},
		{
			name: "repo step with url and destination",
			step: func() Step {
				var s Step
				s.ID = "clone_repo"
				s.Type = "repo"
				require.NoError(t, s.SetConfig(RepoStep{URL: "https://example.com/repo.git", Destination: "/tmp/repo"}))
				return s
			}(),
		},
		{
			name: "repo step missing destination",
			step: func() Step {
				var s Step
				s.ID = "clone_repo"
				s.Type = "repo"
				require.NoError(t, s.SetConfig(RepoStep{URL: "https://example.com/repo.git"}))
				return s
			}(),
			wantError: &streamyerrors.ValidationError{},
		},
		{
			name: "symlink step valid",
			step: func() Step {
				var s Step
				s.ID = "link_file"
				s.Type = "symlink"
				require.NoError(t, s.SetConfig(SymlinkStep{Source: "/tmp/source", Target: "/tmp/target"}))
				return s
			}(),
		},
		{
			name: "copy step missing destination",
			step: func() Step {
				var s Step
				s.ID = "copy_file"
				s.Type = "copy"
				require.NoError(t, s.SetConfig(CopyStep{Source: "/tmp/src"}))
				return s
			}(),
			wantError: &streamyerrors.ValidationError{},
		},
		{
			name: "command step with command",
			step: func() Step {
				var s Step
				s.ID = "run_script"
				s.Type = "command"
				require.NoError(t, s.SetConfig(CommandStep{Command: "echo hello"}))
				return s
			}(),
		},
		{
			name: "command step missing command",
			step: func() Step {
				var s Step
				s.ID = "run_script"
				s.Type = "command"
				require.NoError(t, s.SetConfig(CommandStep{}))
				return s
			}(),
			wantError: &streamyerrors.ValidationError{},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateStep(tc.step)
			if tc.wantError == nil {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.IsType(t, tc.wantError, err)
		})
	}
}

func TestValidateStepWithNilPointer(t *testing.T) {
	t.Parallel()

	// Test package step with nil config
	step := Step{
		ID:   "test",
		Type: "package",
	}
	require.NoError(t, step.SetConfig(nil))
	err := ValidateStep(step)
	require.Error(t, err)
}

func TestValidateStep_TemplateAndLineInFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		step      Step
		wantError bool
	}{
		{
			name: "template step valid",
			step: func() Step {
				var s Step
				s.ID = "render_template"
				s.Type = "template"
				require.NoError(t, s.SetConfig(TemplateStep{Source: "/tmp/template.tmpl", Destination: "/tmp/output.txt"}))
				return s
			}(),
			wantError: false,
		},
		{
			name: "template step missing source",
			step: func() Step {
				var s Step
				s.ID = "render_template"
				s.Type = "template"
				require.NoError(t, s.SetConfig(TemplateStep{Destination: "/tmp/output.txt"}))
				return s
			}(),
			wantError: true,
		},
		{
			name: "template step nil config",
			step: func() Step {
				var s Step
				s.ID = "render_template"
				s.Type = "template"
				require.NoError(t, s.SetConfig(nil))
				return s
			}(),
			wantError: true,
		},
		{
			name: "line_in_file step valid",
			step: func() Step {
				var s Step
				s.ID = "ensure_line"
				s.Type = "line_in_file"
				require.NoError(t, s.SetConfig(LineInFileStep{File: "/tmp/file.txt", Line: "some line", State: "present"}))
				return s
			}(),
			wantError: false,
		},
		{
			name: "line_in_file step missing file",
			step: func() Step {
				var s Step
				s.ID = "ensure_line"
				s.Type = "line_in_file"
				require.NoError(t, s.SetConfig(LineInFileStep{Line: "some line", State: "present"}))
				return s
			}(),
			wantError: true,
		},
		{
			name: "line_in_file step nil config",
			step: func() Step {
				var s Step
				s.ID = "ensure_line"
				s.Type = "line_in_file"
				require.NoError(t, s.SetConfig(nil))
				return s
			}(),
			wantError: true,
		},
		{
			name: "unknown step type",
			step: Step{
				ID:   "test",
				Type: "unknown_type",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateStep(tt.step)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

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
			step: Step{
				ID:   "install_git",
				Type: "package",
				Package: &PackageStep{
					Packages: []string{"git"},
					Manager:  "apt",
				},
			},
		},
		{
			name: "package step missing packages",
			step: Step{
				ID:      "install_none",
				Type:    "package",
				Package: &PackageStep{},
			},
			wantError: &streamyerrors.ValidationError{},
		},
		{
			name: "repo step with url and destination",
			step: Step{
				ID:   "clone_repo",
				Type: "repo",
				Repo: &RepoStep{
					URL:         "https://example.com/repo.git",
					Destination: "/tmp/repo",
				},
			},
		},
		{
			name: "repo step missing destination",
			step: Step{
				ID:   "clone_repo",
				Type: "repo",
				Repo: &RepoStep{
					URL: "https://example.com/repo.git",
				},
			},
			wantError: &streamyerrors.ValidationError{},
		},
		{
			name: "symlink step valid",
			step: Step{
				ID:   "link_file",
				Type: "symlink",
				Symlink: &SymlinkStep{
					Source: "/tmp/source",
					Target: "/tmp/target",
				},
			},
		},
		{
			name: "copy step missing destination",
			step: Step{
				ID:   "copy_file",
				Type: "copy",
				Copy: &CopyStep{
					Source: "/tmp/src",
				},
			},
			wantError: &streamyerrors.ValidationError{},
		},
		{
			name: "command step with command",
			step: Step{
				ID:   "run_script",
				Type: "command",
				Command: &CommandStep{
					Command: "echo hello",
				},
			},
		},
		{
			name: "command step missing command",
			step: Step{
				ID:      "run_script",
				Type:    "command",
				Command: &CommandStep{},
			},
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
		ID:      "test",
		Type:    "package",
		Package: nil,
	}
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
			step: Step{
				ID:   "render_template",
				Type: "template",
				Template: &TemplateStep{
					Source:      "/tmp/template.tmpl",
					Destination: "/tmp/output.txt",
				},
			},
			wantError: false,
		},
		{
			name: "template step missing source",
			step: Step{
				ID:   "render_template",
				Type: "template",
				Template: &TemplateStep{
					Destination: "/tmp/output.txt",
				},
			},
			wantError: true,
		},
		{
			name: "template step nil config",
			step: Step{
				ID:       "render_template",
				Type:     "template",
				Template: nil,
			},
			wantError: true,
		},
		{
			name: "line_in_file step valid",
			step: Step{
				ID:   "ensure_line",
				Type: "line_in_file",
				LineInFile: &LineInFileStep{
					File:  "/tmp/file.txt",
					Line:  "some line",
					State: "present",
				},
			},
			wantError: false,
		},
		{
			name: "line_in_file step missing file",
			step: Step{
				ID:   "ensure_line",
				Type: "line_in_file",
				LineInFile: &LineInFileStep{
					Line:  "some line",
					State: "present",
				},
			},
			wantError: true,
		},
		{
			name: "line_in_file step nil config",
			step: Step{
				ID:         "ensure_line",
				Type:       "line_in_file",
				LineInFile: nil,
			},
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

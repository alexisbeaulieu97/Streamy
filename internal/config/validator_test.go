package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	validConfig := &Config{
		Version: "1.0.0",
		Name:    "Valid",
		Steps: []Step{
			{
				ID:   "install_git",
				Type: "package",
				Package: &PackageStep{
					Packages: []string{"git"},
				},
			},
			{
				ID:        "clone_repo",
				Type:      "repo",
				DependsOn: []string{"install_git"},
				Repo: &RepoStep{
					URL:         "https://example.com/repo.git",
					Destination: "/tmp/repo",
				},
			},
		},
	}

	duplicateIDs := &Config{
		Version: "1.0",
		Name:    "Duplicate IDs",
		Steps: []Step{
			{ID: "dup", Type: "command", Command: &CommandStep{Command: "echo"}},
			{ID: "dup", Type: "command", Command: &CommandStep{Command: "echo"}},
		},
	}

	missingDependency := &Config{
		Version: "1.0",
		Name:    "Missing Dependency",
		Steps: []Step{
			{ID: "first", Type: "command", Command: &CommandStep{Command: "echo"}},
			{ID: "second", Type: "command", DependsOn: []string{"missing"}, Command: &CommandStep{Command: "echo"}},
		},
	}

	cycleConfig := &Config{
		Version: "1.0",
		Name:    "Cycle",
		Steps: []Step{
			{ID: "a", Type: "command", Enabled: true, DependsOn: []string{"c"}, Command: &CommandStep{Command: "echo a"}},
			{ID: "b", Type: "command", Enabled: true, DependsOn: []string{"a"}, Command: &CommandStep{Command: "echo b"}},
			{ID: "c", Type: "command", Enabled: true, DependsOn: []string{"b"}, Command: &CommandStep{Command: "echo c"}},
		},
	}

	disabledCycle := &Config{
		Version: "1.0",
		Name:    "Disabled Cycle",
		Steps: []Step{
			{ID: "x", Type: "command", Enabled: false, DependsOn: []string{"y"}, Command: &CommandStep{Command: "echo x"}},
			{ID: "y", Type: "command", Enabled: false, DependsOn: []string{"x"}, Command: &CommandStep{Command: "echo y"}},
		},
	}

	cases := []struct {
		name      string
		cfg       *Config
		wantError error
	}{
		{name: "valid config passes", cfg: validConfig, wantError: nil},
		{name: "duplicate ids", cfg: duplicateIDs, wantError: &streamyerrors.ValidationError{}},
		{name: "missing dependency", cfg: missingDependency, wantError: &streamyerrors.ValidationError{}},
		{name: "circular dependency", cfg: cycleConfig, wantError: &streamyerrors.ValidationError{}},
		{name: "disabled cycle ignored", cfg: disabledCycle, wantError: nil},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateConfig(tc.cfg)
			switch tc.wantError {
			case nil:
				require.NoError(t, err)
			default:
				require.Error(t, err)
				require.IsType(t, tc.wantError, err)
			}
		})
	}
}

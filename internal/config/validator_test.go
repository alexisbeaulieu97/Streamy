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
		Steps: func() []Step {
			pkgStep := Step{ID: "install_git", Type: "package"}
			require.NoError(t, pkgStep.SetConfig(PackageStep{Packages: []string{"git"}}))

			repoStep := Step{ID: "clone_repo", Type: "repo", DependsOn: []string{"install_git"}}
			require.NoError(t, repoStep.SetConfig(RepoStep{URL: "https://example.com/repo.git", Destination: "/tmp/repo"}))

			return []Step{pkgStep, repoStep}
		}(),
	}

	duplicateIDs := &Config{
		Version: "1.0",
		Name:    "Duplicate IDs",
		Steps: func() []Step {
			stepA := Step{ID: "dup", Type: "command"}
			require.NoError(t, stepA.SetConfig(CommandStep{Command: "echo"}))
			stepB := Step{ID: "dup", Type: "command"}
			require.NoError(t, stepB.SetConfig(CommandStep{Command: "echo"}))
			return []Step{stepA, stepB}
		}(),
	}

	missingDependency := &Config{
		Version: "1.0",
		Name:    "Missing Dependency",
		Steps: func() []Step {
			first := Step{ID: "first", Type: "command"}
			require.NoError(t, first.SetConfig(CommandStep{Command: "echo"}))
			second := Step{ID: "second", Type: "command", DependsOn: []string{"missing"}}
			require.NoError(t, second.SetConfig(CommandStep{Command: "echo"}))
			return []Step{first, second}
		}(),
	}

	cycleConfig := &Config{
		Version: "1.0",
		Name:    "Cycle",
		Steps: func() []Step {
			a := Step{ID: "a", Type: "command", Enabled: true, DependsOn: []string{"c"}}
			require.NoError(t, a.SetConfig(CommandStep{Command: "echo a"}))
			b := Step{ID: "b", Type: "command", Enabled: true, DependsOn: []string{"a"}}
			require.NoError(t, b.SetConfig(CommandStep{Command: "echo b"}))
			c := Step{ID: "c", Type: "command", Enabled: true, DependsOn: []string{"b"}}
			require.NoError(t, c.SetConfig(CommandStep{Command: "echo c"}))
			return []Step{a, b, c}
		}(),
	}

	disabledCycle := &Config{
		Version: "1.0",
		Name:    "Disabled Cycle",
		Steps: func() []Step {
			x := Step{ID: "x", Type: "command", Enabled: false, DependsOn: []string{"y"}}
			require.NoError(t, x.SetConfig(CommandStep{Command: "echo x"}))
			y := Step{ID: "y", Type: "command", Enabled: false, DependsOn: []string{"x"}}
			require.NoError(t, y.SetConfig(CommandStep{Command: "echo y"}))
			return []Step{x, y}
		}(),
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

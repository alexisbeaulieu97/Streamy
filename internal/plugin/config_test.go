package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfigCIEnforcesStrictPolicies(t *testing.T) {
	envVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_HOME"}

	for _, env := range envVars {
		t.Run(env, func(t *testing.T) {
			for _, reset := range envVars {
				t.Setenv(reset, "")
			}
			t.Setenv(env, "1")

			cfg := DefaultConfig()

			require.Equal(t, PolicyStrict, cfg.DependencyPolicy)
			require.Equal(t, AccessStrict, cfg.AccessPolicy)
		})
	}
}

func TestDefaultConfigInteractiveDefaults(t *testing.T) {
	// Test interactive defaults by ensuring we're not in CI environment
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")

	cfg := DefaultConfig()

	require.Equal(t, PolicyGraceful, cfg.DependencyPolicy)
	require.Equal(t, AccessWarn, cfg.AccessPolicy)
}

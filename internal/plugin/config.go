package plugin

import (
	"os"
	"strings"
)

// DependencyPolicy controls how the registry responds to validation failures.
type DependencyPolicy string

const (
	// PolicyStrict fails fast when dependency validation fails.
	PolicyStrict DependencyPolicy = "strict"
	// PolicyGraceful attempts to continue by disabling affected plugins.
	PolicyGraceful DependencyPolicy = "graceful"
)

// AccessPolicy controls how undeclared dependency access is handled.
type AccessPolicy string

const (
	// AccessStrict returns errors for undeclared dependency usage.
	AccessStrict AccessPolicy = "strict"
	// AccessWarn logs warnings for undeclared dependency usage.
	AccessWarn AccessPolicy = "warn"
	// AccessOff disables undeclared dependency enforcement.
	AccessOff AccessPolicy = "off"
)

// RegistryConfig configures registry validation and dependency access policies.
type RegistryConfig struct {
	DependencyPolicy DependencyPolicy
	AccessPolicy     AccessPolicy
}

// DefaultConfig returns environment-aware defaults for the registry configuration.
func DefaultConfig() *RegistryConfig {
	if isCIEnvironment() {
		return &RegistryConfig{
			DependencyPolicy: PolicyStrict,
			AccessPolicy:     AccessStrict,
		}
	}

	return &RegistryConfig{
		DependencyPolicy: PolicyGraceful,
		AccessPolicy:     AccessWarn,
	}
}

func isCIEnvironment() bool {
	ciEnvVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_HOME",
	}

	for _, key := range ciEnvVars {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" && strings.ToLower(value) != "false" && value != "0" {
			return true
		}
	}

	return false
}

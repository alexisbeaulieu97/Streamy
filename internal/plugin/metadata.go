package plugin

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	apiverPattern = regexp.MustCompile(`^\d+\.x$`)
)

// PluginMetadata describes plugin identity and dependency requirements.
type PluginMetadata struct {
	Name         string
	Version      string
	APIVersion   string
	Dependencies []Dependency
	Stateful     bool
	Description  string
}

// Dependency captures a dependency on another plugin.
type Dependency struct {
	Name              string
	VersionConstraint *VersionConstraint
}

// Validate ensures metadata is well-formed.
func (m PluginMetadata) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("plugin metadata requires a non-empty Name")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("plugin '%s' metadata requires Version", m.Name)
	}
	if !semverPattern.MatchString(m.Version) {
		return fmt.Errorf("plugin '%s' has invalid Version '%s' (expected format: X.Y.Z)", m.Name, m.Version)
	}
	if strings.TrimSpace(m.APIVersion) == "" {
		return fmt.Errorf("plugin '%s' metadata requires APIVersion", m.Name)
	}
	if !apiverPattern.MatchString(m.APIVersion) {
		return fmt.Errorf("plugin '%s' has invalid APIVersion '%s' (expected format: N.x)", m.Name, m.APIVersion)
	}

	seenDeps := map[string]struct{}{}
	for _, dep := range m.Dependencies {
		if err := dep.Validate(m.Name); err != nil {
			return err
		}
		if dep.Name == m.Name {
			return fmt.Errorf("plugin '%s' cannot depend on itself", m.Name)
		}
		if _, exists := seenDeps[dep.Name]; exists {
			return fmt.Errorf("plugin '%s' lists dependency '%s' more than once", m.Name, dep.Name)
		}
		seenDeps[dep.Name] = struct{}{}
	}

	return nil
}

// Validate ensures the dependency entry is well-formed.
func (d Dependency) Validate(owner string) error {
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("plugin '%s' declares dependency with empty name", owner)
	}
	if d.VersionConstraint == nil {
		return nil
	}
	if d.VersionConstraint.MajorVersion < 0 {
		return fmt.Errorf("plugin '%s' declares dependency '%s' with invalid version constraint (negative major version)", owner, d.Name)
	}
	return nil
}

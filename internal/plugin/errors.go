package plugin

import (
	"fmt"
	"sort"
	"strings"
)

// ErrPluginNotFound is returned when the requested plugin is not registered.
type ErrPluginNotFound struct {
	Name string
}

func (e ErrPluginNotFound) Error() string {
	return fmt.Sprintf("plugin '%s' not found in registry\nHint: ensure the plugin is registered before usage", e.Name)
}

// ErrCircularDependency is returned when a dependency cycle is detected.
type ErrCircularDependency struct {
	Cycle []string
}

func (e ErrCircularDependency) Error() string {
	if len(e.Cycle) == 0 {
		return "circular dependency detected\nHint: review plugin dependencies to remove cycles"
	}

	sequence := append(append([]string{}, e.Cycle...), e.Cycle[0])
	return fmt.Sprintf(
		"circular dependency detected: %s\nHint: break the cycle by removing or refactoring one of the dependencies",
		strings.Join(sequence, " -> "),
	)
}

// ErrVersionConflict captures version mismatches between dependents and a plugin.
type ErrVersionConflict struct {
	Plugin        string
	RequiredBy    map[string]string // dependent -> version constraint
	ActualVersion string
}

func (e ErrVersionConflict) Error() string {
	conflicts := make([]string, 0, len(e.RequiredBy))
	for dependent, constraint := range e.RequiredBy {
		conflicts = append(conflicts, fmt.Sprintf("%s requires %s", dependent, constraint))
	}
	sort.Strings(conflicts)

	return fmt.Sprintf(
		"version conflict for plugin '%s' (actual %s):\n  %s\nHint: align plugin versions or relax constraints",
		e.Plugin,
		e.ActualVersion,
		strings.Join(conflicts, "\n  "),
	)
}

// ErrUndeclaredDependency is returned when a plugin accesses a dependency it did not declare.
type ErrUndeclaredDependency struct {
	Caller     string
	Dependency string
}

func (e ErrUndeclaredDependency) Error() string {
	return fmt.Sprintf(
		"plugin '%s' attempted to access undeclared dependency '%s'\nHint: add '%s' to PluginMetadata.Dependencies",
		e.Caller,
		e.Dependency,
		e.Dependency,
	)
}

// ErrMissingDependency is returned when a declared dependency has not been registered.
type ErrMissingDependency struct {
	Plugin     string
	Dependency string
}

func (e ErrMissingDependency) Error() string {
	return fmt.Sprintf(
		"plugin '%s' declares dependency '%s' which is not registered\nHint: register the dependency before validating or initializing plugins",
		e.Plugin,
		e.Dependency,
	)
}

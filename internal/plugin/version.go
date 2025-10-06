package plugin

import (
	"fmt"
	"strconv"
	"strings"
)

// VersionConstraint restricts acceptable plugin versions to a single major version.
type VersionConstraint struct {
	MajorVersion int
}

// ParseVersionConstraint parses a string in the form "N.x" into a VersionConstraint.
func ParseVersionConstraint(s string) (*VersionConstraint, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil, fmt.Errorf("version constraint string is empty")
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) != 2 || parts[1] != "x" {
		return nil, fmt.Errorf("invalid version constraint '%s' (expected format: N.x)", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version in constraint '%s'", s)
	}
	if major < 0 {
		return nil, fmt.Errorf("major version must be non-negative in constraint '%s'", s)
	}

	return &VersionConstraint{MajorVersion: major}, nil
}

// MustParseVersionConstraint panics if the constraint cannot be parsed.
func MustParseVersionConstraint(s string) *VersionConstraint {
	vc, err := ParseVersionConstraint(s)
	if err != nil {
		panic(err)
	}
	return vc
}

// Satisfies determines whether the provided semantic version satisfies the constraint.
func (vc *VersionConstraint) Satisfies(version string) bool {
	if vc == nil {
		return true
	}
	major, ok := parseMajor(version)
	if !ok {
		return false
	}
	return major == vc.MajorVersion
}

func parseMajor(version string) (int, bool) {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return 0, false
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) < 1 {
		return 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false
	}
	return major, true
}

// String returns the canonical representation of the constraint.
func (vc *VersionConstraint) String() string {
	if vc == nil {
		return ""
	}
	return fmt.Sprintf("%d.x", vc.MajorVersion)
}

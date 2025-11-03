package registry

import (
	"fmt"
	"strings"
)

const canonicalIDSeparator = "@"

// BuildCanonicalPipelineID returns the canonical registry identifier in the
// form <id>@<version>. Both components are validated prior to joining.
func BuildCanonicalPipelineID(id, version string) (string, error) {
	baseID := strings.TrimSpace(id)
	baseVersion := strings.TrimSpace(version)

	if err := ValidatePipelineID(baseID); err != nil {
		return "", err
	}

	if err := validatePipelineVersion(baseVersion); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%s%s", baseID, canonicalIDSeparator, baseVersion), nil
}

// ParseCanonicalPipelineID splits a canonical identifier into its component ID
// and version parts, validating the format along the way.
func ParseCanonicalPipelineID(value string) (string, string, error) {
	canonical := strings.TrimSpace(value)
	if canonical == "" {
		return "", "", fmt.Errorf("canonical pipeline identifier cannot be empty")
	}

	parts := strings.Split(canonical, canonicalIDSeparator)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid canonical pipeline identifier %q: must be in <id>@<version> format", value)
	}

	rawID := parts[0]
	rawVersion := parts[1]

	if rawID != strings.TrimSpace(rawID) || rawVersion != strings.TrimSpace(rawVersion) {
		return "", "", fmt.Errorf("invalid canonical pipeline identifier %q: whitespace around %q is not allowed", value, canonicalIDSeparator)
	}

	id := rawID
	version := rawVersion

	if err := ValidatePipelineID(id); err != nil {
		return "", "", fmt.Errorf("invalid canonical pipeline identifier %q: %w", value, err)
	}

	if err := validatePipelineVersion(version); err != nil {
		return "", "", fmt.Errorf("invalid canonical pipeline identifier %q: %w", value, err)
	}

	return id, version, nil
}

// ValidateCanonicalPipelineID ensures the supplied identifier uses the
// canonical <id>@<version> format.
func ValidateCanonicalPipelineID(value string) error {
	_, _, err := ParseCanonicalPipelineID(value)
	return err
}

// IsCanonicalPipelineID reports whether the provided identifier is already in
// canonical <id>@<version> form.
func IsCanonicalPipelineID(value string) bool {
	return ValidateCanonicalPipelineID(value) == nil
}

// validatePipelineVersion guarantees that versions are well-formed for use in
// canonical identifiers.
func validatePipelineVersion(version string) error {
	if version == "" {
		return fmt.Errorf("pipeline version cannot be empty")
	}

	if strings.Contains(version, canonicalIDSeparator) {
		return fmt.Errorf("pipeline version %q cannot contain %q", version, canonicalIDSeparator)
	}

	if strings.ContainsAny(version, " \t\r\n") {
		return fmt.Errorf("pipeline version %q cannot contain whitespace", version)
	}

	return nil
}

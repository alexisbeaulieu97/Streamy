package registry

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	pipelineIDMaxLength    = 64
	randomIDSuffixLength   = 8
	randomIDSuffixFallback = "abcdefgh"
)

var (
	pipelineIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
	nonAlphanumericExpr = regexp.MustCompile(`[^a-z0-9]+`)
)

// GeneratePipelineID converts a configuration path into a sanitized pipeline ID.
func GeneratePipelineID(path string) string {
	base := filepath.Base(path)
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}

	id := SanitizeFilename(base)
	if id == "" {
		id = fmt.Sprintf("pipeline-%s", randomIDSuffix(randomIDSuffixLength))
	}

	if len(id) > pipelineIDMaxLength {
		id = trimToLength(id, pipelineIDMaxLength)
	}

	if id == "" {
		id = fmt.Sprintf("pipeline-%s", randomIDSuffix(randomIDSuffixLength))
	}

	return id
}

// ValidatePipelineID ensures the provided ID matches the allowed pattern.
func ValidatePipelineID(id string) error {
	if id == "" {
		return fmt.Errorf("pipeline ID cannot be empty")
	}

	if len(id) > pipelineIDMaxLength {
		return fmt.Errorf("pipeline ID %q is too long: maximum length is %d characters", id, pipelineIDMaxLength)
	}

	if !pipelineIDPattern.MatchString(id) {
		return fmt.Errorf("invalid pipeline ID %q: must match %s", id, pipelineIDPattern.String())
	}

	return nil
}

// SanitizeFilename normalizes a filename into an identifier-friendly format.
func SanitizeFilename(name string) string {
	lowered := strings.ToLower(name)
	sanitized := nonAlphanumericExpr.ReplaceAllString(lowered, "-")
	sanitized = strings.Trim(sanitized, "-")

	if len(sanitized) > pipelineIDMaxLength {
		sanitized = trimToLength(sanitized, pipelineIDMaxLength)
	}

	return sanitized
}

func randomIDSuffix(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

	if length <= 0 {
		return ""
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return randomIDSuffixFallback
	}

	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}

	return string(buf)
}

func trimToLength(value string, length int) string {
	if len(value) <= length {
		return strings.Trim(value, "-")
	}

	trimmed := value[:length]
	return strings.Trim(trimmed, "-")
}

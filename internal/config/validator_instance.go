package config

import (
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validatorOnce sync.Once
	validateInst  *validator.Validate

	semverPattern   = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z-.]+)?(?:\+[0-9A-Za-z-.]+)?$`)
	stepIDPattern   = regexp.MustCompile(`^[a-z0-9_-]+$`)
	sshGitPattern   = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+:[a-zA-Z0-9._/~-]+$`)
	validationTypes = map[string]struct{}{"command_exists": {}, "file_exists": {}, "path_contains": {}}
)

// validatorInstance configures and returns the shared validator instance used across the config package.
func validatorInstance() *validator.Validate {
	validatorOnce.Do(func() {
		v := validator.New()

		_ = v.RegisterValidation("semver", func(fl validator.FieldLevel) bool {
			return semverPattern.MatchString(fl.Field().String())
		})

		_ = v.RegisterValidation("step_id", func(fl validator.FieldLevel) bool {
			return stepIDPattern.MatchString(fl.Field().String())
		})

		_ = v.RegisterValidation("git_url", func(fl validator.FieldLevel) bool {
			urlStr := fl.Field().String()
			if urlStr == "" {
				return true // Allow empty if not required
			}

			// Reject whitespace-only strings
			if strings.TrimSpace(urlStr) == "" {
				return false
			}

			// Check for network URLs (http/https only with non-empty host)
			if parsedURL, err := url.Parse(urlStr); err == nil {
				scheme := strings.ToLower(parsedURL.Scheme)
				if scheme == "http" || scheme == "https" {
					// Ensure host is not empty
					if parsedURL.Host != "" {
						return true
					}
				}
			}

			// Check for SSH-style git URLs (user@host:path)
			if sshGitPattern.MatchString(urlStr) {
				return true
			}

			// Check for syntactically-valid file paths without filesystem access
			if isValidFilePath(urlStr) {
				return true
			}

			return false
		})

		validateInst = v
	})

	return validateInst
}

// GetValidator returns a configured validator instance for use outside the config package.
func GetValidator() *validator.Validate {
	return validatorInstance()
}

// isValidFilePath performs syntactic validation of file paths without filesystem access
func isValidFilePath(path string) bool {
	if path == "" {
		return false
	}

	// Check for NUL characters
	if strings.Contains(path, "\x00") {
		return false
	}

	// Check for absolute paths
	if strings.HasPrefix(path, "/") {
		// Additional safety check: reject suspicious absolute paths
		return !strings.Contains(path, "/../") && !strings.HasSuffix(path, "/..")
	}

	// Check for relative paths with explicit prefixes only
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		// Basic safety checks for relative paths
		return !strings.Contains(path, "\x00")
	}

	// Reject all other path formats
	return false
}

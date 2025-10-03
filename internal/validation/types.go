package validation

import "github.com/alexisbeaulieu97/streamy/internal/config"

// ValidationResult captures the outcome of executing a single validation rule.
type ValidationResult struct {
	Validation config.Validation
	Passed     bool
	Message    string
	Error      error
}

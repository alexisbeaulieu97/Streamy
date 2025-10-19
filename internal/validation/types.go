package validation

import domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"

// ValidationResult captures the outcome of executing a single validation rule.
type ValidationResult struct {
	Validation domain.Validation
	Passed     bool
	Message    string
	Error      error
}

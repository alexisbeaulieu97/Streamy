package validation

import domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"

// Result captures the outcome of executing a single validation rule.
type Result struct {
	Validation domain.Validation
	Passed     bool
	Message    string
	Error      error
}

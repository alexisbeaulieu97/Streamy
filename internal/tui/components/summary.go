package components

import (
	"fmt"
	"strings"
)

// ValidationStatus represents a validation outcome for summary rendering.
type ValidationStatus struct {
	Passed  bool
	Message string
}

// SummaryData aggregates counts for rendering summaries.
type SummaryData struct {
	Total       int
	Completed   int
	Finished    bool
	Cancelled   bool
	Validations []ValidationStatus
}

// Summary renders a textual execution summary.
type Summary struct {
	data SummaryData
}

// NewSummary creates a new Summary component.
func NewSummary(data SummaryData) Summary {
	return Summary{data: data}
}

// View renders the summary.
func (s Summary) View() string {
	var lines []string
	if s.data.Total > 0 {
		lines = append(lines, fmt.Sprintf("Steps: %d/%d completed", s.data.Completed, s.data.Total))
	}

	if s.data.Cancelled {
		lines = append(lines, "Execution cancelled")
	} else if s.data.Finished && s.data.Total > 0 {
		if s.data.Completed == s.data.Total {
			lines = append(lines, "Execution finished successfully")
		} else {
			lines = append(lines, "Execution finished with pending steps")
		}
	}

	if len(s.data.Validations) > 0 {
		lines = append(lines, "Validations:")
		for _, v := range s.data.Validations {
			status := "✗"
			if v.Passed {
				status = "✓"
			}
			lines = append(lines, fmt.Sprintf("  %s %s", status, v.Message))
		}
	}

	return strings.Join(lines, "\n")
}

package pipeline

// ResultStatus represents the status of an executed step.
type ResultStatus string

const (
	// StatusSuccess indicates the step completed successfully.
	StatusSuccess ResultStatus = "success"
	// StatusFailure indicates the step failed.
	StatusFailure ResultStatus = "failure"
	// StatusSkipped indicates the step was skipped.
	StatusSkipped ResultStatus = "skipped"
	// StatusAlreadySatisfied indicates the desired state already matched reality.
	StatusAlreadySatisfied ResultStatus = "already_satisfied"
)

// StepResult captures the outcome of a step execution.
type StepResult struct {
	StepID   string
	Status   ResultStatus
	Duration int
	Message  string
	Output   string
	Error    *DomainError
	Changed  bool
	Diff     string
}

// IsSuccess returns true when the step completed successfully.
func (r StepResult) IsSuccess() bool {
	return r.Status == StatusSuccess || r.Status == StatusAlreadySatisfied
}

// IsFailure returns true when the step failed.
func (r StepResult) IsFailure() bool {
	return r.Status == StatusFailure
}

// FormatOutput returns a human-readable summary of the result.
func (r StepResult) FormatOutput() string {
	if r.Error != nil {
		return r.Error.Error()
	}

	if r.Output != "" {
		return r.Output
	}

	return r.Message
}

// VerificationStatus represents the outcome of a validation check.
type VerificationStatus string

const (
	// VerificationSatisfied indicates the validation passed.
	VerificationSatisfied VerificationStatus = "satisfied"
	// VerificationFailed indicates the validation failed.
	VerificationFailed VerificationStatus = "failed"
	// VerificationUnknown indicates the validation outcome is undetermined.
	VerificationUnknown VerificationStatus = "unknown"
)

// VerificationResult captures the outcome of a validation step.
type VerificationResult struct {
	StepID  string
	Type    string
	Status  VerificationStatus
	Message string
	Details map[string]interface{}
}

// IsSatisfied returns true if the validation passed.
func (v VerificationResult) IsSatisfied() bool {
	return v.Status == VerificationSatisfied
}

// FormatMessage returns a message describing the validation result.
func (v VerificationResult) FormatMessage() string {
	if v.Message != "" {
		return v.Message
	}

	return string(v.Status)
}

// VerificationSummary aggregates validation results and counts.
type VerificationSummary struct {
	TotalChecks    int
	PassedChecks   int
	FailedChecks   int
	Results        []VerificationResult
	FailureDetails []map[string]interface{}
}

// Add appends a result and updates counters.
func (s *VerificationSummary) Add(result VerificationResult) {
	s.Results = append(s.Results, result)
	s.TotalChecks++

	switch result.Status {
	case VerificationSatisfied:
		s.PassedChecks++
	case VerificationFailed:
		s.FailedChecks++
		if result.Details != nil {
			if s.FailureDetails == nil {
				s.FailureDetails = make([]map[string]interface{}, 0)
			}

			s.FailureDetails = append(s.FailureDetails, result.Details)
		}
	}
}

// Merge combines another summary into this one.
func (s *VerificationSummary) Merge(other VerificationSummary) {
	s.TotalChecks += other.TotalChecks
	s.PassedChecks += other.PassedChecks
	s.FailedChecks += other.FailedChecks

	s.Results = append(s.Results, other.Results...)
	if len(other.FailureDetails) > 0 {
		s.FailureDetails = append(s.FailureDetails, other.FailureDetails...)
	}
}

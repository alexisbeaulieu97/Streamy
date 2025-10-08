package model

// EvaluationResult contains the result of evaluating a step's current state
// against its desired state. This struct is returned by Plugin.Evaluate()
// and passed to Plugin.Apply() when action is required.
type EvaluationResult struct {
	// StepID is the unique identifier of the evaluated step
	StepID string

	// CurrentState represents the current state of the resource
	// relative to the desired state (Satisfied, Missing, Drifted, Blocked, Unknown)
	CurrentState VerificationStatus

	// RequiresAction indicates whether Apply() should be called
	// true for Missing or Drifted states, false for Satisfied, Blocked, or Unknown
	RequiresAction bool

	// Message is a human-readable description of the state assessment
	// Must be non-empty and explain what was found
	Message string

	// Diff is an optional formatted diff showing what would change
	// Should be populated when RequiresAction is true for dry-run previews
	Diff string

	// InternalData is opaque data passed from Evaluate() to Apply()
	// Used to avoid recomputation and pass domain-specific data
	InternalData any
}

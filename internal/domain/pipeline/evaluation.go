package pipeline

// EvaluationResult captures the outcome of evaluating a step without applying
// changes.
type EvaluationResult struct {
	RequiresAction bool
	CurrentState   string
	DesiredState   string
	Diff           string
	InternalData   interface{}
}

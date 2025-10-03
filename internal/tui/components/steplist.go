package components

import (
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

// StepEntry represents a single step for rendering.
type StepEntry struct {
	ID     string
	Result model.StepResult
}

// StepList renders a list of steps with their current status.
type StepList struct {
	entries []StepEntry
}

// NewStepList constructs a step list component.
func NewStepList(order []string, steps map[string]model.StepResult) StepList {
	entries := make([]StepEntry, 0, len(order))
	for _, id := range order {
		entries = append(entries, StepEntry{ID: id, Result: steps[id]})
	}
	return StepList{entries: entries}
}

// Entries returns the ordered step entries.
func (s StepList) Entries() []StepEntry {
	clone := make([]StepEntry, len(s.entries))
	copy(clone, s.entries)
	return clone
}

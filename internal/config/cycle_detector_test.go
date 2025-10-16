package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		target   string
		expected int
	}{
		{
			name:     "found at beginning",
			slice:    []string{"a", "b", "c"},
			target:   "a",
			expected: 0,
		},
		{
			name:     "found in middle",
			slice:    []string{"a", "b", "c"},
			target:   "b",
			expected: 1,
		},
		{
			name:     "found at end",
			slice:    []string{"a", "b", "c"},
			target:   "c",
			expected: 2,
		},
		{
			name:     "not found",
			slice:    []string{"a", "b", "c"},
			target:   "d",
			expected: -1,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			target:   "a",
			expected: -1,
		},
		{
			name:     "single element found",
			slice:    []string{"x"},
			target:   "x",
			expected: 0,
		},
		{
			name:     "single element not found",
			slice:    []string{"x"},
			target:   "y",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.slice, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectCycle_NoCycle(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true},
		{ID: "b", Enabled: true, DependsOn: []string{"a"}},
		{ID: "c", Enabled: true, DependsOn: []string{"b"}},
	}

	cycle := detectCycle(steps)
	assert.Nil(t, cycle)
}

func TestDetectCycle_SimpleDirectCycle(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true, DependsOn: []string{"b"}},
		{ID: "b", Enabled: true, DependsOn: []string{"a"}},
	}

	cycle := detectCycle(steps)
	require.NotNil(t, cycle)
	assert.True(t, len(cycle) >= 2, "cycle should have at least 2 nodes")
	assert.Contains(t, cycle, "a")
	assert.Contains(t, cycle, "b")
}

func TestDetectCycle_IndirectCycle(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true, DependsOn: []string{"b"}},
		{ID: "b", Enabled: true, DependsOn: []string{"c"}},
		{ID: "c", Enabled: true, DependsOn: []string{"a"}},
	}

	cycle := detectCycle(steps)
	require.NotNil(t, cycle)
	assert.Equal(t, 4, len(cycle)) // a->b->c->a
	assert.Contains(t, cycle, "a")
	assert.Contains(t, cycle, "b")
	assert.Contains(t, cycle, "c")
}

func TestDetectCycle_SelfCycle(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true, DependsOn: []string{"a"}},
	}

	cycle := detectCycle(steps)
	require.NotNil(t, cycle)
	assert.Contains(t, cycle, "a")
}

func TestDetectCycle_DisabledStepsIgnored(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true},
		{ID: "b", Enabled: false, DependsOn: []string{"a"}},
		{ID: "c", Enabled: true, DependsOn: []string{"b"}},
	}

	cycle := detectCycle(steps)
	assert.Nil(t, cycle)
}

func TestDetectCycle_DependencyOnDisabledIgnored(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true, DependsOn: []string{"disabled"}},
		{ID: "disabled", Enabled: false},
	}

	cycle := detectCycle(steps)
	assert.Nil(t, cycle)
}

func TestDetectCycle_EmptySteps(t *testing.T) {
	steps := []Step{}
	cycle := detectCycle(steps)
	assert.Nil(t, cycle)
}

func TestDetectCycle_SingleEnabledStep(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true},
	}

	cycle := detectCycle(steps)
	assert.Nil(t, cycle)
}

func TestDetectCycle_MultipleComponentsNoCycle(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true},
		{ID: "b", Enabled: true, DependsOn: []string{"a"}},
		{ID: "c", Enabled: true}, // separate component
		{ID: "d", Enabled: true, DependsOn: []string{"c"}},
	}

	cycle := detectCycle(steps)
	assert.Nil(t, cycle)
}

func TestDetectCycle_CycleInOneComponent(t *testing.T) {
	steps := []Step{
		{ID: "a", Enabled: true},
		{ID: "b", Enabled: true, DependsOn: []string{"a"}},
		{ID: "c", Enabled: true}, // separate component with cycle
		{ID: "d", Enabled: true, DependsOn: []string{"c", "e"}},
		{ID: "e", Enabled: true, DependsOn: []string{"d"}},
	}

	cycle := detectCycle(steps)
	require.NotNil(t, cycle)
	assert.Contains(t, cycle, "d")
	assert.Contains(t, cycle, "e")
}

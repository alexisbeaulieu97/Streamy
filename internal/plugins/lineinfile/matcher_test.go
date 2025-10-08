package lineinfileplugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceLinesStrategies(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma", "beta"}
	matches := &MatchResult{
		Matched:     true,
		LineNumbers: []int{1, 3},
		MatchCount:  2,
	}

	t.Run("first strategy replaces only first occurrence", func(t *testing.T) {
		updated, changed, err := replaceLines(append([]string{}, lines...), matches, "delta", onMultipleFirst)
		require.NoError(t, err)
		require.True(t, changed)
		require.Equal(t, []string{"alpha", "delta", "gamma", "beta"}, updated)
	})

	t.Run("all strategy replaces all occurrences", func(t *testing.T) {
		updated, changed, err := replaceLines(append([]string{}, lines...), matches, "delta", onMultipleAll)
		require.NoError(t, err)
		require.True(t, changed)
		require.Equal(t, []string{"alpha", "delta", "gamma", "delta"}, updated)
	})

	t.Run("error strategy fails on multiple matches", func(t *testing.T) {
		_, changed, err := replaceLines(append([]string{}, lines...), matches, "delta", onMultipleError)
		require.Error(t, err)
		require.False(t, changed)
	})

	t.Run("prompt strategy errors on multiple matches", func(t *testing.T) {
		_, changed, err := replaceLines(append([]string{}, lines...), matches, "delta", onMultiplePrompt)
		require.Error(t, err)
		require.False(t, changed)
	})

	t.Run("prompt strategy updates single match", func(t *testing.T) {
		single := &MatchResult{Matched: true, LineNumbers: []int{1}, MatchCount: 1}
		updated, changed, err := replaceLines(append([]string{}, lines...), single, "delta", onMultiplePrompt)
		require.NoError(t, err)
		require.True(t, changed)
		require.Equal(t, []string{"alpha", "delta", "gamma", "beta"}, updated)
	})

	t.Run("unknown strategy returns error", func(t *testing.T) {
		_, changed, err := replaceLines(append([]string{}, lines...), matches, "delta", "invalid")
		require.Error(t, err)
		require.False(t, changed)
	})
}

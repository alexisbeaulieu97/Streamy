package lineinfileplugin

import (
	"regexp"
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestFindMatchesNoMatch(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma"}
	pattern, _ := regexp.Compile("delta")
	matches := findMatches(lines, pattern)
	require.False(t, matches.Matched)
}

func TestAppendLineIfMissingExisting(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma"}
	updated, changed := appendLineIfMissing(lines, "beta")
	require.False(t, changed)
	require.Equal(t, lines, updated)
}

func TestRemoveMatchedLinesNoMatch(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma"}
	matches := &MatchResult{}
	updated, changed := removeMatchedLines(lines, matches)
	require.False(t, changed)
	require.Equal(t, lines, updated)
}

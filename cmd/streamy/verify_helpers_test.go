package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestGetStatusSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status model.VerificationStatus
		want   string
	}{
		{"satisfied", model.StatusSatisfied, "✔"},
		{"missing", model.StatusMissing, "✖"},
		{"drifted", model.StatusDrifted, "⚠"},
		{"blocked", model.StatusBlocked, "🚫"},
		{"unknown", model.StatusUnknown, "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getStatusSymbol(tt.status)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "1234567890",
			maxLen:   10,
			expected: "1234567890",
		},
		{
			name:     "needs truncation",
			input:    "this is a very long string that needs truncation",
			maxLen:   20,
			expected: "this is a very lo...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncateString(tt.input, tt.maxLen)
			require.Equal(t, tt.expected, got)
			require.LessOrEqual(t, len(got), tt.maxLen)
		})
	}
}

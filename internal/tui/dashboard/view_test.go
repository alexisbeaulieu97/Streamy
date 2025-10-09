package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatLastRun(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "zero time",
			time:     time.Time{},
			expected: "Never",
		},
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "Just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-72 * time.Hour),
			expected: "3 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLastRun(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatLastRunOldDates(t *testing.T) {
	// For dates older than 7 days, should return formatted date
	oldDate := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	result := FormatLastRun(oldDate)
	assert.Equal(t, "Jan 1, 2025", result)
}

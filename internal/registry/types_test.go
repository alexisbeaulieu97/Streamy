package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipelineStatus_Icon(t *testing.T) {
	tests := []struct {
		name   string
		status PipelineStatus
		want   string
	}{
		{"satisfied", StatusSatisfied, "🟢"},
		{"drifted", StatusDrifted, "🟡"},
		{"failed", StatusFailed, "🔴"},
		{"unknown", StatusUnknown, "⚪"},
		{"verifying", StatusVerifying, "⚪"},
		{"applying", StatusApplying, "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Icon())
		})
	}
}

func TestPipelineStatus_IconFallback(t *testing.T) {
	tests := []struct {
		name   string
		status PipelineStatus
		want   string
	}{
		{"satisfied", StatusSatisfied, "[OK]"},
		{"drifted", StatusDrifted, "[!!]"},
		{"failed", StatusFailed, "[XX]"},
		{"unknown", StatusUnknown, "[??]"},
		{"verifying", StatusVerifying, "[??]"},
		{"applying", StatusApplying, "[??]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IconFallback())
		})
	}
}

func TestPipelineStatus_Color(t *testing.T) {
	tests := []struct {
		name   string
		status PipelineStatus
	}{
		{"satisfied", StatusSatisfied},
		{"drifted", StatusDrifted},
		{"failed", StatusFailed},
		{"unknown", StatusUnknown},
		{"verifying", StatusVerifying},
		{"applying", StatusApplying},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := tt.status.Color()
			assert.NotNil(t, color)
		})
	}
}

func TestPipelineStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status PipelineStatus
		want   string
	}{
		{"satisfied", StatusSatisfied, "satisfied"},
		{"drifted", StatusDrifted, "drifted"},
		{"failed", StatusFailed, "failed"},
		{"unknown", StatusUnknown, "unknown"},
		{"verifying", StatusVerifying, "verifying"},
		{"applying", StatusApplying, "applying"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

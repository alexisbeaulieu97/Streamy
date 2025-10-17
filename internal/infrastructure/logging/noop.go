package logging

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// NoOpLogger discards all log entries.
type NoOpLogger struct{}

// Debug implements ports.Logger.
func (n *NoOpLogger) Debug(context.Context, string, ...interface{}) {}

// Info implements ports.Logger.
func (n *NoOpLogger) Info(context.Context, string, ...interface{}) {}

// Warn implements ports.Logger.
func (n *NoOpLogger) Warn(context.Context, string, ...interface{}) {}

// Error implements ports.Logger.
func (n *NoOpLogger) Error(context.Context, string, ...interface{}) {}

// With implements ports.Logger.
func (n *NoOpLogger) With(...interface{}) ports.Logger { return n }

// NewNoOpLogger returns a ports.Logger that discards all log entries.
func NewNoOpLogger() ports.Logger {
	return &NoOpLogger{}
}

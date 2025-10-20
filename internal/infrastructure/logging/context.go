package logging

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// WithCorrelationID stores the provided correlation identifier inside the context.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return ports.WithCorrelationID(ctx, id)
}

// GetCorrelationID retrieves the correlation identifier from the context, returning
// an empty string when none is present.
func GetCorrelationID(ctx context.Context) string {
	return ports.GetCorrelationID(ctx)
}

// GenerateCorrelationID creates a new correlation identifier suitable for request tracing.
func GenerateCorrelationID() string {
	return ports.GenerateCorrelationID()
}

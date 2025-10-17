package ports

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Logger defines Streamy's structured logging contract. All log calls are
// key/value pairs, must be safe for concurrent use, and should automatically
// enrich entries with a correlation ID when present in context. Common fields
// (see specs/009-domain-driven-refactor/observability.md) include:
//   - correlation_id (UUIDv4, generated at CLI entry point)
//   - layer (domain|application|infrastructure)
//   - component (executor, loader, registry, etc.)
//   - step_id / plugin_type / validation_type
//   - duration_ms for timed operations
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, msg string, fields ...interface{})
	Error(ctx context.Context, msg string, fields ...interface{})
	With(fields ...interface{}) Logger
}

type correlationIDKey struct{}

// WithCorrelationID attaches the provided correlation ID to the context so
// downstream layers can emit correlated logs, metrics, and traces.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, id)
}

// GetCorrelationID extracts a correlation ID from context. It returns an empty
// string when none has been set—callers should treat that as "uncorrelated".
func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return ""
}

// GenerateCorrelationID produces a new UUIDv4 string suitable for log
// correlation. CLI entry-points should invoke this once per command execution.
func GenerateCorrelationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("failed to generate correlation id: %v", err))
	}
	// Set UUID version (4) and variant bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	var encoded [32]byte
	hex.Encode(encoded[:], b[:])

	return fmt.Sprintf("%s-%s-%s-%s-%s",
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	)
}

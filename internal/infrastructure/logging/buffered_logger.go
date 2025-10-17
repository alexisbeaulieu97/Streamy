package logging

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// BufferedLogger implements ports.Logger by writing into an EventBuffer.
type BufferedLogger struct {
	buffer *EventBuffer
	fields []interface{}
}

// NewBufferedLogger returns a logger that stores entries in the provided buffer.
func NewBufferedLogger(buffer *EventBuffer) *BufferedLogger {
	return &BufferedLogger{buffer: buffer}
}

// Debug records a debug message in the buffer.
func (l *BufferedLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, levelDebug, msg, fields...)
}

// Info records an info message in the buffer.
func (l *BufferedLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, levelInfo, msg, fields...)
}

// Warn records a warning message in the buffer.
func (l *BufferedLogger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, levelWarn, msg, fields...)
}

// Error records an error message in the buffer.
func (l *BufferedLogger) Error(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, levelError, msg, fields...)
}

// With returns a child buffered logger with persistent fields.
func (l *BufferedLogger) With(fields ...interface{}) ports.Logger {
	nextFields := append(append([]interface{}{}, l.fields...), fields...)
	return &BufferedLogger{buffer: l.buffer, fields: nextFields}
}

func (l *BufferedLogger) log(ctx context.Context, level logLevel, msg string, fields ...interface{}) {
	if l == nil || l.buffer == nil {
		return
	}
	payload := append(append([]interface{}{}, l.fields...), fields...)
	l.buffer.add(bufferedEntry{
		ctx:    ctx,
		level:  level,
		msg:    msg,
		fields: payload,
	})
}

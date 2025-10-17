package logging

import (
	"context"
	"sync"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

const defaultBufferLimit = 1000

type logLevel int

const (
	levelDebug logLevel = iota
	levelInfo
	levelWarn
	levelError
)

type bufferedEntry struct {
	ctx    context.Context
	level  logLevel
	msg    string
	fields []interface{}
}

// EventBuffer stores log events emitted before the primary logger is ready.
type EventBuffer struct {
	mu     sync.Mutex
	limit  int
	events []bufferedEntry
}

// NewEventBuffer creates a buffer with the provided capacity (defaults to 1000).
func NewEventBuffer(limit int) *EventBuffer {
	if limit <= 0 {
		limit = defaultBufferLimit
	}
	return &EventBuffer{
		limit:  limit,
		events: make([]bufferedEntry, 0, limit),
	}
}

func (b *EventBuffer) add(entry bufferedEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) == b.limit {
		copy(b.events, b.events[1:])
		b.events[len(b.events)-1] = entry
		return
	}
	b.events = append(b.events, entry)
}

// Flush replays buffered events using the provided logger, preserving ordering.
func (b *EventBuffer) Flush(delegate ports.Logger) {
	if delegate == nil {
		return
	}
	b.mu.Lock()
	events := make([]bufferedEntry, len(b.events))
	copy(events, b.events)
	b.events = b.events[:0]
	b.mu.Unlock()

	for _, entry := range events {
		switch entry.level {
		case levelDebug:
			delegate.Debug(entry.ctx, entry.msg, entry.fields...)
		case levelWarn:
			delegate.Warn(entry.ctx, entry.msg, entry.fields...)
		case levelError:
			delegate.Error(entry.ctx, entry.msg, entry.fields...)
		default:
			delegate.Info(entry.ctx, entry.msg, entry.fields...)
		}
	}
}

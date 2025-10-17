package events

import (
	"context"
	"sort"
	"sync"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// LoggingPublisher emits domain events using the structured logger.
type LoggingPublisher struct {
	logger ports.Logger
	subs   map[string][]subscriptionEntry
	nextID int
	mu     sync.RWMutex
}

// NewLoggingPublisher creates an event publisher that writes each event as a structured log entry.
func NewLoggingPublisher(logger ports.Logger) *LoggingPublisher {
	return &LoggingPublisher{
		logger: logger,
		subs:   make(map[string][]subscriptionEntry),
	}
}

// Publish renders the event as a structured log entry.
func (p *LoggingPublisher) Publish(ctx context.Context, event ports.DomainEvent) error {
	if p == nil || p.logger == nil || event == nil {
		return nil
	}

	p.mu.RLock()
	handlers := append([]subscriptionEntry(nil), p.subs[event.EventType()]...)
	p.mu.RUnlock()

	fields := []interface{}{"event_type", event.EventType()}
	switch payload := event.Payload().(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(payload))
		for key := range payload {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fields = append(fields, key, payload[key])
		}
	case nil:
	default:
		fields = append(fields, "payload", payload)
	}

	p.logger.Info(ctx, "domain event", fields...)

	for _, entry := range handlers {
		handler := entry.handler
		if handler == nil {
			continue
		}
		if err := handler(ctx, event); err != nil {
			p.logger.Warn(ctx, "event handler failed", "event_type", event.EventType(), "error", err)
		}
	}

	return nil
}

// Subscribe registers a handler for the provided event type.
func (p *LoggingPublisher) Subscribe(eventType string, handler ports.EventHandler) (ports.Subscription, error) {
	if p == nil || handler == nil {
		return noopSubscription{}, nil
	}
	p.mu.Lock()
	p.nextID++
	id := p.nextID
	p.subs[eventType] = append(p.subs[eventType], subscriptionEntry{id: id, handler: handler})
	p.mu.Unlock()

	return subscription{
		cancel: func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			handlers := p.subs[eventType]
			for i, entry := range handlers {
				if entry.id == id {
					p.subs[eventType] = append(handlers[:i], handlers[i+1:]...)
					break
				}
			}
		},
	}, nil
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() {}

type subscription struct {
	cancel func()
}

func (s subscription) Unsubscribe() {
	if s.cancel != nil {
		s.cancel()
	}
}

type subscriptionEntry struct {
	id      int
	handler ports.EventHandler
}

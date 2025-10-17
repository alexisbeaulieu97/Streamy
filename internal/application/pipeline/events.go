package pipeline

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type domainEvent struct {
	eventType string
	payload   interface{}
}

func (e domainEvent) EventType() string {
	return e.eventType
}

func (e domainEvent) Payload() interface{} {
	return e.payload
}

func publishEvent(ctx context.Context, publisher ports.EventPublisher, logger ports.Logger, eventType string, payload map[string]interface{}) {
	if publisher == nil {
		return
	}
	event := domainEvent{
		eventType: eventType,
		payload:   payload,
	}
	if err := publisher.Publish(ctx, event); err != nil && logger != nil {
		logger.Warn(ctx, "failed to publish domain event", "event_type", eventType, "error", err)
	}
}

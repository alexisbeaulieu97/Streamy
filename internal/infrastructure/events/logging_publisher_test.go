package events

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	cblog "github.com/charmbracelet/log"
	"github.com/stretchr/testify/require"

	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

func TestLoggingPublisherIncludesCorrelationID(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger, err := logginginfra.New(logginginfra.Options{
		Writer:    buf,
		Level:     "info",
		Layer:     "test",
		Component: "publisher",
		Formatter: cblog.JSONFormatter,
	})
	require.NoError(t, err)

	publisher := NewLoggingPublisher(logger)

	ctx := logginginfra.WithCorrelationID(context.Background(), "abc-123")
	err = publisher.Publish(ctx, sampleEvent{
		eventType: ports.EventPipelineStarted,
		payload:   map[string]interface{}{"pipeline": "demo"},
	})
	require.NoError(t, err)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	require.Equal(t, "domain event", entry["msg"])
	require.Equal(t, ports.EventPipelineStarted, entry["event_type"])
	require.Equal(t, "abc-123", entry["correlation_id"])
	require.Equal(t, "demo", entry["pipeline"])
}

func TestLoggingPublisherInvokesSubscribers(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger, err := logginginfra.New(logginginfra.Options{
		Writer:    buf,
		Level:     "info",
		Layer:     "test",
		Component: "publisher",
		Formatter: cblog.JSONFormatter,
	})
	require.NoError(t, err)

	publisher := NewLoggingPublisher(logger)

	var handled bool
	_, err = publisher.Subscribe(ports.EventPipelineCompleted, func(ctx context.Context, event ports.DomainEvent) error {
		handled = true
		return nil
	})
	require.NoError(t, err)

	err = publisher.Publish(context.Background(), sampleEvent{
		eventType: ports.EventPipelineCompleted,
		payload:   map[string]interface{}{"pipeline": "demo"},
	})
	require.NoError(t, err)
	require.True(t, handled, "subscriber should be invoked")
}

type sampleEvent struct {
	eventType string
	payload   interface{}
}

func (e sampleEvent) EventType() string    { return e.eventType }
func (e sampleEvent) Payload() interface{} { return e.payload }

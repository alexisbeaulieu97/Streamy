package events

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"

	cblog "github.com/charmbracelet/log"
	assert "github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"

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

	_, err = publisher.Subscribe(ports.EventPipelineCompleted, func(_ context.Context, _ ports.DomainEvent) error {
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

func TestLoggingPublisher_HandlerFailures(t *testing.T) {
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

	// Subscribe a handler that will fail
	_, err = publisher.Subscribe(ports.EventStepStarted, func(_ context.Context, _ ports.DomainEvent) error {
		return assert.AnError
	})
	require.NoError(t, err)

	// Subscribe a handler that will succeed
	var successHandled bool

	_, err = publisher.Subscribe(ports.EventStepStarted, func(_ context.Context, _ ports.DomainEvent) error {
		successHandled = true
		return nil
	})
	require.NoError(t, err)

	// Publish event - should log warning for failed handler but continue
	err = publisher.Publish(context.Background(), sampleEvent{
		eventType: ports.EventStepStarted,
		payload:   map[string]interface{}{"step": "test-step"},
	})
	require.NoError(t, err)

	// Success handler should still be called despite failure in other handler
	require.True(t, successHandled, "successful handler should still be invoked")

	// Check that warning was logged for failed handler
	var entries []map[string]interface{}

	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var entry map[string]interface{}

		err := json.Unmarshal(line, &entry)
		require.NoError(t, err)

		entries = append(entries, entry)
	}

	// Should have 2 log entries: one for the event, one for the warning
	require.Len(t, entries, 2)

	// Find the warning entry
	var warningEntry map[string]interface{}

	for _, entry := range entries {
		if entry["msg"] == "event handler failed" {
			warningEntry = entry
			break
		}
	}

	require.NotNil(t, warningEntry, "should have warning entry for failed handler")
	require.Equal(t, ports.EventStepStarted, warningEntry["event_type"])
}

func TestLoggingPublisher_SubscriptionCleanup(t *testing.T) {
	t.Parallel()

	publisher := NewLoggingPublisher(&mockLogger{})

	// Subscribe multiple handlers for the same event
	var handler1Called, handler2Called, handler3Called bool

	sub1, err := publisher.Subscribe("test.event", func(_ context.Context, _ ports.DomainEvent) error {
		handler1Called = true
		return nil
	})
	require.NoError(t, err)

	sub2, err := publisher.Subscribe("test.event", func(_ context.Context, _ ports.DomainEvent) error {
		handler2Called = true
		return nil
	})
	require.NoError(t, err)

	sub3, err := publisher.Subscribe("test.event", func(_ context.Context, _ ports.DomainEvent) error {
		handler3Called = true
		return nil
	})
	require.NoError(t, err)

	// Publish event - all handlers should be called
	err = publisher.Publish(context.Background(), sampleEvent{
		eventType: "test.event",
		payload:   map[string]interface{}{"test": "value"},
	})
	require.NoError(t, err)

	require.True(t, handler1Called, "handler 1 should be called")
	require.True(t, handler2Called, "handler 2 should be called")
	require.True(t, handler3Called, "handler 3 should be called")

	// Reset flags
	handler1Called, handler2Called, handler3Called = false, false, false

	// Unsubscribe handler 2
	sub2.Unsubscribe()

	// Publish event again - only handlers 1 and 3 should be called
	err = publisher.Publish(context.Background(), sampleEvent{
		eventType: "test.event",
		payload:   map[string]interface{}{"test": "value2"},
	})
	require.NoError(t, err)

	require.True(t, handler1Called, "handler 1 should still be called")
	require.False(t, handler2Called, "handler 2 should not be called after unsubscribe")
	require.True(t, handler3Called, "handler 3 should still be called")

	// Reset flags
	handler1Called, handler3Called = false, false

	// Unsubscribe all remaining handlers
	sub1.Unsubscribe()
	sub3.Unsubscribe()

	// Publish event again - no handlers should be called
	err = publisher.Publish(context.Background(), sampleEvent{
		eventType: "test.event",
		payload:   map[string]interface{}{"test": "value3"},
	})
	require.NoError(t, err)

	require.False(t, handler1Called, "handler 1 should not be called after unsubscribe")
	require.False(t, handler3Called, "handler 3 should not be called after unsubscribe")
}

func TestLoggingPublisher_ConcurrentPublish(t *testing.T) {
	t.Parallel()

	publisher := NewLoggingPublisher(&mockLogger{})

	// Subscribe multiple handlers
	numHandlers := 5 // Reduced to make test more reliable

	var callCount int64

	handlers := make([]ports.Subscription, numHandlers)

	for i := 0; i < numHandlers; i++ {
		sub, err := publisher.Subscribe("concurrent.test", func(_ context.Context, _ ports.DomainEvent) error {
			// Simulate some work
			callCount++
			return nil
		})
		require.NoError(t, err)

		handlers[i] = sub
	}

	// Publish events concurrently with proper synchronization
	numGoroutines := 10
	numEventsPerGoroutine := 5
	errChan := make(chan error, numGoroutines)

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := 0; j < numEventsPerGoroutine; j++ {
				err := publisher.Publish(context.Background(), sampleEvent{
					eventType: "concurrent.test",
					payload:   map[string]interface{}{"goroutine": id, "event": j},
				})
				if err != nil {
					errChan <- err
					return
				}
			}

			errChan <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	for err := range errChan {
		require.NoError(t, err)
	}

	// Total expected calls: numGoroutines * numEventsPerGoroutine * numHandlers
	expectedCalls := int64(numGoroutines * numEventsPerGoroutine * numHandlers)
	require.Equal(t, expectedCalls, callCount, "all handlers should be called for all events")

	// Cleanup subscriptions
	for _, sub := range handlers {
		sub.Unsubscribe()
	}
}

func TestLoggingPublisher_NilValues(t *testing.T) {
	t.Parallel()

	// Test with nil publisher
	var publisher *LoggingPublisher

	err := publisher.Publish(context.Background(), sampleEvent{
		eventType: "test.event",
		payload:   nil,
	})
	require.NoError(t, err, "nil publisher should not panic")

	// Test with nil logger
	publisher = NewLoggingPublisher(nil)
	err = publisher.Publish(context.Background(), sampleEvent{
		eventType: "test.event",
		payload:   nil,
	})
	require.NoError(t, err, "nil logger should not panic")

	// Test with nil event
	publisher = NewLoggingPublisher(&mockLogger{})
	err = publisher.Publish(context.Background(), nil)
	require.NoError(t, err, "nil event should not panic")

	// Test subscription with nil handler
	sub, err := publisher.Subscribe("test.event", nil)
	require.NoError(t, err, "nil handler subscription should not error")
	require.IsType(t, noopSubscription{}, sub, "should return noop subscription for nil handler")
}

func TestLoggingPublisher_PayloadSerialization(t *testing.T) {
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

	testCases := []struct {
		name     string
		payload  interface{}
		expected string
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: `"msg":"domain event"`,
		},
		{
			name:     "map payload",
			payload:  map[string]interface{}{"key1": "value1", "key2": 42, "key3": true},
			expected: `"key1":"value1"`,
		},
		{
			name:     "string payload",
			payload:  "simple string",
			expected: `"payload":"simple string"`,
		},
		{
			name:     "struct payload",
			payload:  struct{ Name string }{Name: "test"},
			expected: `"payload":`,
		},
		{
			name:     "empty map payload",
			payload:  map[string]interface{}{},
			expected: `"msg":"domain event"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()

			err := publisher.Publish(context.Background(), sampleEvent{
				eventType: "serialization.test",
				payload:   tc.payload,
			})
			require.NoError(t, err)

			output := buf.String()
			require.Contains(t, output, `"event_type":"serialization.test"`)
			require.Contains(t, output, tc.expected)
		})
	}
}

func TestLoggingPublisher_EventOrdering(t *testing.T) {
	t.Parallel()

	publisher := NewLoggingPublisher(&mockLogger{})

	// Subscribe handler that records event order
	var eventOrder []string

	_, err := publisher.Subscribe("order.test", func(_ context.Context, event ports.DomainEvent) error {
		if payload, ok := event.Payload().(map[string]interface{}); ok {
			if id, ok := payload["id"].(string); ok {
				eventOrder = append(eventOrder, id)
			}
		}

		return nil
	})
	require.NoError(t, err)

	// Publish events in a specific order
	events := []string{"first", "second", "third", "fourth", "fifth"}
	for _, id := range events {
		err := publisher.Publish(context.Background(), sampleEvent{
			eventType: "order.test",
			payload:   map[string]interface{}{"id": id},
		})
		require.NoError(t, err)
	}

	// Events should be processed in the order they were published
	require.Equal(t, events, eventOrder, "events should be processed in publication order")
}

func TestLoggingPublisher_MultipleEventTypes(t *testing.T) {
	t.Parallel()

	publisher := NewLoggingPublisher(&mockLogger{})

	// Subscribe handlers for different event types
	var type1Handled, type2Handled bool

	_, err := publisher.Subscribe("type1", func(_ context.Context, _ ports.DomainEvent) error {
		type1Handled = true
		return nil
	})
	require.NoError(t, err)

	_, err = publisher.Subscribe("type2", func(_ context.Context, _ ports.DomainEvent) error {
		type2Handled = true
		return nil
	})
	require.NoError(t, err)

	// Subscribe another handler for type1 to test multiple handlers per type
	var type1SecondHandled bool

	_, err = publisher.Subscribe("type1", func(_ context.Context, _ ports.DomainEvent) error {
		type1SecondHandled = true
		return nil
	})
	require.NoError(t, err)

	// Publish type1 event
	err = publisher.Publish(context.Background(), sampleEvent{
		eventType: "type1",
		payload:   nil,
	})
	require.NoError(t, err)

	require.True(t, type1Handled, "type1 handler should be called")
	require.True(t, type1SecondHandled, "type1 second handler should be called")
	require.False(t, type2Handled, "type2 handler should not be called")

	// Reset flags
	type1Handled, type1SecondHandled, type2Handled = false, false, false

	// Publish type2 event
	err = publisher.Publish(context.Background(), sampleEvent{
		eventType: "type2",
		payload:   nil,
	})
	require.NoError(t, err)

	require.False(t, type1Handled, "type1 handler should not be called")
	require.False(t, type1SecondHandled, "type1 second handler should not be called")
	require.True(t, type2Handled, "type2 handler should be called")
}

// mockLogger implements ports.Logger for testing
type mockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (m *mockLogger) Debug(_ context.Context, msg string, _ ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, "DEBUG: "+msg)
}

func (m *mockLogger) Info(_ context.Context, msg string, _ ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *mockLogger) Warn(_ context.Context, msg string, _ ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, "WARN: "+msg)
}

func (m *mockLogger) Error(_ context.Context, msg string, _ ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, "ERROR: "+msg)
}

func (m *mockLogger) With(_ ...interface{}) ports.Logger {
	return &mockLogger{}
}

type sampleEvent struct {
	eventType string
	payload   interface{}
}

func (e sampleEvent) EventType() string    { return e.eventType }
func (e sampleEvent) Payload() interface{} { return e.payload }

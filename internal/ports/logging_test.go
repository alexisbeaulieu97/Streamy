package ports

import (
	"context"
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func TestCorrelationID(t *testing.T) {
	ctx := context.Background()

	// Test GetCorrelationID with no ID in context
	assert.Empty(t, GetCorrelationID(ctx))

	// Test WithCorrelationID and GetCorrelationID
	id := "test-id"
	ctxWithID := WithCorrelationID(ctx, id)
	assert.Equal(t, id, GetCorrelationID(ctxWithID))

	// Test that the original context is not modified
	assert.Empty(t, GetCorrelationID(ctx))
}

func TestGenerateCorrelationID(t *testing.T) {
	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 36) // UUIDs have a fixed length
}

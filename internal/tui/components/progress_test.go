package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewProgress(t *testing.T) {
	t.Parallel()

	t.Run("creates progress with specified total", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(10)
		require.NotNil(t, p.bar)
		require.Equal(t, 10, p.total)
	})

	t.Run("creates progress with zero total", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(0)
		require.NotNil(t, p.bar)
		require.Equal(t, 0, p.total)
	})
}

func TestProgressView(t *testing.T) {
	t.Parallel()

	t.Run("renders with zero total", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(0)
		view := p.View(0)
		require.Contains(t, view, "0/0")
	})

	t.Run("renders with partial completion", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(10)
		view := p.View(5)
		require.Contains(t, view, "5/10")
		require.NotEmpty(t, view)
	})

	t.Run("renders with full completion", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(10)
		view := p.View(10)
		require.Contains(t, view, "10/10")
		require.NotEmpty(t, view)
	})

	t.Run("handles completion beyond total", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(10)
		view := p.View(15)
		require.Contains(t, view, "15/10")
		// Should cap at 100% visually but show actual count
		require.NotEmpty(t, view)
	})

	t.Run("renders with single step total", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(1)
		view := p.View(0)
		require.Contains(t, view, "0/1")

		view = p.View(1)
		require.Contains(t, view, "1/1")
	})

	t.Run("view contains both label and progress bar", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(10)
		view := p.View(5)
		// View should contain both numeric label and some progress indicator
		require.True(t, len(view) > len("5/10"))
		require.Contains(t, view, "5/10")
	})
}

func TestProgressViewFormat(t *testing.T) {
	t.Parallel()

	t.Run("progress bar takes up space", func(t *testing.T) {
		t.Parallel()
		p := NewProgress(100)
		view := p.View(50)

		// The view should be longer than just the label
		label := "50/100"
		require.True(t, len(strings.TrimSpace(view)) > len(label),
			"expected view to contain progress bar in addition to label")
	})
}

package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCard(t *testing.T) {
	data := CardData{
		Title:       "Test Card",
		Description: "This is a test card",
		Icon:        "📝",
	}

	card := NewCard(data)

	require.NotNil(t, card)
	assert.Equal(t, data, card.data)
	assert.Equal(t, DefaultCardStyle(), card.style)
}

func TestCardWithStyle(t *testing.T) {
	data := CardData{Title: "Test"}
	card := NewCard(data)

	customStyle := DefaultCardStyle()
	customStyle.Width = 80

	result := card.WithStyle(customStyle)

	assert.Equal(t, customStyle, card.style)
	assert.Same(t, card, result)
}

func TestCardWithTitleStyle(t *testing.T) {
	data := CardData{Title: "Test"}
	card := NewCard(data)

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	result := card.WithTitleStyle(titleStyle)

	assert.Equal(t, titleStyle, card.style.TitleStyle)
	assert.Same(t, card, result)
}

func TestCardWithContentStyle(t *testing.T) {
	data := CardData{Title: "Test"}
	card := NewCard(data)

	contentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	result := card.WithContentStyle(contentStyle)

	assert.Equal(t, contentStyle, card.style.ContentStyle)
	assert.Same(t, card, result)
}

func TestCardWithIconStyle(t *testing.T) {
	data := CardData{Title: "Test"}
	card := NewCard(data)

	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	result := card.WithIconStyle(iconStyle)

	assert.Equal(t, iconStyle, card.style.IconStyle)
	assert.Same(t, card, result)
}

func TestCardWithBorderStyle(t *testing.T) {
	data := CardData{Title: "Test"}
	card := NewCard(data)

	borderStyle := lipgloss.NewStyle().Border(lipgloss.DoubleBorder())
	result := card.WithBorderStyle(borderStyle)

	assert.Equal(t, borderStyle, card.style.BorderStyle)
	assert.Same(t, card, result)
}

func TestCardWithWidth(t *testing.T) {
	data := CardData{Title: "Test"}
	card := NewCard(data)

	result := card.WithWidth(80)

	assert.Equal(t, 80, card.style.Width)
	assert.Same(t, card, result)
}

func TestCardWithBorder(t *testing.T) {
	data := CardData{Title: "Test"}
	card := NewCard(data)

	border := lipgloss.ThickBorder()
	result := card.WithBorder(border)

	assert.Contains(t, card.style.BorderStyle.String(), "┏")
	assert.Same(t, card, result)
}

func TestCardViewWithTitleOnly(t *testing.T) {
	data := CardData{
		Title: "Test Card",
	}

	card := NewCard(data)
	view := card.View()

	assert.Contains(t, view, "Test Card")
	assert.NotContains(t, view, "Description")
}

func TestCardViewWithAllFields(t *testing.T) {
	data := CardData{
		Title:       "Test Card",
		Description: "This is a test description",
		Icon:        "📝",
		Status:      "Running",
		Metadata: map[string]string{
			"Version": "1.0.0",
			"Author":  "Test User",
		},
		Actions: []string{"Start", "Stop", "Restart"},
	}

	card := NewCard(data)
	view := card.View()

	assert.Contains(t, view, "Test Card")
	assert.Contains(t, view, "This is a test description")
	assert.Contains(t, view, "Running")
	assert.Contains(t, view, "Version: 1.0.0")
	assert.Contains(t, view, "Author: Test User")
	assert.Contains(t, view, "• Start")
	assert.Contains(t, view, "• Stop")
	assert.Contains(t, view, "• Restart")
}

func TestCardViewTextWrapping(t *testing.T) {
	longText := "This is a very long text that should wrap when the card width is limited to test the text wrapping functionality properly."
	data := CardData{
		Title:       "Test Card",
		Description: longText,
	}

	card := NewCard(data).WithWidth(30)
	view := card.View()

	assert.Contains(t, view, "Test Card")
	assert.Contains(t, view, "This is a")
	assert.Contains(t, view, "very long")
}

func TestCardViewEmptyData(t *testing.T) {
	data := CardData{}
	card := NewCard(data)
	view := card.View()

	assert.NotEmpty(t, view)
}

func TestStatusCard(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expectedIcon string
	}{
		{"Success", "success", "✓"},
		{"Error", "error", "✗"},
		{"Failed", "failed", "✗"},
		{"Warning", "warning", "⚠"},
		{"Info", "info", "ℹ"},
		{"Unknown", "unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := CardData{
				Title: "Test Card",
			}

			card := StatusCard(data, tt.status)
			view := card.View()

			assert.Contains(t, view, "Test Card")
			if tt.expectedIcon != "" {
				assert.Contains(t, view, tt.expectedIcon)
			}
		})
	}
}

func TestStatusCardCustomIcon(t *testing.T) {
	data := CardData{
		Title: "Test Card",
		Icon:  "🎯",
	}

	card := StatusCard(data, "success")
	view := card.View()

	assert.Contains(t, view, "Test Card")
	assert.Contains(t, view, "🎯")
	assert.NotContains(t, view, "✓")
}

func TestDefaultCardStyle(t *testing.T) {
	style := DefaultCardStyle()

	assert.Greater(t, style.Width, 0)
	assert.Greater(t, style.Padding, 0)
	assert.NotEqual(t, lipgloss.Style{}, style.BorderStyle)
	assert.NotEqual(t, lipgloss.Style{}, style.TitleStyle)
	assert.NotEqual(t, lipgloss.Style{}, style.ContentStyle)
	assert.NotEqual(t, lipgloss.Style{}, style.IconStyle)
}

func TestCardRenderHeader(t *testing.T) {
	tests := []struct {
		name     string
		data     CardData
		expected string
	}{
		{
			name: "Title only",
			data: CardData{Title: "Test"},
			expected: "Test",
		},
		{
			name: "Title and icon",
			data: CardData{Title: "Test", Icon: "📝"},
			expected: "📝 Test",
		},
		{
			name: "Icon only",
			data: CardData{Icon: "📝"},
			expected: "📝 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := NewCard(tt.data)
			header := card.renderHeader()
			assert.Contains(t, header, tt.expected)
		})
	}
}

func TestCardWrapText(t *testing.T) {
	card := NewCard(CardData{})

	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{
			name:     "Short text",
			text:     "Short",
			width:    20,
			expected: "Short",
		},
		{
			name:     "Long text",
			text:     "This is a very long text",
			width:    10,
			expected: "This is a\nvery long\ntext",
		},
		{
			name:     "Single word longer than width",
			text:     "supercalifragilisticexpialidocious",
			width:    10,
			expected: "supercalif\nragilistic\nexpialidoc\nious",
		},
		{
			name:     "Empty text",
			text:     "",
			width:    10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card.WithWidth(tt.width + 8) // Account for padding/border
			result := card.wrapText(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

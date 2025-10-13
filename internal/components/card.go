package components

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// CardStyle defines the visual appearance of a Card component.
// It includes styles for borders, title, content, icons, and dimensions.
type CardStyle struct {
	// BorderStyle applies to the card's outer border
	BorderStyle lipgloss.Style
	// TitleStyle applies to the card's title text
	TitleStyle lipgloss.Style
	// ContentStyle applies to the card's body content
	ContentStyle lipgloss.Style
	// IconStyle applies to the card's icon (if present)
	IconStyle lipgloss.Style
	// Width is the maximum width of the card in characters
	Width int
	// Padding is the internal spacing around card content
	Padding int
}

// DefaultCardStyle returns a default card style using the current theme.
func DefaultCardStyle() CardStyle {
	baseStyle := Style(lipgloss.NewStyle(), CardBaseStyle()...)

	return CardStyle{
		BorderStyle: baseStyle,
		TitleStyle: Style(
			lipgloss.NewStyle(),
			Typography(TypographyVariantTitle),
			Foreground(PalettePrimary),
		),
		ContentStyle: Style(lipgloss.NewStyle(), Typography(TypographyVariantBody)),
		IconStyle: Style(
			lipgloss.NewStyle(),
			Foreground(PaletteInfo),
		),
		Width:   60,
		Padding: PaddingValue(SpacingSizeSmall),
	}
}

// CardData represents the content and metadata of a card.
type CardData struct {
	// Title is the main heading displayed in the card
	Title string
	// Description is the main body text of the card
	Description string
	// Icon is an optional icon/emoji displayed before the title
	Icon string
	// Status represents the card's status (used for styling)
	Status string
	// Metadata contains additional key-value pairs to display
	Metadata map[string]string
	// Actions is a list of actionable items related to the card
	Actions []string
}

// Card represents a reusable card component with customizable styling.
type Card struct {
	data  CardData
	style CardStyle
}

// NewCard creates a new card with the given data.
func NewCard(data CardData) *Card {
	return &Card{
		data:  data,
		style: DefaultCardStyle(),
	}
}

// WithStyle sets a custom style for the card.
func (c *Card) WithStyle(style CardStyle) *Card {
	c.style = style
	return c
}

// WithTitleStyle sets a custom style for just the title.
func (c *Card) WithTitleStyle(style lipgloss.Style) *Card {
	c.style.TitleStyle = style
	return c
}

// WithContentStyle sets a custom style for just the content.
func (c *Card) WithContentStyle(style lipgloss.Style) *Card {
	c.style.ContentStyle = style
	return c
}

// WithIconStyle sets a custom style for just the icon.
func (c *Card) WithIconStyle(style lipgloss.Style) *Card {
	c.style.IconStyle = style
	return c
}

// WithBorderStyle sets a custom style for just the border.
func (c *Card) WithBorderStyle(style lipgloss.Style) *Card {
	c.style.BorderStyle = style
	return c
}

// WithWidth sets the card width.
func (c *Card) WithWidth(width int) *Card {
	c.style.Width = width
	c.style.BorderStyle = c.style.BorderStyle.Width(width)
	return c
}

// WithBorder sets the border style.
func (c *Card) WithBorder(border lipgloss.Border) *Card {
	c.style.BorderStyle = c.style.BorderStyle.Border(border)
	return c
}

// View renders the card.
func (c *Card) View() string {
	var content []string

	// Header with icon and title
	if c.data.Title != "" {
		header := c.renderHeader()
		content = append(content, header)
	}

	// Description
	if c.data.Description != "" {
		description := c.style.ContentStyle.Render(c.wrapText(c.data.Description))
		content = append(content, description)
	}

	// Status
	if c.data.Status != "" {
		status := c.style.ContentStyle.Render(c.data.Status)
		content = append(content, "")
		content = append(content, status)
	}

	// Metadata
	if len(c.data.Metadata) > 0 {
		content = append(content, "")
		// Sort keys for deterministic output
		keys := make([]string, 0, len(c.data.Metadata))
		for k := range c.data.Metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			line := fmt.Sprintf("%s: %s", key, c.data.Metadata[key])
			content = append(content, c.style.ContentStyle.Render(line))
		}
	}

	// Actions
	if len(c.data.Actions) > 0 {
		content = append(content, "")
		for _, action := range c.data.Actions {
			content = append(content, c.style.ContentStyle.Render("• "+action))
		}
	}

	innerContent := strings.Join(content, "\n")
	return c.style.BorderStyle.Render(innerContent)
}

// renderHeader creates the header with icon and title.
func (c *Card) renderHeader() string {
	var header strings.Builder

	if c.data.Icon != "" {
		icon := c.style.IconStyle.Render(c.data.Icon + " ")
		header.WriteString(icon)
	}

	title := c.style.TitleStyle.Render(c.data.Title)
	header.WriteString(title)

	return header.String()
}

// wrapText wraps text to fit within the card width.
// It handles long words by breaking them across multiple lines.
func (c *Card) wrapText(text string) string {
	if c.style.Width <= 0 {
		return text
	}

	borderWidth := horizontalBorderWidth(c.style.BorderStyle)
	paddingWidth := c.style.Padding * 2

	maxWidth := c.style.Width - paddingWidth - borderWidth
	if maxWidth <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		// Handle words longer than maxWidth
		if utf8.RuneCountInString(word) > maxWidth {
			wordRunes := []rune(word)
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = ""
			}
			// Break long word across lines
			for len(wordRunes) > maxWidth {
				lines = append(lines, string(wordRunes[:maxWidth]))
				wordRunes = wordRunes[maxWidth:]
			}
			if len(wordRunes) > 0 {
				currentLine = string(wordRunes)
			}
			continue
		}

		testLine := currentLine
		if currentLine != "" {
			testLine += " "
		}
		testLine += word

		if utf8.RuneCountInString(testLine) <= maxWidth {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// horizontalBorderWidth sums left and right border sizes, falling back to zero on error.
func horizontalBorderWidth(style lipgloss.Style) (width int) {
	defer func() {
		if recover() != nil {
			width = 0
		}
	}()

	width = style.GetBorderLeftSize() + style.GetBorderRightSize()
	if width < 0 {
		return 0
	}
	return width
}

// StatusCard creates a card with status-specific styling using theme colors.
func StatusCard(data CardData, status string) *Card {
	style := DefaultCardStyle()
	var statusStyle []StyleApplier

	switch status {
	case "success":
		statusStyle = []StyleApplier{
			Foreground(PaletteSuccess),
		}
		if data.Icon == "" {
			data.Icon = "✓"
		}
	case "error", "failed":
		statusStyle = []StyleApplier{
			Foreground(PaletteDanger),
		}
		if data.Icon == "" {
			data.Icon = "✗"
		}
	case "warning":
		statusStyle = []StyleApplier{
			Foreground(PaletteWarning),
		}
		if data.Icon == "" {
			data.Icon = "⚠"
		}
	case "info":
		statusStyle = []StyleApplier{
			Foreground(PaletteInfo),
		}
		if data.Icon == "" {
			data.Icon = "ℹ"
		}
	}

	style.BorderStyle = Style(style.BorderStyle, statusStyle...)
	style.IconStyle = Style(style.IconStyle, statusStyle...)

	return NewCard(data).WithStyle(style)
}

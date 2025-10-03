package components

import (
	"fmt"
	"math"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

// Progress renders overall execution completion.
type Progress struct {
	bar   progress.Model
	total int
}

// NewProgress creates a progress component for the given total.
func NewProgress(total int) Progress {
	bar := progress.New(progress.WithDefaultGradient())
	bar.Width = 30
	return Progress{bar: bar, total: total}
}

// View renders the progress bar for the provided completion count.
func (p Progress) View(completed int) string {
	ratio := 0.0
	if p.total > 0 {
		ratio = math.Min(1.0, float64(completed)/float64(p.total))
	}
	label := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d/%d", completed, p.total))
	return lipgloss.JoinHorizontal(lipgloss.Left, label, " ", p.bar.ViewAs(ratio))
}

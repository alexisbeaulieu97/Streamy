package dashboard

import (
	"context"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	pipelineapp "github.com/alexisbeaulieu97/streamy/internal/app/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// Model is the main dashboard model
type Model struct {
	// Core data
	pipelines   []registry.Pipeline
	registry    *registry.Registry
	statusCache *registry.StatusCache
	service     *pipelineapp.Service

	// UI state
	viewMode     ViewMode
	cursor       int
	selectedID   string
	scrollOffset int

	// Component state
	spinner spinner.Model

	// Operation state
	loading       map[string]bool
	operations    map[string]Operation
	operationCtxs map[string]context.CancelFunc
	errors        map[string]string
	showError     bool
	errorMsg      string

	// Refresh state
	refreshing      bool
	refreshProgress int
	refreshTotal    int

	// Confirmation state
	confirmAction   string
	confirmPipeline string
	confirmMessage  string

	// Dimensions
	width  int
	height int

	// Configuration
	refreshInterval time.Duration
	confirmations   bool
	useUnicode      bool
}

// Operation tracks an in-progress async operation
type Operation struct {
	Type       string // "verify", "apply", "refresh"
	PipelineID string
	StartedAt  time.Time
}

// NewModel creates a new dashboard model
func NewModel(pipelines []registry.Pipeline, reg *registry.Registry, cache *registry.StatusCache, svc *pipelineapp.Service) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	m := Model{
		pipelines:       pipelines,
		registry:        reg,
		statusCache:     cache,
		service:         svc,
		viewMode:        ViewList,
		cursor:          0,
		loading:         make(map[string]bool),
		operations:      make(map[string]Operation),
		operationCtxs:   make(map[string]context.CancelFunc),
		errors:          make(map[string]string),
		spinner:         s,
		confirmations:   true,
		useUnicode:      true,
		width:           80,
		height:          24,
		refreshInterval: 0, // Disabled by default
	}

	// Load cached statuses
	for i := range m.pipelines {
		if cached, ok := cache.Get(m.pipelines[i].ID); ok {
			m.pipelines[i].Status = cached.Status
			m.pipelines[i].LastRun = cached.LastRun
		} else {
			m.pipelines[i].Status = registry.StatusUnknown
		}
	}

	// Sort pipelines by priority
	m.sortPipelines()

	return m
}

// Init initializes the model and returns initial commands
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.spinner.Tick,
	}

	// Load initial statuses if available
	if len(m.pipelines) > 0 {
		cmds = append(cmds, loadInitialStatusCmd(m.pipelines, m.statusCache))
	}

	return tea.Batch(cmds...)
}

// Helper Methods

// sortPipelines sorts pipelines by status priority: failed > drifted > satisfied > unknown
func (m *Model) sortPipelines() {
	sort.Slice(m.pipelines, func(i, j int) bool {
		return m.getStatusPriority(m.pipelines[i].Status) < m.getStatusPriority(m.pipelines[j].Status)
	})
}

// getStatusPriority returns sort priority for a status (lower = higher priority)
func (m *Model) getStatusPriority(status registry.PipelineStatus) int {
	switch status {
	case registry.StatusFailed:
		return 0
	case registry.StatusDrifted:
		return 1
	case registry.StatusSatisfied:
		return 2
	case registry.StatusVerifying, registry.StatusApplying:
		return 3
	default: // Unknown
		return 4
	}
}

// CountByStatus returns counts of pipelines in each status
func (m *Model) CountByStatus() map[registry.PipelineStatus]int {
	counts := make(map[registry.PipelineStatus]int)
	for _, p := range m.pipelines {
		counts[p.Status]++
	}
	return counts
}

// GetSelectedPipeline returns the currently selected pipeline
func (m *Model) GetSelectedPipeline() (registry.Pipeline, bool) {
	if m.cursor < 0 || m.cursor >= len(m.pipelines) {
		return registry.Pipeline{}, false
	}
	return m.pipelines[m.cursor], true
}

// GetPipelineByID returns a pipeline by its ID
func (m *Model) GetPipelineByID(id string) (registry.Pipeline, int, bool) {
	for i, p := range m.pipelines {
		if p.ID == id {
			return p, i, true
		}
	}
	return registry.Pipeline{}, -1, false
}

// UpdatePipelineStatus updates the status of a pipeline
func (m *Model) UpdatePipelineStatus(id string, status registry.PipelineStatus, lastRun time.Time) {
	for i := range m.pipelines {
		if m.pipelines[i].ID == id {
			m.pipelines[i].Status = status
			m.pipelines[i].LastRun = lastRun
			break
		}
	}
}

// MoveCursorUp moves cursor up with wrapping
func (m *Model) MoveCursorUp() {
	if len(m.pipelines) == 0 {
		return
	}
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.pipelines) - 1
	}
}

// MoveCursorDown moves cursor down with wrapping
func (m *Model) MoveCursorDown() {
	if len(m.pipelines) == 0 {
		return
	}
	m.cursor++
	if m.cursor >= len(m.pipelines) {
		m.cursor = 0
	}
}

// SetCursor sets cursor to specific index
func (m *Model) SetCursor(index int) {
	if index >= 0 && index < len(m.pipelines) {
		m.cursor = index
	}
}

// IsLoading checks if a pipeline has an operation in progress
func (m *Model) IsLoading(id string) bool {
	return m.loading[id]
}

// HasError checks if a pipeline has an error
func (m *Model) HasError(id string) bool {
	_, ok := m.errors[id]
	return ok
}

// GetError returns the error message for a pipeline
func (m *Model) GetError(id string) string {
	return m.errors[id]
}

// ClearError clears the error for a pipeline
func (m *Model) ClearError(id string) {
	delete(m.errors, id)
}

// GetViewMode returns the current view mode
func (m *Model) GetViewMode() ViewMode {
	return m.viewMode
}

// IsRefreshing returns whether a refresh-all is in progress
func (m *Model) IsRefreshing() bool {
	return m.refreshing
}

// GetRefreshTotal returns the total number of pipelines being refreshed
func (m *Model) GetRefreshTotal() int {
	return m.refreshTotal
}

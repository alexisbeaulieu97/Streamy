# Developer Quickstart: Interactive Dashboard

**Phase**: 1 (Design & Contracts)  
**Date**: October 8, 2025  
**Audience**: Developers implementing or extending the dashboard feature

## Overview

This guide helps you get started with implementing the interactive dashboard for Streamy. It covers setup, development workflow, testing, and common patterns.

## Prerequisites

- Go 1.25.1+ installed
- Familiarity with Streamy's codebase structure
- Basic understanding of Bubble Tea (Elm Architecture pattern)
- Terminal with ANSI color support for testing

## Project Structure Quick Reference

```
cmd/streamy/
├── dashboard.go         # NEW: Dashboard command entry point
├── root.go              # MODIFY: Route to dashboard when no subcommands
└── ...

internal/
├── tui/dashboard/       # NEW: Dashboard TUI implementation
│   ├── model.go         # Model state
│   ├── update.go        # Update logic (message handling)
│   ├── view.go          # View rendering (list view)
│   ├── detail.go        # Detail view rendering
│   ├── messages.go      # Custom message types
│   ├── commands.go      # Async command constructors
│   └── styles.go        # Lipgloss styling
├── registry/            # NEW: Registry abstraction
│   ├── registry.go      # Registry CRUD operations
│   ├── types.go         # Pipeline, RegistryEntry types
│   └── cache.go         # Status cache persistence
└── ...

specs/007-build-an-interactive/
├── data-model.md        # Data structures reference
├── contracts/
│   ├── messages.md      # Message types contract
│   ├── model.md         # Model interface contract
│   └── commands.md      # Command patterns
└── ...
```

## Development Workflow

### Step 1: Set Up Development Environment

```bash
# Clone and navigate to repo
cd /path/to/streamy

# Check out feature branch
git checkout 007-build-an-interactive

# Ensure dependencies are up to date
go mod tidy

# Run existing tests to ensure baseline
go test ./...
```

### Step 2: Create Registry Package First

The registry package is foundational - start here.

**File: `internal/registry/types.go`**

```go
package registry

import "time"

// Pipeline represents a registered Streamy pipeline
type Pipeline struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Path         string    `json:"path"`
    Description  string    `json:"description"`
    RegisteredAt time.Time `json:"registered_at"`
}

// (See data-model.md for complete type definitions)
```

**File: `internal/registry/registry.go`**

```go
package registry

import (
    "encoding/json"
    "os"
    "sync"
)

type Registry struct {
    path      string
    mu        sync.RWMutex
    pipelines []Pipeline
}

func NewRegistry(path string) (*Registry, error) {
    r := &Registry{path: path}
    if err := r.Load(); err != nil {
        return nil, err
    }
    return r, nil
}

// Implement Load, Save, List, Get, Add, Remove methods
// (See contracts/model.md for method signatures)
```

**Test: `internal/registry/registry_test.go`**

```go
func TestRegistryLoad(t *testing.T) {
    tmpfile := createTestRegistry(t) // Helper to create temp JSON
    defer os.Remove(tmpfile)
    
    r, err := NewRegistry(tmpfile)
    assert.NoError(t, err)
    assert.Len(t, r.List(), 2) // Assuming test fixture has 2 pipelines
}
```

### Step 3: Create Dashboard TUI Skeleton

**File: `internal/tui/dashboard/model.go`**

```go
package dashboard

import (
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/spinner"
    tea "github.com/charmbracelet/bubbletea"
    
    "github.com/alexisbeaulieu97/streamy/internal/registry"
)

type Model struct {
    pipelines   []registry.Pipeline
    registry    *registry.Registry
    statusCache *registry.StatusCache
    
    viewMode    ViewMode
    cursor      int
    selectedID  string
    
    list        list.Model
    spinner     spinner.Model
    
    loading     map[string]bool
    errors      map[string]string
    
    width       int
    height      int
}

type ViewMode int

const (
    ViewList ViewMode = iota
    ViewDetail
    ViewHelp
    ViewConfirm
)

func NewModel(pipelines []registry.Pipeline, reg *registry.Registry, cache *registry.StatusCache) Model {
    // Initialize model
    m := Model{
        pipelines:   pipelines,
        registry:    reg,
        statusCache: cache,
        viewMode:    ViewList,
        cursor:      0,
        loading:     make(map[string]bool),
        errors:      make(map[string]string),
    }
    
    // Initialize Bubbles components
    m.list = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
    m.spinner = spinner.New()
    
    return m
}

// Implement tea.Model interface
func (m Model) Init() tea.Cmd {
    return loadInitialStatusCmd(m.pipelines)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // TODO: Implement message handling
    return m, nil
}

func (m Model) View() string {
    // TODO: Implement view rendering
    return "Dashboard coming soon..."
}
```

### Step 4: Implement Message Types

**File: `internal/tui/dashboard/messages.go`**

```go
package dashboard

import (
    "time"
    "github.com/alexisbeaulieu97/streamy/internal/model"
)

// Navigation messages
type PipelineSelectedMsg struct {
    Pipeline registry.Pipeline
}

type BackToListMsg struct{}

// Operation messages
type VerifyStartedMsg struct {
    PipelineID string
}

type VerifyCompleteMsg struct {
    PipelineID string
    Result     model.ExecutionResult
}

// (See contracts/messages.md for complete list)
```

### Step 5: Implement Update Logic

**File: `internal/tui/dashboard/update.go`**

```go
package dashboard

import tea "github.com/charmbracelet/bubbletea"

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    
    // System messages
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.list.SetSize(msg.Width, msg.Height-4) // Adjust for header/footer
        return m, nil
    
    // Keyboard input
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    
    // Navigation messages
    case PipelineSelectedMsg:
        m.selectedID = msg.Pipeline.ID
        m.viewMode = ViewDetail
        return m, loadDetailCmd(msg.Pipeline.ID)
    
    case BackToListMsg:
        m.viewMode = ViewList
        m.selectedID = ""
        return m, nil
    
    // Operation messages
    case VerifyCompleteMsg:
        return m.handleVerifyComplete(msg)
    
    // (See contracts/messages.md for all message handlers)
    
    default:
        return m, nil
    }
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // View-specific key handling
    switch m.viewMode {
    case ViewList:
        return m.handleListKeys(msg)
    case ViewDetail:
        return m.handleDetailKeys(msg)
    case ViewHelp:
        return m.handleHelpKeys(msg)
    case ViewConfirm:
        return m.handleConfirmKeys(msg)
    }
    return m, nil
}

func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "up", "k":
        if m.cursor > 0 {
            m.cursor--
        }
        return m, nil
    
    case "down", "j":
        if m.cursor < len(m.pipelines)-1 {
            m.cursor++
        }
        return m, nil
    
    case "enter":
        if len(m.pipelines) > 0 {
            return m, func() tea.Msg {
                return PipelineSelectedMsg{Pipeline: m.pipelines[m.cursor]}
            }
        }
        return m, nil
    
    case "r":
        return m, refreshAllCmd(m.pipelines)
    
    case "q", "ctrl+c":
        return m, tea.Quit
    
    case "?":
        m.viewMode = ViewHelp
        return m, nil
    }
    
    return m, nil
}
```

### Step 6: Implement View Rendering

**File: `internal/tui/dashboard/view.go`**

```go
package dashboard

import (
    "fmt"
    "strings"
    "github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
    switch m.viewMode {
    case ViewList:
        return m.renderListView()
    case ViewDetail:
        return m.renderDetailView()
    case ViewHelp:
        return m.renderHelpView()
    case ViewConfirm:
        return m.renderConfirmView()
    }
    return ""
}

func (m Model) renderListView() string {
    // Header
    header := m.renderHeader()
    
    // Body (pipeline list)
    body := m.renderPipelineList()
    
    // Footer (key hints)
    footer := m.renderFooter()
    
    // Combine with lipgloss
    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        body,
        footer,
    )
}

func (m Model) renderHeader() string {
    title := titleStyle.Render("Streamy Dashboard")
    
    // Count pipelines by status
    satisfied, drifted, failed, unknown := m.countByStatus()
    summary := fmt.Sprintf(
        "🟢 %d  🟡 %d  🔴 %d  ⚪ %d  Total: %d",
        satisfied, drifted, failed, unknown, len(m.pipelines),
    )
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        title,
        summaryStyle.Render(summary),
        strings.Repeat("─", m.width),
    )
}

func (m Model) renderPipelineList() string {
    if len(m.pipelines) == 0 {
        return emptyStateStyle.Render("No pipelines registered. Run 'streamy register' to add one.")
    }
    
    var items []string
    for i, p := range m.pipelines {
        items = append(items, m.renderPipelineItem(p, i == m.cursor))
    }
    
    return strings.Join(items, "\n")
}

func (m Model) renderPipelineItem(p registry.Pipeline, selected bool) string {
    // Status icon
    status := m.getPipelineStatus(p.ID)
    icon := status.Icon()
    
    // Loading indicator
    if m.loading[p.ID] {
        icon = m.spinner.View()
    }
    
    // Format line
    line := fmt.Sprintf("%s %s - %s", icon, p.Name, p.Description)
    
    // Apply style
    style := itemStyle
    if selected {
        style = selectedItemStyle
    }
    
    return style.Render(line)
}

func (m Model) renderFooter() string {
    hints := "↑↓: navigate  ↵: select  r: refresh  ?: help  q: quit"
    return footerStyle.Render(hints)
}
```

**File: `internal/tui/dashboard/styles.go`**

```go
package dashboard

import "github.com/charmbracelet/lipgloss"

var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("205")).
        MarginTop(1).
        MarginBottom(1)
    
    summaryStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("240"))
    
    itemStyle = lipgloss.NewStyle().
        PaddingLeft(2)
    
    selectedItemStyle = itemStyle.Copy().
        Background(lipgloss.Color("240")).
        Foreground(lipgloss.Color("15"))
    
    emptyStateStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        Italic(true).
        MarginTop(2).
        MarginLeft(2)
    
    footerStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        BorderTop(true).
        BorderStyle(lipgloss.NormalBorder()).
        MarginTop(1)
)
```

### Step 7: Implement Commands

**File: `internal/tui/dashboard/commands.go`**

```go
package dashboard

import (
    "context"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/alexisbeaulieu97/streamy/internal/engine"
)

func verifyCmd(pipelineID string, configPath string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        result, err := engine.VerifyPipeline(ctx, configPath)
        
        if err != nil {
            return VerifyErrorMsg{
                PipelineID: pipelineID,
                Error:      convertError(err),
            }
        }
        
        return VerifyCompleteMsg{
            PipelineID: pipelineID,
            Result:     convertResult(result),
        }
    }
}

// (See contracts/commands.md for all command implementations)
```

### Step 8: Wire Up CLI Command

**File: `cmd/streamy/dashboard.go`**

```go
package main

import (
    "fmt"
    "os"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"
    
    "github.com/alexisbeaulieu97/streamy/internal/registry"
    "github.com/alexisbeaulieu97/streamy/internal/tui/dashboard"
)

func newDashboardCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "dashboard",
        Short: "Launch interactive dashboard",
        RunE:  runDashboard,
    }
}

func runDashboard(cmd *cobra.Command, args []string) error {
    // Load registry
    regPath := registry.DefaultPath() // ~/.streamy/registry.json
    reg, err := registry.NewRegistry(regPath)
    if err != nil {
        return fmt.Errorf("failed to load registry: %w", err)
    }
    
    // Load status cache
    cachePath := registry.DefaultCachePath() // ~/.streamy/status-cache.json
    cache, err := registry.NewStatusCache(cachePath)
    if err != nil {
        // Cache is optional, proceed with empty cache
        cache = registry.NewEmptyStatusCache(cachePath)
    }
    
    // Create dashboard model
    pipelines := reg.List()
    model := dashboard.NewModel(pipelines, reg, cache)
    
    // Run Bubble Tea program
    p := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        return fmt.Errorf("dashboard error: %w", err)
    }
    
    return nil
}
```

**File: `cmd/streamy/root.go` (modify)**

```go
func Execute() {
    rootCmd := &cobra.Command{
        Use:   "streamy",
        Short: "Streamy - Declarative environment management",
        RunE: func(cmd *cobra.Command, args []string) error {
            // If no subcommands, launch dashboard
            if len(args) == 0 {
                return runDashboard(cmd, args)
            }
            return cmd.Help()
        },
    }
    
    // Add subcommands
    rootCmd.AddCommand(newApplyCmd())
    rootCmd.AddCommand(newVerifyCmd())
    rootCmd.AddCommand(newDashboardCmd()) // Explicit dashboard command
    // ...
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## Testing Guide

### Unit Tests

```bash
# Test registry package
go test ./internal/registry/... -v

# Test dashboard model
go test ./internal/tui/dashboard/... -v

# Run with coverage
go test ./internal/tui/dashboard/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

**File: `tests/integration_dashboard_test.go`**

```go
package tests

import (
    "testing"
    "github.com/stretchr/testify/assert"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/alexisbeaulieu97/streamy/internal/tui/dashboard"
)

func TestDashboardNavigationFlow(t *testing.T) {
    // Setup test registry and cache
    testPipelines := createTestPipelines(t)
    testReg := createTestRegistry(t, testPipelines)
    testCache := createTestCache(t)
    
    // Create model
    m := dashboard.NewModel(testPipelines, testReg, testCache)
    
    // Simulate navigation down
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
    assert.Equal(t, 1, m.GetCursor())
    
    // Simulate selection
    m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
    assert.NotNil(t, cmd)
    
    // Execute command (simulate message)
    msg := cmd()
    selectedMsg, ok := msg.(dashboard.PipelineSelectedMsg)
    assert.True(t, ok)
    assert.Equal(t, testPipelines[1].ID, selectedMsg.Pipeline.ID)
}
```

### Manual Testing

```bash
# Build binary
go build -o streamy ./cmd/streamy

# Run dashboard
./streamy

# Test with sample pipelines
./streamy register --path testdata/configs/dev.yaml --name "Dev Environment"
./streamy register --path testdata/configs/prod.yaml --name "Production"
./streamy  # Launch dashboard, should show 2 pipelines
```

## Common Patterns

### Pattern 1: Adding a New Message Type

1. Define message in `messages.go`:
   ```go
   type NewActionMsg struct {
       Data string
   }
   ```

2. Add handler in `update.go`:
   ```go
   case NewActionMsg:
       // Handle message
       return m, cmdIfNeeded
   ```

3. Add test:
   ```go
   func TestNewActionMsg(t *testing.T) {
       m := NewModel(...)
       m, _ = m.Update(NewActionMsg{Data: "test"})
       // Assert state changed correctly
   }
   ```

### Pattern 2: Adding a New View Mode

1. Add to `ViewMode` enum:
   ```go
   const (
       ViewList ViewMode = iota
       ViewDetail
       ViewHelp
       ViewConfirm
       ViewNewMode  // NEW
   )
   ```

2. Implement render function:
   ```go
   func (m Model) renderNewModeView() string {
       // Render view
   }
   ```

3. Add to `View()` dispatcher:
   ```go
   case ViewNewMode:
       return m.renderNewModeView()
   ```

4. Add transition logic in `Update()`:
   ```go
   case SomeMsg:
       m.viewMode = ViewNewMode
       return m, nil
   ```

### Pattern 3: Adding Async Operation

1. Define messages:
   ```go
   type OperationStartedMsg struct { ... }
   type OperationCompleteMsg struct { ... }
   type OperationErrorMsg struct { ... }
   ```

2. Create command:
   ```go
   func operationCmd(params) tea.Cmd {
       return func() tea.Msg {
           result, err := doWork(params)
           if err != nil {
               return OperationErrorMsg{...}
           }
           return OperationCompleteMsg{...}
       }
   }
   ```

3. Trigger from Update():
   ```go
   case KeyMsg:
       if msg.String() == "o" { // Trigger operation
           return m, operationCmd(params)
       }
   ```

4. Handle completion:
   ```go
   case OperationCompleteMsg:
       // Update model with result
       return m, nil
   ```

## Debugging Tips

### Enable Debug Logging

```go
import "github.com/rs/zerolog/log"

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    log.Debug().
        Str("type", fmt.Sprintf("%T", msg)).
        Interface("msg", msg).
        Msg("Update received message")
    
    // ... handle message
}
```

### View Debug Output

```bash
# Run with debug logging to file
./streamy 2> debug.log

# In another terminal, tail the log
tail -f debug.log
```

### Inspect Model State

```go
func (m Model) DebugString() string {
    return fmt.Sprintf(
        "ViewMode: %d, Cursor: %d, Loading: %v, Pipelines: %d",
        m.viewMode, m.cursor, m.loading, len(m.pipelines),
    )
}

// In View()
if debugMode {
    return m.DebugString()
}
```

## Performance Optimization Checklist

- [ ] Cache static strings (header, footer)
- [ ] Only re-render changed items in list
- [ ] Use `lipgloss.Inline(true)` for non-layout styles
- [ ] Avoid regexp in View() (pre-compile if needed)
- [ ] Profile with `go test -bench -cpuprofile`
- [ ] Check memory allocations with `-benchmem`
- [ ] Test with 100+ pipelines for performance validation

## Next Steps

1. **Implement Core**: Complete model, update, view, and commands
2. **Add Tests**: Unit tests for all message handlers
3. **Integration**: Wire up to CLI, test with real configs
4. **Detail View**: Implement pipeline detail view (FR-006)
5. **Polish**: Add help overlay, error handling, confirmation dialogs
6. **Document**: Update main README with dashboard usage

## Resources

- **Bubble Tea Tutorial**: https://github.com/charmbracelet/bubbletea/tree/master/tutorials
- **Lipgloss Examples**: https://github.com/charmbracelet/lipgloss/tree/master/examples
- **Bubbles Components**: https://github.com/charmbracelet/bubbles
- **Feature Spec**: `specs/007-build-an-interactive/spec.md`
- **Data Model**: `specs/007-build-an-interactive/data-model.md`
- **Contracts**: `specs/007-build-an-interactive/contracts/`

## Getting Help

- Check existing TUI code in `internal/tui/` for patterns
- Review Bubble Tea examples for inspiration
- Refer to contracts for interface definitions
- Ask in team chat or create GitHub issue for blockers

Happy coding! 🚀

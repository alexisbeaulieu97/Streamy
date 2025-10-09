# Quickstart: Registry Management CLI

**Phase**: 1 - Design & Contracts  
**Date**: October 9, 2025  
**Audience**: Developers implementing this feature

## Overview

This quickstart guide provides implementation guidance for adding registry management CLI commands to Streamy. It focuses on the critical path to MVP, leveraging existing infrastructure, and avoiding common pitfalls.

## Terminology Glossary

**Key Terms**: To maintain consistency across implementation:

- **Pipeline ID**: The unique identifier for a pipeline in the registry (e.g., "my-project"). This is the primary key used for all registry operations (list, unregister, refresh, show). Generated automatically from the filename or provided explicitly via `--id` flag.

- **Pipeline Name**: A human-friendly descriptive label for a pipeline (e.g., "Production Environment"). This is metadata stored in the registry but NOT used as a lookup key. Set via `--name` flag during registration.

- **Refresh Command**: The user-facing CLI command (`streamy registry refresh`) that updates pipeline statuses by running verification checks.

- **Verify Operation**: The internal engine operation (`engine.Verify()`) that checks if a pipeline's desired state matches actual state. The refresh command invokes this internally for each pipeline.

- **Registry**: Persistent storage file (`~/.streamy/registry.json`) containing all registered pipeline metadata.

- **Status Cache**: Runtime state file (`~/.streamy/status.json`) containing current verification results (satisfied, drifted, failed, unknown).

## Prerequisites

- Familiarity with Go 1.25+ and Cobra CLI framework
- Understanding of existing Streamy codebase structure
- Read: `data-model.md`, `research.md`, and `contracts/registry-cli.md`

## Implementation Checklist

### Phase 1: Core Infrastructure (Day 1)

- [ ] **1.1**: Create `internal/registry/helpers.go`
  - `GeneratePipelineID(path string) string`
  - `ValidatePipelineID(id string) error`
  - `SanitizeFilename(name string) string`
  
- [ ] **1.2**: Create `cmd/streamy/registry.go` (parent command)
  - Group command: `streamy registry <subcommand>`
  - Help text and subcommand registration
  
- [ ] **1.3**: Test ID generation logic
  - Unit tests for `GeneratePipelineID`
  - Edge cases: special characters, long names, empty strings

### Phase 2: Register Command (Day 1-2)

- [ ] **2.1**: Create `cmd/streamy/register.go`
  - Command struct with flags (id, name, description)
  - Argument validation (exactly 1 path)
  
- [ ] **2.2**: Implement validation logic
  - File existence check
  - Config parsing via `config.ParseAndValidate`
  - ID uniqueness check
  
- [ ] **2.3**: Implement registration flow
  - Convert path to absolute
  - Create Pipeline struct
  - Call `Registry.Add()` and `Registry.Save()`
  
- [ ] **2.4**: Add error handling
  - Structured errors with suggestions
  - Log verbosity controlled by `--verbose` flag
  
- [ ] **2.5**: Create `cmd/streamy/register_test.go`
  - Test with valid config
  - Test duplicate ID error
  - Test invalid config error
  - Test ID generation

### Phase 3: List Command (Day 2)

- [ ] **3.1**: Create `cmd/streamy/list.go`
  - Command struct with flags (format, json)
  - No arguments validation
  
- [ ] **3.2**: Implement table output
  - Use `text/tabwriter` for alignment
  - Unicode icons with ASCII fallback
  - Relative timestamps ("2 hours ago")
  
- [ ] **3.3**: Implement JSON output
  - Marshal registry + status into JSON
  - Pretty-print with 2-space indent
  - Include version and count fields
  
- [ ] **3.4**: Handle empty registry
  - Friendly message with hint to register
  
- [ ] **3.5**: Create `cmd/streamy/list_test.go`
  - Test table format with multiple pipelines
  - Test JSON format validation
  - Test empty registry output

### Phase 4: Unregister Command (Day 3)

- [ ] **4.1**: Create `cmd/streamy/unregister.go`
  - Command struct with flags (force)
  - Argument validation (exactly 1 ID)
  
- [ ] **4.2**: Implement confirmation prompt
  - Use `bufio.Scanner` for stdin
  - Skip if `--force` flag set
  - Handle non-TTY (require --force)
  
- [ ] **4.3**: Implement removal flow
  - Call `Registry.Get()` to verify exists
  - Call `Registry.Remove()` and `Registry.Save()`
  - Optionally clean status cache
  
- [ ] **4.4**: Create `cmd/streamy/unregister_test.go`
  - Test successful removal
  - Test not-found error
  - Test confirmation logic

### Phase 5: Refresh Command (Day 3-4)

- [ ] **5.1**: Create `cmd/streamy/refresh.go`
  - Command struct with flags (concurrency)
  - Optional pipeline-id argument
  
- [ ] **5.2**: Implement single pipeline refresh
  - Load config and execute verify
  - Update status cache with result
  - Print result summary
  
- [ ] **5.3**: Implement bulk refresh
  - Worker pool with semaphore
  - Progress indicators during execution
  - Aggregate results and print summary
  
- [ ] **5.4**: Handle missing config files gracefully
  - Mark status as failed with descriptive error
  - Don't halt entire refresh on one failure
  
- [ ] **5.5**: Create `cmd/streamy/refresh_test.go`
  - Test single pipeline refresh
  - Test bulk refresh with concurrency
  - Test missing config handling

### Phase 6: Integration (Day 4)

- [ ] **6.1**: Register commands in `root.go`
  - Add registry command group
  - Maintain existing command structure
  
- [ ] **6.2**: Create `tests/integration_registry_test.go`
  - End-to-end test: register → list → refresh → unregister
  - Verify file system state changes
  - Test with real config files
  
- [ ] **6.3**: Manual testing
  - Test on Linux, macOS, and Windows
  - Verify dashboard updates after register/unregister
  - Test with 10+ pipelines for performance
  
- [ ] **6.4**: Update documentation
  - Add command help text
  - Update README with registry commands
  - Add examples to docs/

## Implementation Patterns

### Pattern 1: Command Boilerplate

```go
// cmd/streamy/register.go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    
    "github.com/spf13/cobra"
    "streamy/internal/config"
    "streamy/internal/registry"
)

type registerFlags struct {
    id          string
    name        string
    description string
}

func newRegisterCmd(rootFlags *rootFlags) *cobra.Command {
    flags := &registerFlags{}
    
    cmd := &cobra.Command{
        Use:   "register <config-path>",
        Short: "Register a new pipeline in the registry",
        Long:  `Register a Streamy configuration file...`,
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runRegister(args[0], flags, rootFlags)
        },
    }
    
    cmd.Flags().StringVarP(&flags.id, "id", "i", "", "Pipeline ID (auto-generated if omitted)")
    cmd.Flags().StringVarP(&flags.name, "name", "n", "", "Pipeline name (filename if omitted)")
    cmd.Flags().StringVarP(&flags.description, "description", "d", "", "Pipeline description")
    
    return cmd
}

func runRegister(configPath string, flags *registerFlags, rootFlags *rootFlags) error {
    // Implementation here
    return nil
}
```

### Pattern 2: Path Validation

```go
func validateAndNormalizePath(path string) (string, error) {
    // Expand tilde
    if strings.HasPrefix(path, "~") {
        home, err := os.UserHomeDir()
        if err != nil {
            return "", fmt.Errorf("cannot expand ~: %w", err)
        }
        path = filepath.Join(home, path[1:])
    }
    
    // Convert to absolute
    absPath, err := filepath.Abs(path)
    if err != nil {
        return "", fmt.Errorf("cannot resolve absolute path: %w", err)
    }
    
    // Check existence
    if _, err := os.Stat(absPath); err != nil {
        return "", fmt.Errorf("file not found: %w", err)
    }
    
    return absPath, nil
}
```

### Pattern 3: Registry Initialization

```go
func loadRegistry() (*registry.Registry, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("cannot determine home directory: %w", err)
    }
    
    registryPath := filepath.Join(home, ".streamy", "registry.json")
    
    reg, err := registry.NewRegistry(registryPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load registry: %w", err)
    }
    
    return reg, nil
}
```

### Pattern 4: Table Output with Tabwriter

```go
import "text/tabwriter"

func printTableOutput(pipelines []registry.Pipeline, statuses map[string]registry.CachedStatus) {
    w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
    defer w.Flush()
    
    // Header
    fmt.Fprintln(w, "ID\tNAME\tSTATUS\tLAST RUN\tPATH")
    
    // Rows
    for _, p := range pipelines {
        status := statuses[p.ID]
        statusStr := formatStatus(status.Status)
        lastRun := formatTimestamp(status.LastRun)
        
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
            p.ID, p.Name, statusStr, lastRun, p.Path)
    }
}

func formatStatus(status registry.PipelineStatus) string {
    if isUnicodeSupported() {
        return status.Icon() + " " + string(status)
    }
    return status.IconFallback() + " " + string(status)
}

func formatTimestamp(t time.Time) string {
    if t.IsZero() {
        return "never"
    }
    
    now := time.Now()
    diff := now.Sub(t)
    
    if diff < time.Hour {
        mins := int(diff.Minutes())
        return fmt.Sprintf("%d minutes ago", mins)
    } else if diff < 24*time.Hour {
        hours := int(diff.Hours())
        return fmt.Sprintf("%d hours ago", hours)
    } else {
        days := int(diff.Hours() / 24)
        return fmt.Sprintf("%d days ago", days)
    }
}
```

### Pattern 5: Confirmation Prompt

```go
import "bufio"

func confirmAction(prompt string, force bool) bool {
    if force {
        return true
    }
    
    // Check if stdin is a terminal
    if !isatty.IsTerminal(os.Stdin.Fd()) {
        fmt.Fprintln(os.Stderr, "Error: cannot prompt for confirmation: not a terminal")
        fmt.Fprintln(os.Stderr, "Suggestion: Use --force flag in non-interactive environments")
        return false
    }
    
    fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
    
    scanner := bufio.NewScanner(os.Stdin)
    if !scanner.Scan() {
        return false
    }
    
    response := strings.ToLower(strings.TrimSpace(scanner.Text()))
    return response == "y" || response == "yes"
}
```

### Pattern 6: Concurrent Refresh

```go
func refreshAllPipelines(pipelines []registry.Pipeline, concurrency int) []RefreshResult {
    results := make([]RefreshResult, len(pipelines))
    semaphore := make(chan struct{}, concurrency)
    var wg sync.WaitGroup
    var mu sync.Mutex // For progress output
    
    for i, pipeline := range pipelines {
        wg.Add(1)
        go func(idx int, p registry.Pipeline) {
            defer wg.Done()
            
            semaphore <- struct{}{}        // Acquire
            defer func() { <-semaphore }() // Release
            
            // Print progress (thread-safe)
            mu.Lock()
            fmt.Printf("[%d/%d] %s... ", idx+1, len(pipelines), p.ID)
            mu.Unlock()
            
            result := verifyPipeline(p)
            results[idx] = result
            
            // Print result (thread-safe)
            mu.Lock()
            fmt.Printf("%s\n", formatRefreshResult(result))
            mu.Unlock()
        }(i, pipeline)
    }
    
    wg.Wait()
    return results
}
```

## Common Pitfalls

### ❌ Pitfall 1: Forgetting to Save Registry

```go
// WRONG: Changes not persisted
reg.Add(pipeline)
// Missing: reg.Save()
```

```go
// CORRECT: Always save after modifications
reg.Add(pipeline)
if err := reg.Save(); err != nil {
    return fmt.Errorf("failed to save registry: %w", err)
}
```

### ❌ Pitfall 2: Not Converting Relative Paths

```go
// WRONG: Stores relative path (breaks when CWD changes)
pipeline.Path = userProvidedPath
```

```go
// CORRECT: Always use absolute paths
absPath, err := filepath.Abs(userProvidedPath)
if err != nil {
    return err
}
pipeline.Path = absPath
```

### ❌ Pitfall 3: Ignoring Non-TTY Environments

```go
// WRONG: Hangs in CI/CD pipelines
fmt.Print("Confirm? [y/N]: ")
var response string
fmt.Scanln(&response)
```

```go
// CORRECT: Detect TTY and require --force
if !isatty.IsTerminal(os.Stdin.Fd()) && !force {
    return fmt.Errorf("cannot prompt: not a terminal (use --force)")
}
```

### ❌ Pitfall 4: Race Conditions in Concurrent Refresh

```go
// WRONG: Concurrent writes to shared map
for _, p := range pipelines {
    go func(pipeline registry.Pipeline) {
        status := verify(pipeline)
        statusCache[pipeline.ID] = status // RACE!
    }(p)
}
```

```go
// CORRECT: Use mutex or preallocated slice
results := make([]RefreshResult, len(pipelines))
for i, p := range pipelines {
    go func(idx int, pipeline registry.Pipeline) {
        results[idx] = verify(pipeline) // No race (unique index)
    }(i, p)
}
```

### ❌ Pitfall 5: Not Handling Missing Config Files

```go
// WRONG: Panic on missing file during refresh
cfg, err := config.ParseAndValidate(pipeline.Path)
if err != nil {
    panic(err) // Crashes entire refresh
}
```

```go
// CORRECT: Mark as failed and continue
cfg, err := config.ParseAndValidate(pipeline.Path)
if err != nil {
    return RefreshResult{
        PipelineID: pipeline.ID,
        Status:     registry.StatusFailed,
        Error:      fmt.Errorf("config not found: %w", err),
    }
}
```

## Testing Strategy

### Unit Tests

**Location**: `cmd/streamy/*_test.go`

**What to test**:
- Argument validation
- Flag parsing
- ID generation and validation
- Path normalization
- Error message formatting

**Example**:
```go
func TestGeneratePipelineID(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"/path/to/Dev Setup.yaml", "dev-setup"},
        {"/path/to/my_config.yaml", "my-config"},
        {"special!@#chars.yaml", "special-chars"},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            result := GeneratePipelineID(tt.input)
            if result != tt.expected {
                t.Errorf("got %s, want %s", result, tt.expected)
            }
        })
    }
}
```

### Integration Tests

**Location**: `tests/integration_registry_test.go`

**What to test**:
- Full command execution (register → list → refresh → unregister)
- File system state verification
- Registry JSON content validation
- Status cache updates

**Setup pattern**:
```go
func TestRegistryCommands(t *testing.T) {
    // Setup: Create temporary directory
    tmpDir := t.TempDir()
    os.Setenv("HOME", tmpDir) // Override home for testing
    
    // Create test config file
    configPath := filepath.Join(tmpDir, "test-config.yaml")
    createTestConfig(t, configPath)
    
    // Test: Register
    err := runRegisterCmd(configPath, "--id", "test-pipeline")
    require.NoError(t, err)
    
    // Verify: Registry file exists
    registryPath := filepath.Join(tmpDir, ".streamy", "registry.json")
    require.FileExists(t, registryPath)
    
    // Verify: Pipeline in registry
    reg, err := registry.NewRegistry(registryPath)
    require.NoError(t, err)
    pipelines := reg.List()
    require.Len(t, pipelines, 1)
    require.Equal(t, "test-pipeline", pipelines[0].ID)
    
    // Test: List
    output := captureOutput(func() {
        runListCmd()
    })
    require.Contains(t, output, "test-pipeline")
    
    // Test: Unregister
    err = runUnregisterCmd("test-pipeline", "--force")
    require.NoError(t, err)
    
    // Verify: Registry empty
    reg.Load()
    pipelines = reg.List()
    require.Len(t, pipelines, 0)
}
```

## Performance Targets

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Register single pipeline | <100ms | Time from command start to completion |
| List 50 pipelines | <1s | Time to load and display table |
| Refresh 10 pipelines (concurrent) | <30s | Total time for batch verification |
| Unregister single pipeline | <50ms | Time to remove and save |

## Success Criteria

✅ **MVP Complete** when:
- All 4 commands (register, unregister, list, refresh) implemented
- Unit tests pass with >80% coverage
- Integration tests verify end-to-end workflows
- Dashboard automatically updates after register/unregister
- Error messages include actionable suggestions
- Commands work on Linux, macOS, and Windows

## Next Steps After MVP

1. **Show command** (P3 priority): Detailed pipeline inspection
2. **List filters**: `--status`, `--path-contains` flags
3. **Pagination**: `--limit`, `--offset` for large registries
4. **Export/Import**: Backup and restore registry
5. **Aliases**: Shorter command names (e.g., `reg`, `unreg`, `ls`)

## Resources

- **Existing commands**: `cmd/streamy/apply.go`, `cmd/streamy/verify.go`
- **Registry package**: `internal/registry/registry.go`
- **Data model**: `specs/008-extend-streamy-with/data-model.md`
- **Contracts**: `specs/008-extend-streamy-with/contracts/registry-cli.md`
- **Cobra docs**: https://github.com/spf13/cobra/blob/main/user_guide.md

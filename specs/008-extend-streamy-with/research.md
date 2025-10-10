# Research: Registry Management CLI Commands

**Phase**: 0 - Research & Technical Discovery  
**Date**: October 9, 2025  
**Status**: Complete

## Overview

This document consolidates technical research for implementing registry management CLI commands. The primary goal is to leverage existing infrastructure while identifying patterns and best practices for CLI command implementation in Go using Cobra.

## Research Tasks Completed

### 1. Existing Registry Infrastructure Analysis

**Decision**: Use existing `internal/registry` package without modifications  
**Rationale**: The package already provides all required functionality:
  - Thread-safe Registry type with Add/Remove/List/Get/Update methods
  - Atomic file writes via temp-file-then-rename pattern
  - JSON serialization with schema versioning
  - StatusCache for runtime state separate from persistent registry
  - Pipeline and PipelineStatus types fully defined

**Findings**:
```go
// Existing capabilities we can leverage directly:
- Registry.Add(Pipeline) error           // For register command
- Registry.Remove(id string) error       // For unregister command  
- Registry.List() []Pipeline             // For list command
- Registry.Get(id string) (Pipeline, error)  // For show command
- Registry.Save() error                  // Atomic persistence
- StatusCache.Get/Set/Save()             // For refresh command
```

**Alternatives Considered**:
  - Creating new registry package: Rejected because existing implementation is production-ready with proper concurrency control and atomic writes
  - Modifying registry schema: Rejected because current schema satisfies all requirements in spec

### 2. Cobra Command Patterns in Streamy

**Decision**: Follow existing command structure from `apply.go` and `verify.go`  
**Rationale**: Consistency with established patterns reduces cognitive load and maintenance burden

**Pattern Identified**:
```go
// Standard command structure used in codebase:
func newCommandCmd(rootFlags *rootFlags) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "command [args]",
        Short: "Brief description",
        Long:  "Detailed description with examples",
        Args:  cobra.ExactArgs(1), // or cobra.NoArgs, etc.
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. Validate arguments
            // 2. Load/initialize dependencies
            // 3. Execute operation
            // 4. Handle errors with context
            // 5. Display results
            return nil
        },
    }
    
    // Add command-specific flags
    cmd.Flags().StringVar(&variable, "flag-name", defaultValue, "description")
    
    return cmd
}
```

**Best Practices from Existing Code**:
  - Use `RunE` instead of `Run` to return errors properly
  - Leverage `rootFlags` for global flags (verbose, dry-run)
  - Set `SilenceUsage: true` on root to prevent help spam on errors
  - Use `fmt.Fprintf(os.Stderr, ...)` for error messages
  - Use structured output for machine-readable formats

### 3. Pipeline ID Generation Strategy

**Decision**: Use sanitized filename as default ID, allow explicit ID override via flag  
**Rationale**: Balances user convenience with predictability and control

**Implementation Approach**:
```go
// Algorithm for ID generation:
func GeneratePipelineID(configPath string) string {
    // 1. Extract filename without extension
    filename := filepath.Base(configPath)
    filename = strings.TrimSuffix(filename, filepath.Ext(filename))
    
    // 2. Sanitize: lowercase, replace non-alphanumeric with hyphens
    id := strings.ToLower(filename)
    id = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(id, "-")
    id = strings.Trim(id, "-")
    
    // 3. Ensure minimum length (e.g., at least 1 char)
    if id == "" {
        id = "pipeline-" + randomString(8)
    }
    
    return id
}
```

**Validation Rules**:
  - ID must match: `^[a-z0-9][a-z0-9-]*[a-z0-9]$` (alphanumeric + hyphens, no leading/trailing hyphens)
  - Length: 1-64 characters
  - Uniqueness: Enforced by Registry.Add duplicate check

**Alternatives Considered**:
  - UUID generation: Rejected because not human-friendly for CLI usage
  - Hash of file path: Rejected because not stable across file moves
  - User-required ID: Rejected because adds friction to common case

### 4. Configuration File Validation

**Decision**: Use existing `config.ParseAndValidate()` from `internal/config`  
**Rationale**: Reuse battle-tested validation logic already used by apply/verify commands

**Validation Flow**:
```go
// Validation steps for register command:
func validateConfigFile(path string) error {
    // 1. Check file exists
    if _, err := os.Stat(path); err != nil {
        return fmt.Errorf("config file not found: %w", err)
    }
    
    // 2. Parse YAML and validate schema
    cfg, err := config.ParseAndValidate(path)
    if err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }
    
    // 3. Ensure config has steps (non-empty)
    if len(cfg.Steps) == 0 {
        return errors.New("config must contain at least one step")
    }
    
    return nil
}
```

**Error Handling**: Provide actionable error messages with file/line context (already implemented in config parser)

### 5. Concurrent Refresh Implementation

**Decision**: Use existing verify engine with goroutine pool for parallel execution  
**Rationale**: Existing `internal/engine` already implements concurrent DAG execution with proper error handling

**Implementation Pattern**:
```go
// Refresh command uses worker pool pattern:
func refreshPipelines(pipelines []registry.Pipeline, concurrency int) []RefreshResult {
    results := make([]RefreshResult, len(pipelines))
    semaphore := make(chan struct{}, concurrency) // Limit concurrent verifications
    var wg sync.WaitGroup
    
    for i, pipeline := range pipelines {
        wg.Add(1)
        go func(idx int, p registry.Pipeline) {
            defer wg.Done()
            semaphore <- struct{}{}        // Acquire
            defer func() { <-semaphore }() // Release
            
            results[idx] = verifyPipeline(p) // Use existing verify logic
        }(i, pipeline)
    }
    
    wg.Wait()
    return results
}
```

**Concurrency Limits**:
  - Default: 5 concurrent verifications
  - Configurable via `--concurrency` flag
  - Rationale: Balance between speed and system resource usage

### 6. Output Formatting for List Command

**Decision**: Use `text/tabwriter` for aligned table output, support `--json` for scripting  
**Rationale**: Human-readable default, machine-parseable option follows Unix philosophy

**Table Format**:
```
ID             NAME                STATUS      LAST RUN         PATH
dev-setup      Development Setup   🟢 satisfied 2 hours ago      /home/user/.streamy/configs/dev.yaml
staging-env    Staging Environment 🟡 drifted   1 day ago        /home/user/.streamy/configs/staging.yaml
prod-deploy    Production Deploy   ⚪ unknown   never            /home/user/.streamy/configs/prod.yaml
```

**JSON Format**:
```json
{
  "version": "1.0",
  "pipelines": [
    {
      "id": "dev-setup",
      "name": "Development Setup",
      "path": "/home/user/.streamy/configs/dev.yaml",
      "description": "Local development environment",
      "status": "satisfied",
      "last_run": "2025-10-09T10:30:00Z",
      "registered_at": "2025-10-08T15:20:00Z"
    }
  ]
}
```

**Best Practices from Unix Tools**:
  - Wide terminals: Show full paths
  - Narrow terminals: Truncate paths with ellipsis
  - No TTY (piped): Use simpler format without colors/icons
  - Support `--format` flag: `table`, `json`, `csv`

### 7. Confirmation Prompts for Unregister

**Decision**: Use `bufio.Scanner` for stdin confirmation, bypass with `--force` flag  
**Rationale**: Standard Go pattern, simple and reliable

**Implementation**:
```go
func confirmUnregister(pipelineID string, force bool) bool {
    if force {
        return true
    }
    
    fmt.Fprintf(os.Stderr, "Remove pipeline '%s' from registry? [y/N]: ", pipelineID)
    scanner := bufio.NewScanner(os.Stdin)
    
    if !scanner.Scan() {
        return false // EOF or error
    }
    
    response := strings.ToLower(strings.TrimSpace(scanner.Text()))
    return response == "y" || response == "yes"
}
```

**Safety Checks**:
  - Default to "no" (capital N) for safety
  - Require explicit "y" or "yes"
  - Non-interactive environments: Must use `--force`

### 8. Dashboard Integration

**Decision**: No code changes required—dashboard already reads from registry  
**Rationale**: Dashboard's `LoadRegistry()` function in TUI already polls registry file

**Integration Points**:
  - Registry file location: `~/.streamy/registry.json` (already used)
  - Status cache location: `~/.streamy/status.json` (already used)
  - File watch: Dashboard already implements polling (no watch needed)

**Verified Behavior**:
  - Dashboard polls registry every 5 seconds (existing implementation)
  - Register/unregister commands call `Registry.Save()` (atomic write)
  - Next dashboard poll picks up changes automatically

### 9. Error Message Design

**Decision**: Follow established error pattern with context + suggestion  
**Rationale**: Consistent with existing commands, aligns with Constitution principle VII

**Error Template** (from existing code):
```go
type CommandError struct {
    Operation  string   // "register", "unregister", etc.
    Context    string   // What was being attempted
    Cause      error    // Underlying error
    Suggestion string   // Actionable next step
}

func (e *CommandError) Error() string {
    return fmt.Sprintf(
        "Failed to %s: %s\n\n%s\n\nSuggestion: %s",
        e.Operation, e.Context, e.Cause, e.Suggestion,
    )
}
```

**Example Error Messages**:
```
Failed to register pipeline: config file not found

Error: stat /home/user/configs/dev.yaml: no such file or directory

Suggestion: Check that the path is correct. Use an absolute path or a path relative to the current directory.
```

### 10. Testing Strategy

**Decision**: Three-tier testing approach matching existing patterns  
**Rationale**: Balance between coverage, speed, and maintainability

**Test Layers**:

1. **Unit Tests** (`cmd/streamy/*_test.go`):
   - Test individual command logic in isolation
   - Mock registry and file system interactions
   - Fast execution (<10ms per test)
   - Example: Test ID generation, validation logic, argument parsing

2. **Integration Tests** (`tests/integration_registry_test.go`):
   - Test commands end-to-end with real registry files
   - Use temporary directories for isolation
   - Test command sequences (register → list → unregister)
   - Verify file system state changes

3. **Contract Tests** (specification-based):
   - Verify command output matches documented contracts
   - Test error conditions from spec edge cases
   - Validate JSON schema compliance for `--json` output

**Test Utilities Needed**:
```go
// Helper functions for tests:
func setupTestRegistry(t *testing.T) (string, *registry.Registry)
func createTestConfig(t *testing.T, dir string, name string) string
func captureStdout(fn func()) string
func assertTableOutput(t *testing.T, output string, expectedRows int)
```

## Technical Decisions Summary

| Area | Decision | Key Factor |
|------|----------|------------|
| Registry Package | Reuse existing `internal/registry` | Already implements all requirements |
| Command Framework | Cobra (existing) | Consistency with current commands |
| ID Generation | Sanitized filename + override flag | Balance convenience and control |
| Config Validation | Use existing parser | Reuse battle-tested logic |
| Concurrency | Worker pool with semaphore | Bound resource usage |
| Output Format | Tabwriter + JSON option | Human & machine readable |
| Confirmation | bufio.Scanner + --force | Standard Go pattern |
| Dashboard Integration | None needed | Already reads registry file |
| Error Messages | Structured with suggestions | Matches existing pattern |
| Testing | Unit + Integration + Contract | Coverage with fast feedback |

## Open Questions

**None** - All technical unknowns from initial planning have been resolved through analysis of existing codebase and Go best practices.

## References

- Existing code: `cmd/streamy/apply.go`, `cmd/streamy/verify.go`
- Registry implementation: `internal/registry/registry.go`
- Config parser: `internal/config/parser.go`
- Cobra documentation: https://github.com/spf13/cobra
- Go standard library: `text/tabwriter`, `encoding/json`, `bufio`

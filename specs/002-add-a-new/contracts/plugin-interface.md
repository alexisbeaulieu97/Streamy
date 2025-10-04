# Contract: Template Plugin Interface

**Version**: 1.0.0  
**Plugin Type**: `template`  
**Interface**: `plugin.Plugin`

## Interface Contract

The template plugin MUST implement the standard `plugin.Plugin` interface:

```go
type Plugin interface {
    Metadata() Metadata
    Schema() interface{}
    Check(ctx context.Context, step *config.Step) (bool, error)
    Apply(ctx context.Context, step *config.Step) (*model.StepResult, error)
    DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error)
}
```

## Method Contracts

### Metadata() Metadata

**Purpose**: Return plugin identification metadata

**Contract**:
- MUST return non-nil `Metadata` struct
- `Name` field MUST be `"template-renderer"`
- `Version` field MUST follow semver (start with `"1.0.0"`)
- `Type` field MUST be `"template"` (matches step type in YAML)

**Example**:
```go
func (p *templatePlugin) Metadata() plugin.Metadata {
    return plugin.Metadata{
        Name:    "template-renderer",
        Version: "1.0.0",
        Type:    "template",
    }
}
```

**Postconditions**:
- Return value is never nil
- Type matches registry key used in `init()`

---

### Schema() interface{}

**Purpose**: Return struct representing configuration schema

**Contract**:
- MUST return `config.TemplateStep{}` struct
- Used for documentation generation and validation
- Struct MUST have yaml tags matching expected configuration

**Example**:
```go
func (p *templatePlugin) Schema() interface{} {
    return config.TemplateStep{}
}
```

**Postconditions**:
- Return value can be marshaled to JSON for schema docs
- Yaml tags match actual parsing behavior

---

### Check(ctx context.Context, step *config.Step) (bool, error)

**Purpose**: Fast idempotency check - determine if step needs to run

**Preconditions**:
- `step` is non-nil
- `step.Template` is non-nil (validated before plugin invocation)
- `ctx` respects cancellation

**Contract**:
- MUST NOT modify any files or system state
- MUST be fast (<50ms for typical configs)
- Return `(true, nil)` if step can be skipped (already satisfied)
- Return `(false, nil)` if step needs to run
- Return `(false, error)` if unable to determine (treated as needs-apply)

**Algorithm**:
1. Check if source template file exists and is readable
2. If destination doesn't exist → return `(false, nil)` (needs creation)
3. Render template to memory (use same logic as Apply)
4. Hash rendered content (SHA-256)
5. Hash existing destination file (SHA-256)
6. Compare hashes:
   - Equal → return `(true, nil)` (skip)
   - Different → return `(false, nil)` (needs update)

**Error Cases**:
- Source file not found → return `(false, ValidationError)`
- Source file not readable → return `(false, PluginError)`
- Template syntax error → return `(false, PluginError)`
- Missing variable (if !allow_missing) → return `(false, PluginError)`
- Context cancelled → return `(false, context.Canceled)`

**Example**:
```go
func (p *templatePlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
    cfg := step.Template
    
    // Check source exists
    if _, err := os.Stat(cfg.Source); err != nil {
        return false, streamyerrors.NewPluginError("template", step.ID, 
            fmt.Errorf("source not found: %w", err))
    }
    
    // If destination doesn't exist, needs apply
    if _, err := os.Stat(cfg.Destination); os.IsNotExist(err) {
        return false, nil
    }
    
    // Render and compare hashes
    rendered, err := p.renderTemplate(ctx, cfg)
    if err != nil {
        return false, err
    }
    
    renderedHash := sha256.Sum256(rendered)
    existingHash, err := hashFile(cfg.Destination)
    if err != nil {
        return false, err
    }
    
    return bytes.Equal(renderedHash[:], existingHash[:]), nil
}
```

**Postconditions**:
- No files created or modified
- Error messages include step ID and context
- Performance: <50ms for files <1MB

---

### Apply(ctx context.Context, step *config.Step) (*model.StepResult, error)

**Purpose**: Execute the template rendering and write output

**Preconditions**:
- `step` is non-nil
- `step.Template` is non-nil and validated
- `ctx` respects cancellation
- Typically called after `Check()` returns `(false, nil)`

**Contract**:
- MUST render template with variable substitution
- MUST write rendered output to destination file
- MUST create parent directories if they don't exist
- MUST set file permissions (from mode or source)
- MUST be idempotent (safe to call multiple times)
- MUST populate `StepResult` with status and messages

**Algorithm**:
1. Validate source file exists and is readable
2. Load template content from source file
3. Parse template using `text/template`
4. Build variable context (env vars + inline vars, inline takes precedence)
5. Execute template with variable context
6. Check if destination exists and matches rendered content (idempotency)
7. If matches, return early with StatusSkipped
8. Create parent directories for destination
9. Write rendered content to destination file
10. Set file permissions (explicit mode or copy from source)
11. Return StepResult with StatusComplete

**Error Cases**:
- Source file not found → return error with StatusFailed
- Template parse error → return error with line/column details, StatusFailed
- Missing variable (!allow_missing) → return error with variable name, StatusFailed
- Destination not writable → return error with path, StatusFailed
- Context cancelled → return context.Canceled

**StepResult States**:
- `StatusComplete` - File created or updated successfully
- `StatusSkipped` - File already matches rendered content (idempotent)
- `StatusFailed` - Error occurred (error field populated)

**Example**:
```go
func (p *templatePlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
    cfg := step.Template
    
    // Render template
    rendered, err := p.renderTemplate(ctx, cfg)
    if err != nil {
        return &model.StepResult{
            StepID:  step.ID,
            Status:  model.StatusFailed,
            Message: "Template rendering failed",
            Error:   err,
        }, err
    }
    
    // Idempotency check
    if fileExists(cfg.Destination) {
        existing, _ := os.ReadFile(cfg.Destination)
        if bytes.Equal(rendered, existing) {
            return &model.StepResult{
                StepID:  step.ID,
                Status:  model.StatusSkipped,
                Message: "File already matches rendered content",
            }, nil
        }
    }
    
    // Create parent dirs
    os.MkdirAll(filepath.Dir(cfg.Destination), 0755)
    
    // Determine file mode
    var mode os.FileMode
    if cfg.Mode != nil {
        mode = os.FileMode(*cfg.Mode)
    } else {
        info, _ := os.Stat(cfg.Source)
        mode = info.Mode()
    }
    
    // Write file
    if err := os.WriteFile(cfg.Destination, rendered, mode); err != nil {
        return &model.StepResult{
            StepID:  step.ID,
            Status:  model.StatusFailed,
            Message: "Failed to write destination file",
            Error:   err,
        }, err
    }
    
    return &model.StepResult{
        StepID:  step.ID,
        Status:  model.StatusComplete,
        Message: fmt.Sprintf("Rendered %s to %s", cfg.Source, cfg.Destination),
    }, nil
}
```

**Postconditions**:
- If successful, destination file exists with correct content and permissions
- If skipped, no files modified
- StepResult always populated with StepID, Status, Message
- Errors include context (file paths, line numbers, variable names)
- Performance: <100ms for files <1MB

---

### DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error)

**Purpose**: Preview what Apply would do without modifying files

**Preconditions**:
- Same as `Apply()`

**Contract**:
- MUST NOT modify any files or system state
- MUST render template to memory
- MUST report what would happen if Apply were called
- MUST be fast (faster than Apply since no I/O)

**Algorithm**:
1. Validate source file exists
2. Render template to memory (same as Apply)
3. Check if destination exists:
   - If no → return StatusWouldCreate
   - If yes, compare content:
     - Match → return StatusSkipped
     - Different → return StatusWouldUpdate

**StepResult States**:
- `StatusSkipped` - No changes (file matches rendered content)
- `StatusWouldCreate` - Destination doesn't exist, would be created
- `StatusWouldUpdate` - Destination exists but differs, would be updated
- `StatusFailed` - Error during preview (e.g., template syntax error)

**Example**:
```go
func (p *templatePlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
    cfg := step.Template
    
    // Render template (same as Apply, but don't write)
    rendered, err := p.renderTemplate(ctx, cfg)
    if err != nil {
        return &model.StepResult{
            StepID:  step.ID,
            Status:  model.StatusFailed,
            Message: "Template rendering failed",
            Error:   err,
        }, err
    }
    
    // Determine what would happen
    if !fileExists(cfg.Destination) {
        return &model.StepResult{
            StepID:  step.ID,
            Status:  model.StatusWouldCreate,
            Message: fmt.Sprintf("Would create %s", cfg.Destination),
        }, nil
    }
    
    existing, _ := os.ReadFile(cfg.Destination)
    if bytes.Equal(rendered, existing) {
        return &model.StepResult{
            StepID:  step.ID,
            Status:  model.StatusSkipped,
            Message: "No changes needed",
        }, nil
    }
    
    return &model.StepResult{
        StepID:  step.ID,
        Status:  model.StatusWouldUpdate,
        Message: fmt.Sprintf("Would update %s", cfg.Destination),
    }, nil
}
```

**Postconditions**:
- No files created or modified
- StepResult indicates what Apply would do
- Performance: <50ms for files <1MB

---

## Registration Contract

The plugin MUST register itself in an `init()` function:

```go
func init() {
    if err := plugin.RegisterPlugin("template", New()); err != nil {
        panic(err)
    }
}
```

**Contract**:
- Key must be `"template"` (matches step type)
- Must happen before main() execution
- Duplicate registration causes panic (by design)

---

## Error Handling Contract

All errors MUST:
- Include step ID context
- Use `pkg/errors` helpers (`NewPluginError`, `NewValidationError`)
- Provide actionable messages with:
  - What went wrong
  - Where it went wrong (file path, line/column)
  - How to fix it (suggestions)

**Example Error Messages**:
```
✗ template rendering failed in step 'render-config'
  source: templates/app.conf.tmpl
  error: undefined variable "DATABASE_URL" at line 15, column 12
  suggestion: add to 'vars' map or set allow_missing: true

✗ template syntax error in step 'render-config'
  source: templates/app.conf.tmpl
  error: unexpected "}" at line 8, column 25
  suggestion: check template syntax, ensure all {{if}} have {{end}}
```

---

## Performance Contract

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Metadata() | <1µs | Instant (returns struct) |
| Schema() | <1µs | Instant (returns struct) |
| Check() | <50ms | For files <1MB |
| Apply() | <100ms | For files <1MB |
| DryRun() | <50ms | For files <1MB |

Exceeding these targets for typical configs is a performance regression.

---

## Thread Safety Contract

All methods MUST be:
- Thread-safe (multiple concurrent calls allowed)
- Stateless (no shared mutable state between calls)
- Context-aware (respect ctx cancellation)

The plugin struct itself contains no state; all state passed via `step` parameter.

---

## Test Contract

The plugin MUST include tests covering:

1. **Happy Path**: Basic rendering with inline vars
2. **Environment Variables**: Substitution from env
3. **Variable Precedence**: Inline overrides env
4. **Idempotency**: Re-running doesn't overwrite identical files
5. **Missing Variables**: Fails by default, succeeds with allow_missing
6. **Template Syntax Errors**: Clear error messages with line/column
7. **File Permissions**: Mode set correctly
8. **DryRun**: Shows correct would-create/would-update/skip status
9. **Parent Directory Creation**: Creates destination dirs
10. **Edge Cases**: Empty template, large files, special characters

Minimum test coverage: 80% (measured by `go test -cover`)

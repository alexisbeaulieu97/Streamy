# Research: Template Plugin Implementation

**Date**: 2025-10-04  
**Feature**: Template Plugin for Dynamic File Rendering

## Research Questions & Findings

### 1. Go text/template Package Capabilities

**Decision**: Use Go's `text/template` package from standard library

**Rationale**:
- Zero external dependencies (aligns with Constitution Principle I)
- Mature, well-documented, battle-tested in production
- Supports variables, conditionals (`{{if}}`), loops (`{{range}}`), functions
- Built-in error reporting with line/column information
- Data context via struct fields (`.VariableName` syntax)
- Thread-safe template execution

**Alternatives Considered**:
- `html/template`: Rejected - adds HTML escaping we don't need for config files
- External template libraries (Mustache, Handlebars): Rejected - adds dependencies
- Simple string replacement: Rejected - doesn't support conditionals/loops for future extensibility

**Key Implementation Notes**:
- Use `template.New().Parse()` for template parsing
- Pass variables as map or struct to `Execute()`
- Error handling: `Parse()` errors include line/column via `template.Error`
- Funcs: Can add custom functions via `template.Funcs()` for future extensions

### 2. Variable Substitution Strategy

**Decision**: Two-tier variable resolution: inline vars override environment vars

**Rationale**:
- Explicit (inline vars) over implicit (env vars) principle
- Allows team configs to override local environment
- Common pattern in config management tools (Docker Compose, Kubernetes)
- Simple to implement: merge maps with inline vars taking precedence

**Implementation Approach**:
```go
// Pseudo-code
vars := make(map[string]string)
// First: load environment variables
for k, v := range os.Environ() { vars[k] = v }
// Second: override with inline vars (if env=true)
if cfg.Env {
    for k, v := range os.Environ() { vars[k] = v }
}
// Third: apply inline vars (always takes precedence)
for k, v := range cfg.Vars { vars[k] = v }
```

**Alternatives Considered**:
- Inline vars only: Rejected - forces duplication in configs
- Env vars only: Rejected - less explicit, harder to version control
- Env vars override inline: Rejected - violates explicit-over-implicit principle

### 3. Idempotency Implementation

**Decision**: SHA-256 hash comparison of rendered content vs existing file

**Rationale**:
- Fast: SHA-256 hashing is ~500MB/s on modern CPUs
- Reliable: cryptographic hash eliminates false positives
- Memory efficient: no need to load both files for comparison
- Follows pattern from copy plugin (already uses SHA-256)

**Implementation Approach**:
1. Render template to memory buffer
2. Hash rendered content (SHA-256)
3. If destination exists, hash existing file
4. Compare hashes - if equal, return "skipped" status
5. If different or file doesn't exist, write new content

**Alternatives Considered**:
- Byte-by-byte comparison: Rejected - requires loading both files, slower
- Timestamp comparison: Rejected - unreliable (clock skew, git operations)
- CRC32: Rejected - not cryptographically secure, potential collisions

### 4. File Permission Handling

**Decision**: Copy source file permissions by default, allow explicit override via optional `mode` field

**Rationale**:
- Preserves template author's intent (executable templates → executable outputs)
- Explicit override for security-sensitive files (e.g., 0600 for credentials)
- Follows Unix philosophy: predictable, composable behavior
- Aligns with Constitution Principle IV (safe defaults)

**Implementation Approach**:
```go
var fileMode os.FileMode
if cfg.Mode != nil {
    fileMode = os.FileMode(*cfg.Mode) // Explicit override
} else {
    sourceInfo, _ := os.Stat(cfg.Source)
    fileMode = sourceInfo.Mode() // Copy from source
}
os.WriteFile(cfg.Destination, rendered, fileMode)
```

**Alternatives Considered**:
- System umask: Rejected - unpredictable across environments
- Fixed 0644: Rejected - breaks executable script templates
- Require explicit mode always: Rejected - unnecessary friction for common cases

### 5. Error Handling for Missing Variables

**Decision**: Fail fast by default, allow `allow_missing: true` for optional variables

**Rationale**:
- Fail fast prevents deployment of incomplete configs (Constitution Principle IV)
- Clear error messages reduce debugging time (Constitution Principle V)
- Opt-in tolerance for optional variables (flexibility without sacrificing safety)
- Go text/template supports this via `template.Option("missingkey=error")`

**Implementation Approach**:
```go
tmpl := template.New("template")
if !cfg.AllowMissing {
    tmpl = tmpl.Option("missingkey=error") // Fail on missing vars
} else {
    tmpl = tmpl.Option("missingkey=zero") // Replace with zero value
}
```

**Error Message Format**:
```
template rendering failed: variable "DATABASE_URL" is undefined
  template: line 15, column 12
  suggestion: define in 'vars' map or set allow_missing: true
```

**Alternatives Considered**:
- Always allow missing: Rejected - silent failures in production
- Always fail on missing: Rejected - prevents legitimate optional variable use cases
- Warn on missing: Rejected - warnings often ignored, leads to production issues

### 6. Dry-Run Mode Implementation

**Decision**: Render to memory, show diff, skip write

**Rationale**:
- Consistent with other plugins' DryRun behavior
- Shows exact changes without side effects
- Leverages standard diff libraries or simple "would create/update" messages

**Implementation Approach**:
```go
func (p *templatePlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
    rendered := renderTemplate(step.Template)
    
    if fileExists(step.Template.Destination) {
        existingHash := hashFile(step.Template.Destination)
        renderedHash := hash(rendered)
        if existingHash == renderedHash {
            return &model.StepResult{Status: StatusSkipped, Message: "No changes"}, nil
        }
        return &model.StepResult{Status: StatusWouldUpdate, Message: "Would update file"}, nil
    }
    return &model.StepResult{Status: StatusWouldCreate, Message: "Would create file"}, nil
}
```

**Alternatives Considered**:
- Full diff output: Deferred - nice-to-have, not MVP requirement
- No dry-run: Rejected - violates Constitution Principle IV

### 7. Template Syntax Error Handling

**Decision**: Fail immediately with line/column precision from Go text/template parser

**Rationale**:
- Go's `template.Parse()` provides detailed error information
- Fast feedback during config development
- Prevents confusing runtime errors
- Aligns with Constitution Principle V (clear diagnostics)

**Error Extraction**:
```go
_, err := template.New("tmpl").Parse(templateContent)
if err != nil {
    // Go provides errors like:
    // template: tmpl:15:12: unexpected "}" in operand
    return fmt.Errorf("template syntax error in %s: %w", cfg.Source, err)
}
```

### 8. Plugin Interface Integration

**Decision**: Follow existing plugin patterns (copy, package, command plugins)

**Rationale**:
- Consistency across plugin ecosystem (Constitution Principle VII)
- Proven patterns reduce implementation risk
- Automatic integration with DAG engine, logging, TUI

**Required Methods**:
- `Metadata()` - returns name="template-renderer", version="1.0.0", type="template"
- `Schema()` - returns `TemplateStep{}` struct for docs
- `Check()` - fast idempotency check (hash comparison only)
- `Apply()` - full render and write
- `DryRun()` - render to memory, report changes

**Registration**:
```go
// cmd/streamy/plugins_import.go
import _ "github.com/alexisbeaulieu97/streamy/internal/plugins/template"

// internal/plugins/template/template.go
func init() {
    plugin.RegisterPlugin("template", New())
}
```

## Technology Stack Confirmation

| Component | Technology | Version | Rationale |
|-----------|-----------|---------|-----------|
| Template Engine | Go text/template | stdlib | Zero deps, full feature set |
| Hashing | crypto/sha256 | stdlib | Fast, reliable idempotency |
| File I/O | os, io, path/filepath | stdlib | Cross-platform, standard |
| Testing | testing package | stdlib | Table-driven, temp dirs |
| Error Handling | pkg/errors helpers | internal | Consistent error context |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Large template files (>100MB) slow rendering | Low | Medium | Document size limits, add timeout |
| Complex templates with loops cause CPU spikes | Low | Medium | Document best practices, consider complexity limits |
| Binary data in templates corrupts output | Very Low | Low | Validate template is text, error on binary |
| Concurrent template renders conflict | Very Low | Low | Template execution is stateless and thread-safe |

## Performance Benchmarks (Expected)

Based on Go text/template benchmarks and similar implementations:

- Template parsing: <5ms for typical configs (<10KB)
- Variable substitution: <1ms per variable
- SHA-256 hashing: ~500MB/s (~20ms for 10MB file)
- File write: I/O bound (~100MB/s on modern SSD)

**Target**: <100ms end-to-end for typical config file (<1MB, <20 variables)

## Open Questions

None - all technical unknowns resolved during clarification session and research phase.

## References

- Go text/template docs: https://pkg.go.dev/text/template
- Existing plugin patterns: `internal/plugins/{copy,command,package}/`
- Constitution: `.specify/memory/constitution.md`
- Feature spec: `specs/002-add-a-new/spec.md`

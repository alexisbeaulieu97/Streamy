# Data Model: Template Plugin

**Date**: 2025-10-04  
**Feature**: Template Plugin for Dynamic File Rendering

## Configuration Entity

### TemplateStep

Represents a single template rendering operation in a Streamy configuration file.

**Location**: `internal/config/types.go`

```go
// TemplateStep renders a file from a template with variable substitution.
type TemplateStep struct {
    Source       string            `yaml:"source" validate:"required"`
    Destination  string            `yaml:"destination" validate:"required,nefield=Source"`
    Vars         map[string]string `yaml:"vars,omitempty"`
    Env          bool              `yaml:"env,omitempty"`
    AllowMissing bool              `yaml:"allow_missing,omitempty"`
    Mode         *uint32           `yaml:"mode,omitempty" validate:"omitempty,min=0,max=0777"`
}
```

**Fields**:

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `source` | string | Yes | - | non-empty file path | Path to template file (Go text/template format) |
| `destination` | string | Yes | - | non-empty, different from source | Path where rendered file will be written |
| `vars` | map[string]string | No | empty map | keys follow Go identifier rules | Inline variables for template substitution (takes precedence over env vars) |
| `env` | bool | No | true | - | If true, environment variables are available for substitution |
| `allow_missing` | bool | No | false | - | If true, missing variables are replaced with empty string; if false, missing variables cause failure |
| `mode` | *uint32 | No | nil (copy from source) | 0-0777 octal | File permissions for destination file; if nil, copies from source template |

**Validation Rules** (enforced by `internal/config/validator.go`):

1. `source` must be a non-empty string
2. `destination` must be a non-empty string and different from `source`
3. `vars` keys must match Go identifier pattern: `^[a-zA-Z_][a-zA-Z0-9_]*$`
4. `mode`, if provided, must be valid Unix file permission (0-0777 octal)
5. At least one of `vars` or `env=true` should provide variables (warning, not error)

**Default Values**:
- `env`: true (environment variables enabled by default)
- `allow_missing`: false (fail on missing variables by default)
- `mode`: nil (copy permissions from source template)
- `vars`: empty map

**Examples**:

```yaml
# Minimal example
- id: render-config
  type: template
  source: templates/app.conf.tmpl
  destination: /etc/app/app.conf

# With inline variables
- id: render-env-file
  type: template
  source: templates/.env.tmpl
  destination: .env
  vars:
    DATABASE_URL: postgres://localhost/mydb
    API_KEY: abc123
  mode: 0600  # Restrict permissions for secrets

# With optional variables
- id: render-optional-config
  type: template
  source: templates/optional.conf.tmpl
  destination: config/optional.conf
  allow_missing: true
  env: false  # Only use inline vars

# Complex example with dependencies
- id: render-docker-compose
  type: template
  source: templates/docker-compose.yml.tmpl
  destination: docker-compose.yml
  vars:
    APP_VERSION: "1.2.3"
    REPLICA_COUNT: "3"
  depends_on:
    - clone-repo
```

**Integration with Step**:

The `TemplateStep` is embedded in the main `Step` struct via inline YAML:

```go
// In internal/config/types.go
type Step struct {
    // ... base fields (ID, Type, Name, etc.)
    
    // Type-specific configs (inline)
    Template *TemplateStep `yaml:",inline,omitempty"`
    Package  *PackageStep  `yaml:",inline,omitempty"`
    Copy     *CopyStep     `yaml:",inline,omitempty"`
    // ... other step types
}
```

## Runtime Entities

### TemplateContext

Internal data structure passed to Go text/template during rendering.

**Not serialized** - exists only during template execution.

```go
// TemplateContext provides variables to the template engine
type TemplateContext map[string]string
```

**Structure**:
- Flat map of variable name → value
- Merged from environment variables (if `env=true`) and inline `vars`
- Inline `vars` override environment variables with same name

**Lifecycle**:
1. Created in `Apply()` method
2. Populated with env vars (if enabled)
3. Overlaid with inline vars
4. Passed to `template.Execute()`
5. Discarded after rendering

### TemplateMetadata

Plugin metadata returned by `Metadata()` method.

```go
type TemplateMetadata struct {
    Name    string // "template-renderer"
    Version string // "1.0.0"
    Type    string // "template"
}
```

## Error Entities

Errors follow Streamy's structured error pattern using `pkg/errors` helpers.

### ValidationError

Raised during config validation (before execution).

```go
// Example: invalid variable name
streamyerrors.NewValidationError(
    stepID,
    "variable name 'my-var' invalid: must follow Go identifier rules (alphanumeric and underscore only)",
    nil,
)
```

### PluginError

Raised during plugin execution (Check, Apply, DryRun).

```go
// Example: template file not found
streamyerrors.NewPluginError(
    "template",
    stepID,
    fmt.Errorf("source template not found: %s", cfg.Source),
)

// Example: missing variable
streamyerrors.NewPluginError(
    "template",
    stepID,
    fmt.Errorf("undefined variable 'DATABASE_URL' at line 15, column 12"),
)

// Example: template syntax error
streamyerrors.NewPluginError(
    "template",
    stepID,
    fmt.Errorf("template syntax error in %s: %w", cfg.Source, parseErr),
)
```

## State Transitions

Template rendering follows this state flow:

```
[Config Loaded]
      ↓
[Validation] → (Error: invalid config) → FAIL
      ↓
[Check: idempotency test]
      ├→ (Hashes match) → SKIP (no-op)
      └→ (Hashes differ or destination missing) → NEEDS_APPLY
            ↓
      [Apply: render template]
            ├→ (Parse error) → FAIL
            ├→ (Missing var & !allow_missing) → FAIL
            ├→ (Write error) → FAIL
            └→ (Success) → COMPLETE
```

**DryRun State**:
```
[DryRun: simulate rendering]
      ├→ (Would create) → WOULD_CREATE
      ├→ (Would update) → WOULD_UPDATE
      └→ (No changes) → SKIP
```

## Relationships

```
Step (1) ──contains──> (0..1) TemplateStep
     ↓
     uses
     ↓
TemplatePlugin ──creates──> TemplateContext
     ↓                            ↓
     parses                       provides vars to
     ↓                            ↓
Template Source File ←────────→ text/template Engine
     ↓                            ↓
     reads content               renders output
     ↓                            ↓
[File Content] ──hashed──> [SHA-256] ──compared with──> [Existing Destination Hash]
                                                              ↓
                                                    (match=SKIP, diff=WRITE)
                                                              ↓
                                                    [Destination File]
```

## Validation Constraints Summary

| Constraint | Enforced By | Phase | Error Type |
|-----------|-------------|-------|------------|
| Source required | YAML validator | Parse | ValidationError |
| Destination required | YAML validator | Parse | ValidationError |
| Source ≠ Destination | YAML validator | Parse | ValidationError |
| Var names valid Go identifiers | Custom validator | Parse | ValidationError |
| Mode in range 0-0777 | YAML validator | Parse | ValidationError |
| Source file exists | Plugin Check | Runtime | PluginError |
| Source is readable | Plugin Check | Runtime | PluginError |
| Destination writable | Plugin Apply | Runtime | PluginError |
| Template syntax valid | Plugin Apply | Runtime | PluginError |
| Variables defined (if !allow_missing) | Plugin Apply | Runtime | PluginError |

## Performance Characteristics

| Operation | Complexity | Typical Time | Notes |
|-----------|-----------|--------------|-------|
| Parse template | O(n) | <5ms | n = template size in bytes |
| Variable substitution | O(v) | <1ms | v = number of variables |
| Hash computation | O(m) | ~20ms | m = rendered content size (assumes 500MB/s) |
| File write | O(m) | I/O bound | ~100MB/s on SSD |
| Idempotency check | O(m) | 2× hash time | Hash both existing and rendered |

**Memory Usage**:
- Peak: ~3× template size (original + rendered + hash buffer)
- Typical: <10MB for config files

## Future Extensions (Out of Scope for MVP)

Potential enhancements that don't require data model changes:

1. **Template Functions**: Add custom functions via `template.Funcs()`
   - Example: `{{.VarName | default "fallback"}}`
   - Implementation: extend TemplateContext with function map

2. **Include/Partial Templates**: Support `{{template "header" .}}`
   - Requires multi-file template parsing
   - Add optional `includes` field to TemplateStep

3. **Structured Variables**: Support nested maps/lists
   - Change `Vars map[string]string` to `Vars map[string]interface{}`
   - Requires YAML parsing adjustments

4. **Diff Output in DryRun**: Show actual changes
   - Enhance DryRun to compute and display unified diff
   - No data model changes required

None of these require breaking changes to the core data model.

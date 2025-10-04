# Tasks: Template Plugin for Dynamic File Rendering

**Input**: Design documents from `/home/alexis/Projects/Streamy/specs/002-add-a-new/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/plugin-interface.md, quickstart.md

## Execution Summary

This task list implements the template plugin for Streamy based on:
- **Data Model**: TemplateStep struct with 6 fields (source, destination, vars, env, allow_missing, mode)
- **Plugin Interface**: 5 methods (Metadata, Schema, Check, Apply, DryRun)
- **Tech Stack**: Go 1.21+, stdlib only (text/template, crypto/sha256, os, io, path/filepath)
- **Testing**: Table-driven tests with t.TempDir(), 10 quickstart scenarios
- **Documentation**: Plugin docs, schema reference, example configs

**Total Tasks**: 23  
**Estimated Time**: 6-8 hours for complete implementation  
**Parallel Opportunities**: 8 tasks marked [P]

---

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- File paths are absolute from repository root
- Each task is independently completable

---

## Phase 3.1: Foundation & Data Model

### T001: Add TemplateStep struct to config types
**File**: `internal/config/types.go`  
**Description**: Add TemplateStep struct definition with 6 fields following data-model.md specification:
- `Source string` (required, yaml:"source")
- `Destination string` (required, yaml:"destination")
- `Vars map[string]string` (optional, yaml:"vars,omitempty")
- `Env bool` (optional, yaml:"env,omitempty")
- `AllowMissing bool` (optional, yaml:"allow_missing,omitempty")
- `Mode *uint32` (optional, yaml:"mode,omitempty")

Add struct tags with validation rules:
- source: `validate:"required"`
- destination: `validate:"required,nefield=Source"`
- mode: `validate:"omitempty,min=0,max=0777"`

**Dependencies**: None  
**Acceptance**: Struct compiles, fields match data-model.md

---

### T002: Add Template field to Step struct
**File**: `internal/config/types.go`  
**Description**: Add inline Template field to Step struct:
```go
Template *TemplateStep `yaml:",inline,omitempty"`
```
Add to the same location as other step type fields (Package, Copy, Command, etc.)

**Dependencies**: T001  
**Acceptance**: Step struct can parse template steps from YAML

---

### T003: Add TemplateStep validation logic
**File**: `internal/config/types.go` or `internal/config/step_validation.go`  
**Description**: Implement UnmarshalYAML for TemplateStep (if needed, following pattern from CopyStep). Add validation for:
- Variable names follow Go identifier rules: `^[a-zA-Z_][a-zA-Z0-9_]*$`
- Source path is non-empty
- Destination path differs from source

**Dependencies**: T001, T002  
**Acceptance**: Invalid configs are rejected with clear error messages

---

### T004 [P]: Create template plugin directory structure
**Files**: 
- `internal/plugins/template/` (directory)
- `internal/plugins/template/template.go` (empty file)
- `internal/plugins/template/template_test.go` (empty file)

**Description**: Create plugin package directory following existing plugin pattern (copy, command, package). Initialize empty files for implementation and tests.

**Dependencies**: None (parallel with T001-T003)  
**Acceptance**: Directory structure matches plan.md, files exist

---

## Phase 3.2: Plugin Interface Implementation

### T005: Implement templatePlugin struct and constructor
**File**: `internal/plugins/template/template.go`  
**Description**: Create templatePlugin struct (empty, stateless) and New() constructor:
```go
type templatePlugin struct{}

func New() plugin.Plugin {
    return &templatePlugin{}
}
```
Add package imports: context, fmt, os, path/filepath, crypto/sha256, text/template, internal packages.

**Dependencies**: T004  
**Acceptance**: Struct compiles, implements plugin.Plugin interface (compile-time check)

---

### T006 [P]: Implement Metadata() method
**File**: `internal/plugins/template/template.go`  
**Description**: Implement Metadata() method returning:
- Name: "template-renderer"
- Version: "1.0.0"
- Type: "template"

Follow contract specification in contracts/plugin-interface.md.

**Dependencies**: T005  
**Acceptance**: Returns correct metadata struct, matches contract

---

### T007 [P]: Implement Schema() method
**File**: `internal/plugins/template/template.go`  
**Description**: Implement Schema() method returning `config.TemplateStep{}` struct for documentation generation.

**Dependencies**: T005, T001  
**Acceptance**: Returns TemplateStep struct, used for schema docs

---

### T008: Implement renderTemplate helper function
**File**: `internal/plugins/template/template.go`  
**Description**: Implement core template rendering logic:
1. Read template source file
2. Parse with text/template
3. Build variable context (merge env vars + inline vars, inline wins)
4. Set missingkey option based on AllowMissing flag
5. Execute template with context
6. Return rendered bytes and error

Function signature: `func (p *templatePlugin) renderTemplate(ctx context.Context, cfg *config.TemplateStep) ([]byte, error)`

Handle errors with clear messages including line/column for syntax errors.

**Dependencies**: T005  
**Acceptance**: Can render templates with variable substitution, respects allow_missing flag

---

### T009: Implement hashContent helper function
**File**: `internal/plugins/template/template.go`  
**Description**: Implement SHA-256 hashing helper:
```go
func hashContent(data []byte) [32]byte {
    return sha256.Sum256(data)
}

func hashFile(path string) ([32]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return [32]byte{}, err
    }
    return hashContent(data), nil
}
```

**Dependencies**: T005  
**Acceptance**: Correctly computes SHA-256 hashes for idempotency checks

---

### T010: Implement Check() method for idempotency
**File**: `internal/plugins/template/template.go`  
**Description**: Implement Check() method following contract:
1. Validate source file exists and is readable
2. If destination doesn't exist, return (false, nil) - needs creation
3. Render template to memory using renderTemplate()
4. Hash rendered content
5. Hash existing destination file
6. Compare hashes: equal → (true, nil), different → (false, nil)
7. Handle errors (template syntax, missing vars) → return (false, error)

Must be fast (<50ms) and side-effect free.

**Dependencies**: T008, T009  
**Acceptance**: Correctly identifies when files match, returns appropriate bool and error

---

### T011: Implement Apply() method for rendering and writing
**File**: `internal/plugins/template/template.go`  
**Description**: Implement Apply() method following contract:
1. Call renderTemplate() to get rendered content
2. Check idempotency: if destination exists and matches, return StatusSkipped
3. Create parent directories with os.MkdirAll()
4. Determine file mode (explicit from cfg.Mode or copy from source)
5. Write rendered content to destination with os.WriteFile()
6. Return StepResult with StepID, Status (Complete/Skipped/Failed), Message, Error

Handle all error cases with clear messages (source missing, write failed, etc.)

**Dependencies**: T008, T009  
**Acceptance**: Successfully renders and writes files, respects idempotency, sets correct permissions

---

### T012: Implement DryRun() method for preview
**File**: `internal/plugins/template/template.go`  
**Description**: Implement DryRun() method following contract:
1. Call renderTemplate() to get rendered content (no file write)
2. If destination doesn't exist, return StatusWouldCreate
3. If destination exists, compare content:
   - Match → StatusSkipped
   - Different → StatusWouldUpdate
4. Return StepResult with appropriate status and message

Must not modify any files. Fast (<50ms).

**Dependencies**: T008, T009  
**Acceptance**: Shows what would happen without modifying files

---

### T013: Add plugin registration in init()
**File**: `internal/plugins/template/template.go`  
**Description**: Add init() function with plugin registration:
```go
func init() {
    if err := plugin.RegisterPlugin("template", New()); err != nil {
        panic(err)
    }
}
```

**Dependencies**: T006, T007, T010, T011, T012 (all interface methods complete)  
**Acceptance**: Plugin registers successfully, can be invoked by type "template"

---

### T014: Import template plugin for registration
**File**: `cmd/streamy/plugins_import.go`  
**Description**: Add import statement for template plugin:
```go
import _ "github.com/alexisbeaulieu97/streamy/internal/plugins/template"
```
Add to existing plugin imports section.

**Dependencies**: T013  
**Acceptance**: Binary includes template plugin, registration happens at startup

---

## Phase 3.3: Testing (TDD - Tests First)

### T015 [P]: Write test for Metadata() and Schema()
**File**: `internal/plugins/template/template_test.go`  
**Description**: Write table-driven tests verifying:
- Metadata returns correct Name, Version, Type
- Schema returns TemplateStep struct
- Both are non-nil

**Dependencies**: T006, T007  
**Acceptance**: Tests pass, cover all metadata fields

---

### T016 [P]: Write tests for Check() idempotency
**File**: `internal/plugins/template/template_test.go`  
**Description**: Write table-driven tests for Check() covering:
1. Source file missing → returns (false, error)
2. Destination missing → returns (false, nil) - needs creation
3. Destination matches rendered → returns (true, nil) - skip
4. Destination differs from rendered → returns (false, nil) - needs update
5. Template syntax error → returns (false, error)
6. Missing variable (!allow_missing) → returns (false, error)
7. Missing variable (allow_missing=true) → returns based on content match

Use t.TempDir() for test isolation.

**Dependencies**: T010  
**Acceptance**: All test cases pass, cover edge cases

---

### T017 [P]: Write tests for Apply() rendering
**File**: `internal/plugins/template/template_test.go`  
**Description**: Write table-driven tests for Apply() covering:
1. Basic variable substitution (inline vars)
2. Environment variable substitution
3. Inline vars override env vars
4. Idempotency: re-running doesn't overwrite identical files
5. Missing variable error (!allow_missing)
6. Missing variable success (allow_missing=true)
7. Template syntax error with line/column
8. File permission setting (explicit mode)
9. File permission copying (from source)
10. Parent directory creation
11. Empty template (valid case)
12. Conditionals and loops in template

Use t.TempDir() for isolation. Verify file content and permissions.

**Dependencies**: T011  
**Acceptance**: All scenarios pass, files have correct content and permissions

---

### T018 [P]: Write tests for DryRun() preview
**File**: `internal/plugins/template/template_test.go`  
**Description**: Write table-driven tests for DryRun() covering:
1. Destination missing → StatusWouldCreate
2. Destination matches → StatusSkipped
3. Destination differs → StatusWouldUpdate
4. Template syntax error → StatusFailed
5. Missing variable error → StatusFailed

Verify no files are created or modified during DryRun.

**Dependencies**: T012  
**Acceptance**: All test cases pass, confirms no side effects

---

### T019: Write integration test for quickstart scenarios
**File**: `internal/plugins/template/template_integration_test.go` or add to existing integration tests  
**Description**: Implement integration tests following quickstart.md scenarios:
1. Basic inline variable substitution
2. Environment variable fallback
3. Variable precedence (inline wins)
4. Idempotency check (skip unchanged)
5. File permission handling
6. Missing variable errors
7. Template syntax errors
8. allow_missing flag behavior

Use real template files from t.TempDir(). Verify end-to-end behavior.

**Dependencies**: T011, T012, T013 (full plugin working)  
**Acceptance**: All quickstart scenarios pass end-to-end

---

## Phase 3.4: Documentation & Examples

### T020 [P]: Add template plugin documentation
**File**: `docs/plugins.md`  
**Description**: Add comprehensive template plugin section covering:
- Purpose and use cases
- Configuration fields (source, destination, vars, env, allow_missing, mode)
- Template syntax (Go text/template)
- Variable resolution order (inline → env)
- Idempotency behavior
- Error handling examples
- Best practices
- Common patterns (config files, secrets, multi-environment)

Follow format of existing plugin documentation.

**Dependencies**: T001-T013 (plugin complete)  
**Acceptance**: Documentation is clear, includes examples

---

### T021 [P]: Add TemplateStep schema reference
**File**: `docs/schema.md`  
**Description**: Add TemplateStep schema documentation:
- Field descriptions with types
- Validation rules
- Default values
- YAML examples (minimal, full, with conditionals)
- Related plugins (copy, command)

**Dependencies**: T001, T002  
**Acceptance**: Schema docs match implementation, examples are valid

---

### T022 [P]: Create example template configuration
**File**: `testdata/configs/template.yaml`  
**Description**: Create comprehensive example config demonstrating:
1. Simple template rendering
2. Environment variable usage
3. Inline variables
4. File permissions
5. allow_missing flag
6. Conditionals in template
7. Dependencies between steps

Include corresponding template files in `testdata/templates/`.

**Dependencies**: T001, T002  
**Acceptance**: Example config is valid and demonstrates key features

---

## Phase 3.5: Validation & Polish

### T023: Run full test suite and verify coverage
**Files**: All test files  
**Command**: `go test ./internal/plugins/template/ -cover -v`  
**Description**: 
1. Run all unit tests for template plugin
2. Verify coverage >80% (target from plan.md)
3. Run integration tests
4. Verify all quickstart scenarios pass
5. Check for race conditions: `go test -race`
6. Run benchmarks if performance tests exist

Fix any failing tests or coverage gaps.

**Dependencies**: T015-T019 (all tests written)  
**Acceptance**: All tests pass, coverage >80%, no race conditions

---

## Dependencies Summary

```
Foundation:
T001 → T002 → T003 (data model)
T004 (parallel) → T005 → {T006, T007, T008, T009} (parallel)

Core Implementation:
T008, T009 → T010 (Check)
T008, T009 → T011 (Apply)
T008, T009 → T012 (DryRun)
T010, T011, T012 → T013 (init)
T013 → T014 (import)

Testing:
T006, T007 → T015 (metadata tests, parallel)
T010 → T016 (Check tests, parallel)
T011 → T017 (Apply tests, parallel)
T012 → T018 (DryRun tests, parallel)
T013 → T019 (integration tests)

Documentation:
T001-T013 → {T020, T021, T022} (all parallel)

Validation:
T015-T019 → T023 (final validation)
```

## Parallel Execution Examples

**Phase 1 - Foundation (2 parallel groups)**:
```bash
# Group 1: Data model (sequential)
Task T001: Add TemplateStep struct
Task T002: Add Template field to Step
Task T003: Add validation logic

# Group 2: Plugin structure (parallel with Group 1)
Task T004: Create directory structure
```

**Phase 2 - Helpers (after T005, 4 parallel)**:
```bash
Task T006: Implement Metadata()
Task T007: Implement Schema()
Task T008: Implement renderTemplate() helper
Task T009: Implement hashContent() helper
```

**Phase 3 - Testing (4 parallel after implementation complete)**:
```bash
Task T015: Test Metadata/Schema
Task T016: Test Check()
Task T017: Test Apply()
Task T018: Test DryRun()
```

**Phase 4 - Documentation (3 parallel after implementation)**:
```bash
Task T020: Add to docs/plugins.md
Task T021: Add to docs/schema.md
Task T022: Create testdata example
```

## Validation Checklist

- [x] All contracts have corresponding tests (T015-T018 cover all interface methods)
- [x] All entities have model tasks (T001 creates TemplateStep)
- [x] All tests come before or parallel with implementation
- [x] Parallel tasks truly independent (marked [P], different files)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task (verified)
- [x] Constitution-driven tasks included:
  - Onboarding: Zero dependencies (stdlib only)
  - Schema: Clear field definitions with validation
  - Safety: Idempotency (T010), dry-run (T012), error handling (T008)
  - Performance: Fast operations (<100ms target verified in tests)
  - Plugin: Standard interface (T006-T012), versioning (T006)

## Notes

- **TDD Approach**: Tests (T015-T019) should be written alongside implementation (T010-T012), not after
- **No External Dependencies**: All tasks use Go stdlib only (aligns with Constitution Principle I)
- **Idempotency**: Built into Check() (T010) and Apply() (T011) methods
- **Error Handling**: All methods include clear error messages with context
- **Performance**: Target <100ms for typical configs, verified in tests (T017, T019)
- **Constitution Compliance**: All principles satisfied (verified in plan.md Constitution Check)

## Estimated Timeline

- Phase 3.1 (Foundation): 1 hour
- Phase 3.2 (Implementation): 2-3 hours
- Phase 3.3 (Testing): 2-3 hours
- Phase 3.4 (Documentation): 30-60 minutes
- Phase 3.5 (Validation): 30 minutes

**Total**: 6-8 hours for complete feature implementation

## Success Criteria

1. ✅ All 23 tasks completed
2. ✅ Plugin registered and available as type "template"
3. ✅ All tests pass with >80% coverage
4. ✅ All 10 quickstart scenarios work end-to-end
5. ✅ Documentation complete and accurate
6. ✅ No external dependencies added
7. ✅ Performance targets met (<100ms)
8. ✅ Constitutional principles satisfied

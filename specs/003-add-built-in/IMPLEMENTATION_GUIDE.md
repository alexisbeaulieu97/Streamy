# Implementation Guide: line_in_file Plugin

**Date**: October 4, 2025  
**Status**: Ready to continue after T001-T006  
**Issue Resolved**: Interface alignment (use existing `Metadata/Check/Apply/DryRun` interface)

---

## Quick Reference: Existing Plugin Pattern

### 1. Plugin Registration Pattern
Study: `internal/plugins/command/command.go`

```go
package lineinfileplugin

import (
	"context"
	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type lineInFilePlugin struct{}

// New creates a new line_in_file plugin instance.
func New() plugin.Plugin {
	return &lineInFilePlugin{}
}

func init() {
	if err := plugin.RegisterPlugin("line_in_file", New()); err != nil {
		panic(err)
	}
}
```

### 2. Implement Plugin Interface

```go
func (p *lineInFilePlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "line-in-file",
		Version: "1.0.0",
		Type:    "line_in_file",  // ← This is what shows in config.Step.Type
	}
}

func (p *lineInFilePlugin) Schema() interface{} {
	return config.LineInFileStep{}  // ← Returns config struct for schema generation
}

func (p *lineInFilePlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	cfg := step.LineInFile
	if cfg == nil {
		return false, streamyerrors.NewValidationError(step.ID, "line_in_file configuration missing", nil)
	}
	
	// Return true if file already in desired state (idempotency check)
	// Return false if changes needed
	// Return error on validation/execution problems
	
	// TODO: Implement idempotency check logic
	return false, nil
}

func (p *lineInFilePlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg := step.LineInFile
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "line_in_file configuration missing", nil)
	}
	
	// Validate configuration first
	// Read file state
	// Perform modifications (append/replace/remove)
	// Create backup if requested
	// Write atomically
	// Return StepResult
	
	// TODO: Implement execution logic
	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusChanged,
		Message: "line modified",
	}, nil
}

func (p *lineInFilePlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg := step.LineInFile
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "line_in_file configuration missing", nil)
	}
	
	// Same logic as Apply but:
	// - Don't write files
	// - Don't create backups
	// - Generate diff preview
	// - Return diff in result
	
	// TODO: Implement dry-run logic
	return &model.StepResult{
		StepID:     step.ID,
		Status:     model.StatusChanged,
		Message:    "would modify line",
		DiffOutput: "diff preview here",
	}, nil
}
```

### 3. Configuration Changes Required

#### File: `internal/config/types.go`

**Step 1**: Add config struct (after `TemplateStep`):
```go
// LineInFileStep manages individual lines in text files.
type LineInFileStep struct {
	File              string `yaml:"file" validate:"required"`
	Line              string `yaml:"line" validate:"required"`
	State             string `yaml:"state,omitempty"`
	Match             string `yaml:"match,omitempty"`
	OnMultipleMatches string `yaml:"on_multiple_matches,omitempty"`
	Backup            bool   `yaml:"backup,omitempty"`
	BackupDir         string `yaml:"backup_dir,omitempty"`
	Encoding          string `yaml:"encoding,omitempty"`
}
```

**Step 2**: Add field to `Step` struct:
```go
type Step struct {
	ID        string   `yaml:"id" validate:"required,step_id"`
	Name      string   `yaml:"name,omitempty"`
	Type      string   `yaml:"type" validate:"required,oneof=package repo symlink copy command template line_in_file"`  // ← Add line_in_file
	DependsOn []string `yaml:"depends_on,omitempty"`
	Enabled   bool     `yaml:"enabled,omitempty"`

	Package    *PackageStep    `yaml:",inline,omitempty"`
	Repo       *RepoStep       `yaml:",inline,omitempty"`
	Symlink    *SymlinkStep    `yaml:",inline,omitempty"`
	Copy       *CopyStep       `yaml:",inline,omitempty"`
	Command    *CommandStep    `yaml:",inline,omitempty"`
	Template   *TemplateStep   `yaml:",inline,omitempty"`
	LineInFile *LineInFileStep `yaml:",inline,omitempty"`  // ← Add this field
}
```

**Step 3**: Update `UnmarshalYAML` switch (in `Step.UnmarshalYAML` method):
```go
switch base.Type {
case "package":
	// ... existing cases ...
case "template":
	var tmpl TemplateStep
	if err := value.Decode(&tmpl); err != nil {
		return err
	}
	s.Template = &tmpl
case "line_in_file":  // ← Add this case
	var lif LineInFileStep
	if err := value.Decode(&lif); err != nil {
		return err
	}
	s.LineInFile = &lif
}
```

**Step 4** (Optional): Add defaults in `UnmarshalYAML` for `LineInFileStep`:
```go
// UnmarshalYAML applies defaults for line_in_file steps.
func (l *LineInFileStep) UnmarshalYAML(value *yaml.Node) error {
	type rawLineInFile LineInFileStep
	var temp rawLineInFile
	if err := value.Decode(&temp); err != nil {
		return err
	}
	
	// Apply defaults
	if temp.State == "" {
		temp.State = "present"
	}
	if temp.OnMultipleMatches == "" {
		temp.OnMultipleMatches = "prompt"
	}
	if temp.Encoding == "" {
		temp.Encoding = "utf-8"
	}
	
	*l = LineInFileStep(temp)
	return nil
}
```

### 4. Plugin Registration

#### File: `cmd/streamy/plugins_import.go`

Add import:
```go
import (
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"  // ← Add this
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/template"
)
```

---

## Test Writing Adjustments

### For T004 (Name test):
```go
func TestLineInFile_Metadata(t *testing.T) {
	plugin := New()
	meta := plugin.Metadata()
	
	if meta.Type != "line_in_file" {
		t.Errorf("expected Type 'line_in_file', got %q", meta.Type)
	}
}
```

### For T005-T006 (Validation tests):
Validation happens inside `Apply()` and `DryRun()`, so test by calling those methods:

```go
func TestLineInFile_Validate_Valid(t *testing.T) {
	plugin := New()
	
	tests := []struct {
		name string
		step *config.Step
	}{
		{
			name: "valid present without match",
			step: &config.Step{
				ID:   "test",
				Type: "line_in_file",
				LineInFile: &config.LineInFileStep{
					File:  "/tmp/test.txt",
					Line:  "test line",
					State: "present",
				},
			},
		},
		// ... more test cases
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// DryRun should succeed for valid configs
			_, err := plugin.DryRun(context.Background(), tt.step)
			if err != nil && strings.Contains(err.Error(), "validation") {
				t.Errorf("validation failed for valid config: %v", err)
			}
		})
	}
}
```

---

## Next Steps (Unblocked)

1. **Update config types** (see Section 3 above):
   - Edit `internal/config/types.go`
   - Add `LineInFileStep` struct
   - Add `LineInFile` field to `Step`
   - Update `Type` validation to include `line_in_file`
   - Add `case "line_in_file"` to `UnmarshalYAML`

2. **Create plugin skeleton** (see Section 1-2 above):
   - File: `internal/plugins/lineinfile/lineinfile.go`
   - Implement all 5 interface methods with TODOs

3. **Fix T004-T006 tests**:
   - Update test to use `Metadata().Type` instead of `Name()`
   - Validation tests call `Apply()` or `DryRun()` and check for validation errors

4. **Continue with T007-T019** (Execute/DryRun/integration tests):
   - Test `Apply()` method behavior
   - Test `DryRun()` method behavior
   - Test `Check()` method for idempotency

5. **Implement core logic** (T020-T037):
   - Config validation
   - File operations
   - Line matching/manipulation
   - Diff generation
   - Full `Apply()`/`DryRun()`/`Check()` implementation

---

## Error Handling Examples

```go
// Validation error
if cfg.File == "" {
	return nil, streamyerrors.NewValidationError(step.ID, "file path is required", nil)
}

// Regex validation error
pattern, err := regexp.Compile(cfg.Match)
if err != nil {
	return nil, streamyerrors.NewValidationError(step.ID, 
		fmt.Sprintf("invalid match pattern: %v", err), err)
}

// Execution error (file operations)
if err := os.WriteFile(path, content, perm); err != nil {
	return nil, streamyerrors.NewExecutionError(step.ID, err)
}

// Success result
return &model.StepResult{
	StepID:  step.ID,
	Status:  model.StatusChanged,
	Message: fmt.Sprintf("modified %s", cfg.File),
}, nil

// No change (idempotent)
return &model.StepResult{
	StepID:  step.ID,
	Status:  model.StatusUnchanged,
	Message: "file already in desired state",
}, nil
```

---

## Summary

✅ **Interface alignment resolved** - use existing `Metadata/Check/Apply/DryRun` pattern  
✅ **No refactoring required** - adapts to existing plugin system  
✅ **Clear implementation path** - follow command plugin as reference  
✅ **Configuration changes minimal** - add struct + update Step type  

**You can now continue with T007-T019 tests and T020-T037 implementation!**

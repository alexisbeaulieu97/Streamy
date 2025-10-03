# Data Model: Create Streamy MVP

**Date**: 2025-10-03  
**Feature**: Streamy MVP - Configuration Schema and Domain Entities

## Overview

This document defines the data structures for Streamy's YAML configuration format and internal domain entities. All entities use Go struct tags for YAML parsing (`yaml:`) and validation (`validate:`).

---

## Configuration Schema

### 1. Config (Root)

**Purpose**: Top-level configuration structure representing a complete Streamy setup

**YAML Structure**: Config fields are placed **directly at the root level** of the YAML file (no wrapper key required).

**Fields**:
- `Version` (string, required): Schema version for compatibility (`"1.0"` for MVP)
- `Name` (string, required): Human-readable config name
- `Description` (string, optional): Purpose description
- `Settings` (Settings, optional): Global execution settings
- `Steps` ([]Step, required): List of setup steps
- `Validations` ([]Validation, optional): Post-execution checks

**Validation Rules**:
- Version must match `^[0-9]+\.[0-9]+$` (semantic versioning major.minor)
- Name must be 1-100 characters
- Steps must contain at least 1 step
- All step IDs must be unique

**Go Struct**:
```go
type Config struct {
    Version      string       `yaml:"version" validate:"required,semver"`
    Name         string       `yaml:"name" validate:"required,min=1,max=100"`
    Description  string       `yaml:"description,omitempty"`
    Settings     Settings     `yaml:"settings,omitempty"`
    Steps        []Step       `yaml:"steps" validate:"required,min=1,dive"`
    Validations  []Validation `yaml:"validations,omitempty,dive"`
}
```

**YAML Example** (root-level fields):
```yaml
version: "1.0"
name: "Developer Environment"
description: "Full dev setup for new team members"
settings:
  parallel: 4
  timeout: 300
steps:
  - id: install_git
    type: package
    # ...
validations:
  - type: command_exists
    command: git
```

**Note**: Earlier spec.md drafts showed a `streamy:` wrapper key, but the final design uses **flat root-level structure** for simplicity (aligns with Constitution Principle II: Schema Clarity & Fun).

---

### 2. Settings

**Purpose**: Global execution configuration

**Fields**:
- `Parallel` (int, optional): Max concurrent steps (default: 4)
- `Timeout` (int, optional): Default step timeout in seconds (default: 300)
- `ContinueOnError` (bool, optional): Continue execution on step failure (default: false, MVP always false)
- `DryRun` (bool, optional): Preview mode without execution (default: false, overridden by CLI flag)
- `Verbose` (bool, optional): Detailed logging (default: false, overridden by CLI flag)

**Validation Rules**:
- Parallel must be 1-32 (reasonable concurrency bounds)
- Timeout must be 1-3600 seconds (1 second to 1 hour)

**Go Struct**:
```go
type Settings struct {
    Parallel        int  `yaml:"parallel,omitempty" validate:"omitempty,min=1,max=32"`
    Timeout         int  `yaml:"timeout,omitempty" validate:"omitempty,min=1,max=3600"`
    ContinueOnError bool `yaml:"continue_on_error,omitempty"`
    DryRun          bool `yaml:"dry_run,omitempty"`
    Verbose         bool `yaml:"verbose,omitempty"`
}
```

---

### 3. Step

**Purpose**: Individual setup action with dependency tracking

**Fields**:
- `ID` (string, required): Unique identifier for DAG references
- `Name` (string, optional): Human-readable label
- `Type` (string, required): Step type (package, repo, symlink, copy, command)
- `DependsOn` ([]string, optional): List of step IDs this step depends on
- `Enabled` (bool, optional): Whether step should execute (default: true)
- Type-specific fields (embedded based on Type)

**Validation Rules**:
- ID must match `^[a-z0-9_]+$` (lowercase alphanumeric + underscore)
- ID must be unique across all steps
- Type must be one of: package, repo, symlink, copy, command
- DependsOn IDs must reference existing steps (no forward refs)
- No circular dependencies allowed

**Go Struct**:
```go
type Step struct {
    ID        string   `yaml:"id" validate:"required,alphanum_underscore"`
    Name      string   `yaml:"name,omitempty"`
    Type      string   `yaml:"type" validate:"required,oneof=package repo symlink copy command"`
    DependsOn []string `yaml:"depends_on,omitempty"`
    Enabled   bool     `yaml:"enabled,omitempty"`
    
    // Type-specific fields (one of these will be populated based on Type)
    Package *PackageStep `yaml:",inline,omitempty"`
    Repo    *RepoStep    `yaml:",inline,omitempty"`
    Symlink *SymlinkStep `yaml:",inline,omitempty"`
    Copy    *CopyStep    `yaml:",inline,omitempty"`
    Command *CommandStep `yaml:",inline,omitempty"`
}
```

---

### 4. Step Type: Package

**Purpose**: Install system packages

**Fields**:
- `Packages` ([]string, required): List of package names to install

**Validation Rules**:
- Packages must contain at least 1 package
- Package names must be 1-100 characters

**Go Struct**:
```go
type PackageStep struct {
    Packages []string `yaml:"packages" validate:"required,min=1,dive,min=1,max=100"`
}
```

**YAML Example**:
```yaml
- id: install_git
  name: "Install Git"
  type: package
  packages:
    - git
    - curl
```

---

### 5. Step Type: Repo

**Purpose**: Clone git repositories

**Fields**:
- `URL` (string, required): Git repository URL (https or git protocol)
- `Destination` (string, required): Local path for clone
- `Branch` (string, optional): Branch to checkout (default: default branch)
- `Depth` (int, optional): Shallow clone depth (default: 0 = full clone)

**Validation Rules**:
- URL must be valid git URL (regex: `^(https?|git)://`)
- Destination must be non-empty path
- Depth must be 0 or positive

**Go Struct**:
```go
type RepoStep struct {
    URL         string `yaml:"url" validate:"required,url"`
    Destination string `yaml:"destination" validate:"required"`
    Branch      string `yaml:"branch,omitempty"`
    Depth       int    `yaml:"depth,omitempty" validate:"omitempty,min=0"`
}
```

**YAML Example**:
```yaml
- id: clone_dotfiles
  name: "Clone dotfiles"
  type: repo
  depends_on:
    - install_git
  url: "https://github.com/user/dotfiles.git"
  destination: "~/.dotfiles"
  branch: "main"
```

---

### 6. Step Type: Symlink

**Purpose**: Create symbolic links

**Fields**:
- `Source` (string, required): Path to source file/directory
- `Target` (string, required): Path where symlink should be created
- `Force` (bool, optional): Overwrite existing target (default: false)

**Validation Rules**:
- Source must be non-empty path
- Target must be non-empty path
- Source and Target must not be the same

**Go Struct**:
```go
type SymlinkStep struct {
    Source string `yaml:"source" validate:"required"`
    Target string `yaml:"target" validate:"required,nefield=Source"`
    Force  bool   `yaml:"force,omitempty"`
}
```

**YAML Example**:
```yaml
- id: link_vimrc
  name: "Symlink vimrc"
  type: symlink
  depends_on:
    - clone_dotfiles
  source: "~/.dotfiles/vimrc"
  target: "~/.vimrc"
```

---

### 7. Step Type: Copy

**Purpose**: Copy files or directories

**Fields**:
- `Source` (string, required): Path to source file/directory
- `Destination` (string, required): Path where file should be copied
- `Overwrite` (bool, optional): Overwrite existing destination (default: false)
- `Recursive` (bool, optional): Copy directory recursively (default: false)
- `PreserveMode` (bool, optional): Preserve file permissions (default: true)

**Validation Rules**:
- Source must be non-empty path
- Destination must be non-empty path
- Source and Destination must not be the same

**Go Struct**:
```go
type CopyStep struct {
    Source       string `yaml:"source" validate:"required"`
    Destination  string `yaml:"destination" validate:"required,nefield=Source"`
    Overwrite    bool   `yaml:"overwrite,omitempty"`
    Recursive    bool   `yaml:"recursive,omitempty"`
    PreserveMode bool   `yaml:"preserve_mode,omitempty"`
}
```

**YAML Example**:
```yaml
- id: copy_config
  name: "Copy app config"
  type: copy
  source: "./config/app.conf"
  destination: "~/.config/myapp/app.conf"
  overwrite: true
  preserve_mode: true
```

---

### 8. Step Type: Command

**Purpose**: Execute shell commands

**Fields**:
- `Command` (string, required): Shell command to execute
- `Check` (string, optional): Command to verify idempotency (exit 0 = already satisfied)
- `Shell` (string, optional): Explicit shell to use (default: auto-detect)
- `WorkDir` (string, optional): Working directory for command execution
- `Env` (map[string]string, optional): Additional environment variables

**Validation Rules**:
- Command must be non-empty
- Check must be non-empty if specified

**Go Struct**:
```go
type CommandStep struct {
    Command string            `yaml:"command" validate:"required,min=1"`
    Check   string            `yaml:"check,omitempty"`
    Shell   string            `yaml:"shell,omitempty"`
    WorkDir string            `yaml:"workdir,omitempty"`
    Env     map[string]string `yaml:"env,omitempty"`
}
```

**YAML Example**:
```yaml
- id: add_path_export
  name: "Add to PATH"
  type: command
  command: 'echo "export PATH=$PATH:~/bin" >> ~/.bashrc'
  check: 'grep "export PATH.*~/bin" ~/.bashrc'
```

---

## Validation Schema

### 9. Validation

**Purpose**: Post-execution verification check

**Fields**:
- `Type` (string, required): Validation type (command_exists, file_exists, path_contains)
- Type-specific fields (based on Type)

**Validation Rules**:
- Type must be one of: command_exists, file_exists, path_contains

**Go Struct**:
```go
type Validation struct {
    Type string `yaml:"type" validate:"required,oneof=command_exists file_exists path_contains"`
    
    // Type-specific fields
    CommandExists  *CommandExistsValidation  `yaml:",inline,omitempty"`
    FileExists     *FileExistsValidation     `yaml:",inline,omitempty"`
    PathContains   *PathContainsValidation   `yaml:",inline,omitempty"`
}
```

---

### 10. Validation Type: CommandExists

**Purpose**: Verify a command is available in PATH

**Fields**:
- `Command` (string, required): Command name to check

**Validation Rules**:
- Command must be non-empty

**Go Struct**:
```go
type CommandExistsValidation struct {
    Command string `yaml:"command" validate:"required"`
}
```

**YAML Example**:
```yaml
- type: command_exists
  command: git
```

---

### 11. Validation Type: FileExists

**Purpose**: Verify a file or directory exists

**Fields**:
- `Path` (string, required): File or directory path to check

**Validation Rules**:
- Path must be non-empty

**Go Struct**:
```go
type FileExistsValidation struct {
    Path string `yaml:"path" validate:"required"`
}
```

**YAML Example**:
```yaml
- type: file_exists
  path: "~/.vimrc"
```

---

### 12. Validation Type: PathContains

**Purpose**: Verify a file contains specific text or regex pattern

**Fields**:
- `File` (string, required): File path to search
- `Text` (string, required): Text or regex pattern to find

**Validation Rules**:
- File must be non-empty
- Text must be non-empty

**Go Struct**:
```go
type PathContainsValidation struct {
    File string `yaml:"file" validate:"required"`
    Text string `yaml:"text" validate:"required"`
}
```

**YAML Example**:
```yaml
- type: path_contains
  file: "~/.bashrc"
  text: "export PATH.*~/bin"
```

---

## Internal Domain Entities

### 13. DAG (Directed Acyclic Graph)

**Purpose**: Internal representation of step execution order

**Fields**:
- `Nodes` (map[string]*DAGNode): Step ID to node mapping
- `Levels` ([][]string): Execution levels (level 0 = no dependencies, level 1 = depends on level 0, etc.)

**Relationships**:
- Each DAGNode contains step ID, dependencies, and dependents
- Levels computed via topological sort

**Go Struct**:
```go
type DAG struct {
    Nodes  map[string]*DAGNode
    Levels [][]string
}

type DAGNode struct {
    ID         string
    Step       *Step
    DependsOn  []*DAGNode
    Dependents []*DAGNode
}
```

**State Transitions**:
1. Build: Parse config → create nodes → link dependencies
2. Validate: Detect cycles → topological sort → compute levels
3. Execute: For each level, dispatch to worker pool → wait → next level

---

### 14. ExecutionContext

**Purpose**: Runtime context for step execution

**Fields**:
- `Config` (*Config): Full configuration
- `DryRun` (bool): Preview mode flag
- `Verbose` (bool): Detailed logging flag
- `WorkerPool` (chan struct{}): Semaphore for parallel execution
- `Results` (map[string]*StepResult): Step ID to result mapping

**Go Struct**:
```go
type ExecutionContext struct {
    Config     *Config
    DryRun     bool
    Verbose    bool
    WorkerPool chan struct{}
    Results    map[string]*StepResult
    Logger     *zerolog.Logger
}
```

---

### 15. StepResult

**Purpose**: Outcome of step execution

**Fields**:
- `StepID` (string): Step identifier
- `Status` (string): pending, running, success, skipped, failed
- `Message` (string): Human-readable status message
- `Error` (error): Error if failed
- `Duration` (time.Duration): Execution time
- `Timestamp` (time.Time): When step completed

**Go Struct**:
```go
type StepResult struct {
    StepID    string
    Status    string
    Message   string
    Error     error
    Duration  time.Duration
    Timestamp time.Time
}
```

**Status Transitions**:
- `pending` → `running` → `success` | `skipped` | `failed`

---

### 16. Plugin Interface

**Purpose**: Contract for all step type implementations

**Methods**:
- `Metadata() PluginMetadata`: Plugin name, version, supported types
- `Schema() interface{}`: Step-specific configuration schema
- `Check(ctx context.Context, step *Step) (bool, error)`: Idempotency check
- `Apply(ctx context.Context, step *Step) (*StepResult, error)`: Execute step
- `DryRun(ctx context.Context, step *Step) (*StepResult, error)`: Preview step

**Go Interface**:
```go
type Plugin interface {
    Metadata() PluginMetadata
    Schema() interface{}
    Check(ctx context.Context, step *Step) (bool, error)
    Apply(ctx context.Context, step *Step) (*StepResult, error)
    DryRun(ctx context.Context, step *Step) (*StepResult, error)
}

type PluginMetadata struct {
    Name    string
    Version string
    Type    string
}
```

---

## Validation Logic

### Schema Validation Flow

1. **Parse YAML**: `yaml.Unmarshal()` into Config struct
2. **Validate struct**: `validator.Validate()` on Config (checks required, min, max, regex, etc.)
3. **Cross-field validation**:
   - All step IDs unique
   - All DependsOn IDs reference existing steps
   - No dependency cycles
4. **Report errors**: Collect all validation errors with field paths and YAML line numbers

### DAG Validation Flow

1. **Build graph**: Create DAGNode for each step, link dependencies
2. **Cycle detection**: Depth-first search for back edges
3. **Topological sort**: Kahn's algorithm to compute execution levels
4. **Error reporting**: List cycle participants if detected

---

## Entity Relationships

```
Config
├── Settings (1:1)
├── Steps (1:N)
│   ├── PackageStep (1:0..1)
│   ├── RepoStep (1:0..1)
│   ├── SymlinkStep (1:0..1)
│   ├── CopyStep (1:0..1)
│   └── CommandStep (1:0..1)
└── Validations (1:N)
    ├── CommandExistsValidation (1:0..1)
    ├── FileExistsValidation (1:0..1)
    └── PathContainsValidation (1:0..1)

DAG
├── Nodes (map[string]*DAGNode)
└── Levels ([][]string)

ExecutionContext
├── Config (1:1)
├── WorkerPool (channel)
└── Results (map[string]*StepResult)

Plugin (interface)
└── Implementations: PackagePlugin, RepoPlugin, SymlinkPlugin, CopyPlugin, CommandPlugin
```

---

## Conclusion

This data model provides:
- **Clear schema**: YAML structure with validation rules
- **Type safety**: Go structs with compile-time checking
- **Extensibility**: Plugin interface supports new step types
- **Validation**: Declarative rules via struct tags + custom DAG validation
- **Execution tracking**: StepResult captures outcome and timing

All entities align with constitutional principles (schema clarity, plugin-centric architecture, safety defaults).

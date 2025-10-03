# Quickstart: Create Streamy MVP

**Date**: 2025-10-03  
**Feature**: Streamy MVP - Example Workflows and Integration Scenarios

## Overview

This document provides example workflows demonstrating Streamy's capabilities, from simple single-step configs to complex multi-dependency setups. Each example includes the YAML config, expected output, and verification steps.

---

## Example 1: Simple Package Installation

**Scenario**: Install git and curl on a fresh Ubuntu system

**Config**: `examples/01-simple-packages.yaml`

```yaml
version: "1.0"
name: "Basic Dev Tools"
description: "Install essential development tools"

steps:
  - id: install_git
    name: "Install Git"
    type: package
    packages:
      - git
      
  - id: install_curl
    name: "Install curl"
    type: package
    packages:
      - curl

validations:
  - type: command_exists
    command: git
    
  - type: command_exists
    command: curl
```

**Execution**:
```bash
$ streamy apply examples/01-simple-packages.yaml

Streamy v1.0.0 - Environment Setup Tool
Config: Basic Dev Tools

┌─────────────────────────────────────────┐
│ Parsing configuration...            [✓] │
│ Building dependency graph...        [✓] │
│ Validating config...                [✓] │
└─────────────────────────────────────────┘

Execution Plan:
  Level 0 (2 steps, parallel):
    ├─ install_git: Install Git
    └─ install_curl: Install curl

┌─────────────────────────────────────────┐
│ Executing Steps                         │
├─────────────────────────────────────────┤
│ [RUNNING] install_git                   │
│ [RUNNING] install_curl                  │
│ [SUCCESS] install_git (2.3s)            │
│ [SUCCESS] install_curl (1.8s)           │
└─────────────────────────────────────────┘

Running Validations...
  [✓] command_exists: git
  [✓] command_exists: curl

Summary: 2 steps executed, 2 succeeded, 0 failed, 0 skipped
Validations: 2 passed, 0 failed
Total time: 3.1s
```

**Verification**:
```bash
$ git --version
git version 2.34.1

$ curl --version
curl 7.81.0
```

---

## Example 2: Repository Clone with Dependencies

**Scenario**: Clone dotfiles repo (requires git)

**Config**: `examples/02-clone-repo.yaml`

```yaml
version: "1.0"
name: "Dotfiles Setup"
description: "Clone and link dotfiles"

steps:
  - id: ensure_git
    name: "Ensure Git is installed"
    type: package
    packages:
      - git
      
  - id: clone_dotfiles
    name: "Clone dotfiles repository"
    type: repo
    depends_on:
      - ensure_git
    url: "https://github.com/example/dotfiles.git"
    destination: "~/.dotfiles"
    branch: "main"

validations:
  - type: file_exists
    path: "~/.dotfiles"
    
  - type: file_exists
    path: "~/.dotfiles/README.md"
```

**Execution**:
```bash
$ streamy apply examples/02-clone-repo.yaml

Streamy v1.0.0 - Environment Setup Tool
Config: Dotfiles Setup

Execution Plan:
  Level 0 (1 step):
    └─ ensure_git: Ensure Git is installed
  Level 1 (1 step):
    └─ clone_dotfiles: Clone dotfiles repository

┌─────────────────────────────────────────┐
│ Executing Steps                         │
├─────────────────────────────────────────┤
│ [RUNNING] ensure_git                    │
│ [SKIPPED] ensure_git (already installed)│
│ [RUNNING] clone_dotfiles                │
│ [SUCCESS] clone_dotfiles (5.2s)         │
└─────────────────────────────────────────┘

Running Validations...
  [✓] file_exists: ~/.dotfiles
  [✓] file_exists: ~/.dotfiles/README.md

Summary: 2 steps executed, 1 succeeded, 0 failed, 1 skipped
Validations: 2 passed, 0 failed
Total time: 5.4s
```

**Verification**:
```bash
$ ls -la ~/.dotfiles
total 24
drwxr-xr-x  5 user user 4096 Oct  3 10:30 .
drwxr-xr-x 28 user user 4096 Oct  3 10:30 ..
drwxr-xr-x  8 user user 4096 Oct  3 10:30 .git
-rw-r--r--  1 user user 1234 Oct  3 10:30 README.md
-rw-r--r--  1 user user 5678 Oct  3 10:30 vimrc
```

---

## Example 3: Symlink Configuration Files

**Scenario**: Link dotfiles to home directory

**Config**: `examples/03-symlinks.yaml`

```yaml
version: "1.0"
name: "Dotfile Linking"
description: "Create symlinks for configuration files"

steps:
  - id: link_vimrc
    name: "Symlink vimrc"
    type: symlink
    source: "~/.dotfiles/vimrc"
    target: "~/.vimrc"
    
  - id: link_bashrc
    name: "Symlink bashrc"
    type: symlink
    source: "~/.dotfiles/bashrc"
    target: "~/.bashrc"
    force: true

validations:
  - type: file_exists
    path: "~/.vimrc"
    
  - type: file_exists
    path: "~/.bashrc"
```

**Execution** (with `--dry-run`):
```bash
$ streamy apply examples/03-symlinks.yaml --dry-run

Streamy v1.0.0 - Environment Setup Tool (DRY RUN)
Config: Dotfile Linking

Execution Plan:
  Level 0 (2 steps, parallel):
    ├─ link_vimrc: Symlink vimrc
    └─ link_bashrc: Symlink bashrc

┌─────────────────────────────────────────┐
│ Dry Run - Planned Actions               │
├─────────────────────────────────────────┤
│ [PLAN] link_vimrc                       │
│   → Create symlink:                     │
│      ~/.dotfiles/vimrc → ~/.vimrc       │
│   → Target does not exist (OK)          │
│                                         │
│ [PLAN] link_bashrc                      │
│   → Create symlink:                     │
│      ~/.dotfiles/bashrc → ~/.bashrc     │
│   → Target exists, will overwrite       │
│      (force: true)                      │
└─────────────────────────────────────────┘

Summary: 2 steps planned, 0 executed
Total time: 0.2s

Run without --dry-run to apply changes.
```

**Verification** (after actual run):
```bash
$ ls -la ~/.vimrc
lrwxrwxrwx 1 user user 25 Oct  3 10:35 ~/.vimrc -> /home/user/.dotfiles/vimrc

$ ls -la ~/.bashrc
lrwxrwxrwx 1 user user 26 Oct  3 10:35 ~/.bashrc -> /home/user/.dotfiles/bashrc
```

---

## Example 4: Copy Files with Permissions

**Scenario**: Copy application configuration

**Config**: `examples/04-copy-config.yaml`

```yaml
version: "1.0"
name: "App Configuration"
description: "Copy config files to system locations"

steps:
  - id: create_config_dir
    name: "Create config directory"
    type: command
    command: "mkdir -p ~/.config/myapp"
    check: "test -d ~/.config/myapp"
    
  - id: copy_app_config
    name: "Copy app config"
    type: copy
    depends_on:
      - create_config_dir
    source: "./configs/app.conf"
    destination: "~/.config/myapp/app.conf"
    preserve_mode: true
    overwrite: true

validations:
  - type: file_exists
    path: "~/.config/myapp/app.conf"
```

**Execution**:
```bash
$ streamy apply examples/04-copy-config.yaml --verbose

Streamy v1.0.0 - Environment Setup Tool
Config: App Configuration

[DEBUG] Loaded config: 2 steps, 1 validation
[DEBUG] DAG: 2 nodes, 2 levels
[DEBUG] Worker pool size: 4

Execution Plan:
  Level 0 (1 step):
    └─ create_config_dir: Create config directory
  Level 1 (1 step):
    └─ copy_app_config: Copy app config

┌─────────────────────────────────────────┐
│ [RUNNING] create_config_dir             │
│ [DEBUG] Checking idempotency...         │
│ [DEBUG] Check command: test -d ~/.config│
│ [DEBUG] Check result: false (will exec) │
│ [DEBUG] Executing: mkdir -p ~/.config/..│
│ [SUCCESS] create_config_dir (0.1s)      │
│                                         │
│ [RUNNING] copy_app_config               │
│ [DEBUG] Checking idempotency...         │
│ [DEBUG] Source: ./configs/app.conf      │
│ [DEBUG] Destination: ~/.config/myapp/.. │
│ [DEBUG] Destination exists, overwriting │
│ [DEBUG] Copying with preserved mode     │
│ [SUCCESS] copy_app_config (0.3s)        │
└─────────────────────────────────────────┘

Running Validations...
  [✓] file_exists: ~/.config/myapp/app.conf

Summary: 2 steps executed, 2 succeeded, 0 failed, 0 skipped
Validations: 1 passed, 0 failed
Total time: 0.5s
```

---

## Example 5: Shell Command Execution

**Scenario**: Add directory to PATH in shell config

**Config**: `examples/05-shell-commands.yaml`

```yaml
version: "1.0"
name: "Shell Configuration"
description: "Customize shell environment"

steps:
  - id: create_bin_dir
    name: "Create ~/bin directory"
    type: command
    command: "mkdir -p ~/bin"
    check: "test -d ~/bin"
    
  - id: add_to_path
    name: "Add ~/bin to PATH"
    type: command
    depends_on:
      - create_bin_dir
    command: 'echo "export PATH=\$PATH:~/bin" >> ~/.bashrc'
    check: 'grep -q "export PATH.*~/bin" ~/.bashrc'

validations:
  - type: path_contains
    file: "~/.bashrc"
    text: "export PATH.*~/bin"
```

**Execution**:
```bash
$ streamy apply examples/05-shell-commands.yaml

Streamy v1.0.0 - Environment Setup Tool
Config: Shell Configuration

Execution Plan:
  Level 0 (1 step):
    └─ create_bin_dir: Create ~/bin directory
  Level 1 (1 step):
    └─ add_to_path: Add ~/bin to PATH

┌─────────────────────────────────────────┐
│ [RUNNING] create_bin_dir                │
│ [SUCCESS] create_bin_dir (0.1s)         │
│ [RUNNING] add_to_path                   │
│ [SUCCESS] add_to_path (0.2s)            │
└─────────────────────────────────────────┘

Running Validations...
  [✓] path_contains: ~/.bashrc contains "export PATH.*~/bin"

Summary: 2 steps executed, 2 succeeded, 0 failed, 0 skipped
Validations: 1 passed, 0 failed
Total time: 0.4s
```

**Verification**:
```bash
$ tail -n 1 ~/.bashrc
export PATH=$PATH:~/bin

$ source ~/.bashrc
$ echo $PATH
/usr/local/bin:/usr/bin:/bin:~/bin
```

---

## Example 6: Complete Developer Environment

**Scenario**: Full onboarding setup with all step types

**Config**: `examples/06-full-environment.yaml`

```yaml
version: "1.0"
name: "Full Developer Environment"
description: "Complete dev setup for new team members"

settings:
  parallel: 4
  timeout: 600

steps:
  # Phase 1: Install tools
  - id: install_git
    name: "Install Git"
    type: package
    packages:
      - git
      - git-lfs
      
  - id: install_build_tools
    name: "Install build tools"
    type: package
    packages:
      - build-essential
      - cmake
      
  - id: install_editors
    name: "Install editors"
    type: package
    packages:
      - vim
      - nano
      
  # Phase 2: Clone repositories
  - id: clone_dotfiles
    name: "Clone dotfiles"
    type: repo
    depends_on:
      - install_git
    url: "https://github.com/company/dotfiles.git"
    destination: "~/.dotfiles"
    
  - id: clone_project
    name: "Clone main project"
    type: repo
    depends_on:
      - install_git
    url: "https://github.com/company/project.git"
    destination: "~/projects/project"
    
  # Phase 3: Link configurations
  - id: link_vimrc
    name: "Link vimrc"
    type: symlink
    depends_on:
      - clone_dotfiles
      - install_editors
    source: "~/.dotfiles/vimrc"
    target: "~/.vimrc"
    
  - id: link_gitconfig
    name: "Link gitconfig"
    type: symlink
    depends_on:
      - clone_dotfiles
      - install_git
    source: "~/.dotfiles/gitconfig"
    target: "~/.gitconfig"
    
  # Phase 4: Custom setup
  - id: create_workspace
    name: "Create workspace directories"
    type: command
    command: "mkdir -p ~/workspace/{projects,notes,tmp}"
    check: "test -d ~/workspace/projects"
    
  - id: setup_git_hooks
    name: "Install git hooks"
    type: copy
    depends_on:
      - clone_project
    source: "~/projects/project/.githooks/"
    destination: "~/projects/project/.git/hooks/"
    recursive: true
    preserve_mode: true

validations:
  # Tool validations
  - type: command_exists
    command: git
  - type: command_exists
    command: vim
  - type: command_exists
    command: cmake
    
  # Repository validations
  - type: file_exists
    path: "~/.dotfiles/README.md"
  - type: file_exists
    path: "~/projects/project/README.md"
    
  # Configuration validations
  - type: file_exists
    path: "~/.vimrc"
  - type: file_exists
    path: "~/.gitconfig"
    
  # Workspace validations
  - type: file_exists
    path: "~/workspace/projects"
```

**Execution**:
```bash
$ streamy apply examples/06-full-environment.yaml

Streamy v1.0.0 - Environment Setup Tool
Config: Full Developer Environment

┌─────────────────────────────────────────┐
│ Parsing configuration...            [✓] │
│ Building dependency graph...        [✓] │
│ Validating config...                [✓] │
└─────────────────────────────────────────┘

Execution Plan:
  Level 0 (3 steps, parallel):
    ├─ install_git
    ├─ install_build_tools
    └─ install_editors
  Level 1 (3 steps, parallel):
    ├─ clone_dotfiles (depends: install_git)
    ├─ clone_project (depends: install_git)
    └─ create_workspace
  Level 2 (2 steps, parallel):
    ├─ link_vimrc (depends: clone_dotfiles, install_editors)
    └─ link_gitconfig (depends: clone_dotfiles, install_git)
  Level 3 (1 step):
    └─ setup_git_hooks (depends: clone_project)

┌─────────────────────────────────────────┐
│ Executing Steps (4 workers)             │
├─────────────────────────────────────────┤
│ Level 0:                                │
│ [RUNNING] install_git                   │
│ [RUNNING] install_build_tools           │
│ [RUNNING] install_editors               │
│ [SUCCESS] install_editors (1.2s)        │
│ [SUCCESS] install_git (2.8s)            │
│ [SUCCESS] install_build_tools (3.1s)    │
│                                         │
│ Level 1:                                │
│ [RUNNING] clone_dotfiles                │
│ [RUNNING] clone_project                 │
│ [RUNNING] create_workspace              │
│ [SUCCESS] create_workspace (0.1s)       │
│ [SUCCESS] clone_dotfiles (4.5s)         │
│ [SUCCESS] clone_project (6.2s)          │
│                                         │
│ Level 2:                                │
│ [RUNNING] link_vimrc                    │
│ [RUNNING] link_gitconfig                │
│ [SUCCESS] link_vimrc (0.1s)             │
│ [SUCCESS] link_gitconfig (0.1s)         │
│                                         │
│ Level 3:                                │
│ [RUNNING] setup_git_hooks               │
│ [SUCCESS] setup_git_hooks (0.3s)        │
└─────────────────────────────────────────┘

Running Validations...
  [✓] command_exists: git
  [✓] command_exists: vim
  [✓] command_exists: cmake
  [✓] file_exists: ~/.dotfiles/README.md
  [✓] file_exists: ~/projects/project/README.md
  [✓] file_exists: ~/.vimrc
  [✓] file_exists: ~/.gitconfig
  [✓] file_exists: ~/workspace/projects

Summary: 10 steps executed, 10 succeeded, 0 failed, 0 skipped
Validations: 8 passed, 0 failed
Total time: 18.2s

✓ Environment setup complete!
```

---

## Example 7: Error Handling (Fail-Fast)

**Scenario**: Step fails, execution halts

**Config**: `examples/07-error-handling.yaml`

```yaml
version: "1.0"
name: "Error Handling Demo"

steps:
  - id: step1
    name: "Successful step"
    type: command
    command: "echo 'Step 1 success'"
    
  - id: step2
    name: "Failing step"
    type: command
    depends_on:
      - step1
    command: "exit 1"  # Intentionally fail
    
  - id: step3
    name: "Never reached"
    type: command
    depends_on:
      - step2
    command: "echo 'Step 3 success'"
```

**Execution**:
```bash
$ streamy apply examples/07-error-handling.yaml

Streamy v1.0.0 - Environment Setup Tool
Config: Error Handling Demo

Execution Plan:
  Level 0 (1 step):
    └─ step1: Successful step
  Level 1 (1 step):
    └─ step2: Failing step
  Level 2 (1 step):
    └─ step3: Never reached

┌─────────────────────────────────────────┐
│ [RUNNING] step1                         │
│ [SUCCESS] step1 (0.1s)                  │
│ [RUNNING] step2                         │
│ [FAILED] step2 (0.1s)                   │
│   Error: command exited with status 1  │
│   Command: exit 1                       │
│   Working dir: /home/user               │
│                                         │
│ ⚠ Execution halted due to failure      │
│   Failed step: step2                    │
│   Steps completed: 1/3                  │
│   Steps pending: 1 (step3)              │
└─────────────────────────────────────────┘

Summary: 2 steps attempted, 1 succeeded, 1 failed, 0 skipped
Validations: Skipped (failure occurred)
Total time: 0.3s

✗ Environment setup failed
Exit code: 1
```

---

## Example 8: Idempotency (Re-run Safety)

**Scenario**: Run same config twice, already-satisfied steps skipped

**Config**: `examples/08-idempotency.yaml`

```yaml
version: "1.0"
name: "Idempotency Demo"

steps:
  - id: install_git
    type: package
    packages:
      - git
      
  - id: create_dir
    type: command
    command: "mkdir -p ~/mydir"
    check: "test -d ~/mydir"
```

**First Run**:
```bash
$ streamy apply examples/08-idempotency.yaml

[RUNNING] install_git
[SUCCESS] install_git (2.1s)
[RUNNING] create_dir
[SUCCESS] create_dir (0.1s)

Summary: 2 steps executed, 2 succeeded, 0 failed, 0 skipped
Total time: 2.3s
```

**Second Run** (immediately after):
```bash
$ streamy apply examples/08-idempotency.yaml

[RUNNING] install_git
[SKIPPED] install_git (git already installed)
[RUNNING] create_dir
[SKIPPED] create_dir (check passed: test -d ~/mydir)

Summary: 2 steps attempted, 0 succeeded, 0 failed, 2 skipped
Total time: 0.4s
```

---

## Integration Test Scenarios

### Test 1: Malformed YAML

**Input**: `testdata/invalid.yaml`
```yaml
version: 1.0  # Should be string "1.0"
name: Invalid
steps:
  - id: test
    # Missing required 'type' field
```

**Expected Output**:
```
✗ Configuration validation failed

Error: Invalid YAML structure
  Line 1: Field 'version' must be a string
  Line 4: Field 'type' is required for step 'test'

Fix these errors and try again.
Exit code: 1
```

### Test 2: Circular Dependency

**Input**: `testdata/cycle.yaml`
```yaml
version: "1.0"
name: Circular Dependency
steps:
  - id: step_a
    type: command
    command: "echo A"
    depends_on:
      - step_b
  - id: step_b
    type: command
    command: "echo B"
    depends_on:
      - step_a
```

**Expected Output**:
```
✗ Dependency cycle detected

Cycle path: step_a → step_b → step_a

Remove circular dependencies and try again.
Exit code: 1
```

### Test 3: Missing Dependency Reference

**Input**: `testdata/missing-dep.yaml`
```yaml
version: "1.0"
name: Missing Dependency
steps:
  - id: step_a
    type: command
    command: "echo A"
    depends_on:
      - nonexistent_step
```

**Expected Output**:
```
✗ Configuration validation failed

Error: Invalid dependency reference
  Step 'step_a' depends on 'nonexistent_step', but no step with that ID exists

Fix the dependency reference and try again.
Exit code: 1
```

---

## Conclusion

These quickstart examples demonstrate:
- **Simple to complex**: From 2-step to 10-step configs
- **All step types**: package, repo, symlink, copy, command
- **Dependency management**: Serial and parallel execution
- **Safety features**: Dry-run, idempotency, fail-fast
- **Validation**: Post-execution checks
- **Error handling**: Clear messages, graceful failure
- **Idempotency**: Safe re-runs with skipped steps

Each example is executable and serves as both documentation and integration test fixture.

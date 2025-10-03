# Streamy YAML Schema Reference

Streamy configurations are YAML documents describing environment setup steps, validations, and global settings. This reference summarizes all supported fields and validation rules.

## Root Document

```yaml
version: "1.0"
name: "Developer Environment"
description: "Optional description"
settings:
  parallel: 4
  timeout: 300
steps:
  - id: install_git
    type: package
    packages:
      - git
validations:
  - type: command_exists
    command: git
```

### Fields

| Field        | Type     | Required | Notes |
|--------------|----------|----------|-------|
| `version`    | string   | ✅       | Semantic version `major.minor` (e.g., `"1.0"`). |
| `name`       | string   | ✅       | Human-readable name (1–100 chars). |
| `description`| string   | ❌       | Optional description displayed in the TUI.| 
| `settings`   | object   | ❌       | Execution defaults (see below). |
| `steps`      | array    | ✅       | At least one step. IDs must be unique. |
| `validations`| array    | ❌       | Post-execution checks. |

### Settings

```yaml
settings:
  parallel: 4      # 1-32, default 4
  timeout: 300     # seconds, 1-3600
  continue_on_error: false
  dry_run: false
  verbose: false
```

## Steps

Each step entry must contain:

| Field       | Type     | Required | Validation |
|-------------|----------|----------|------------|
| `id`        | string   | ✅       | Regex `^[a-z0-9_]+$`, unique per config |
| `type`      | string   | ✅       | One of `package`, `repo`, `symlink`, `copy`, `command` |
| `depends_on`| array    | ❌       | Existing step IDs, no cycles allowed |
| `enabled`   | bool     | ❌       | Defaults to `true` |

Type-specific fields are inlined. Only the relevant section must be present.

### package Step

```yaml
- id: install_tools
  type: package
  packages:
    - git
    - curl
  manager: apt
  update: true
```

| Field      | Type     | Required | Notes |
|------------|----------|----------|-------|
| `packages` | array    | ✅       | Each string 1–100 chars |
| `manager`  | string   | ❌       | Optional override (default apt) |
| `update`   | bool     | ❌       | Runs manager update first |

### repo Step

```yaml
- id: dotfiles
  type: repo
  depends_on: [install_tools]
  url: https://github.com/example/dotfiles.git
  destination: ~/.dotfiles
  branch: main
  depth: 1
```

| Field        | Type   | Required | Validation |
|--------------|--------|----------|------------|
| `url`        | string | ✅       | Valid URL (https/git) |
| `destination`| string | ✅       | Target path |
| `branch`     | string | ❌       | Optional branch |
| `depth`      | int    | ❌       | `>= 0` (0 = full clone) |

### symlink Step

```yaml
- id: link_vimrc
  type: symlink
  source: ~/.dotfiles/vimrc
  target: ~/.vimrc
  force: true
```

| Field  | Type   | Required | Notes |
|--------|--------|----------|-------|
| `source` | string | ✅ | Existing file/directory |
| `target` | string | ✅ | Link destination, must differ from source |
| `force`  | bool   | ❌ | Replace target if present |

### copy Step

```yaml
- id: copy_config
  type: copy
  source: ./config/app.conf
  destination: ~/.config/app.conf
  overwrite: true
  recursive: true
  preserve_mode: true
```

| Field          | Type   | Required | Notes |
|----------------|--------|----------|-------|
| `source`       | string | ✅       | File or directory |
| `destination`  | string | ✅       | Target path; must differ from source |
| `overwrite`    | bool   | ❌       | Allow replacing existing files |
| `recursive`    | bool   | ❌       | Required for directory copies |
| `preserve_mode`| bool   | ❌       | Defaults to `true` |

### command Step

```yaml
- id: run_script
  type: command
  command: ./scripts/setup.sh
  check: ./scripts/check.sh
  shell: /bin/bash
  workdir: ~/project
  env:
    STREAMY_ENV: prod
```

| Field    | Type     | Required | Notes |
|----------|----------|----------|-------|
| `command`| string   | ✅       | Shell command executed via detected shell |
| `check`  | string   | ❌       | Optional idempotency check command |
| `shell`  | string   | ❌       | Override default shell detection |
| `workdir`| string   | ❌       | Working directory |
| `env`    | map      | ❌       | Additional environment variables |

## Validations

Validations run after step execution.

### command_exists
```yaml
- type: command_exists
  command: git
```
- Ensures command is available in `$PATH`.

### file_exists
```yaml
- type: file_exists
  path: ~/.streamyrc
```
- Fails if `os.Stat` reports missing.

### path_contains
```yaml
- type: path_contains
  file: ~/.bashrc
  text: "export PATH"
```
- Checks file contents for regex pattern.

## Error Reporting

- Parsing errors include file and line information (`ParseError`).
- Validation failures identify the offending field (`ValidationError`).
- Dependency cycles produce descriptive paths (e.g., `a -> b -> a`).

## Best Practices

- Keep step IDs lowercase with underscores for readability.
- Group related steps via dependencies to maximize parallel execution.
- Use `check` or plugin-specific `Check` behaviour to keep configs idempotent.
- Leverage validations to catch misconfigurations early.
- Set `settings.parallel` based on workload characteristics; defaults to 4.

For end-to-end examples, see `testdata/configs/` and `docs/quickstart.md` (if present). Update this schema file when adding new step types or validation rules.

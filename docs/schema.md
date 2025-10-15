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
| `type`      | string   | ✅       | One of `package`, `repo`, `symlink`, `copy`, `command`, `template` |
| `depends_on`| array    | ❌       | Existing step IDs, no cycles allowed |
| `enabled`   | bool     | ❌       | Defaults to `true` |

Type-specific fields are inlined. Only the relevant section must be present. During execution the engine keeps these fields inside the step's `rawConfig`. Plugins should decode them with `step.DecodeConfig(&config.<StepType>Step{})`, and helpers/tests should populate them via `step.SetConfig(config.<StepType>Step{...})`.

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

### template Step

```yaml
- id: render_config
  type: template
  source: templates/app.conf.tmpl
  destination: config/app.conf
  vars:
    APP_NAME: Streamy
    ENVIRONMENT: production
  env: true
  allow_missing: false
  mode: 0644
```

| Field           | Type                | Required | Notes |
|-----------------|---------------------|----------|-------|
| `source`        | string              | ✅       | Template file path; must exist and be readable |
| `destination`   | string              | ✅       | Output file; must differ from `source` |
| `vars`          | map[string]string   | ❌       | Inline variables (keys must match `^[a-zA-Z_][a-zA-Z0-9_]*$`) |
| `env`           | bool                | ❌       | Defaults to `true`; include environment variables in context |
| `allow_missing` | bool                | ❌       | Defaults to `false`; render missing variables as empty strings when `true` |
| `mode`          | octal (0-0777)      | ❌       | Explicit destination permissions; falls back to source mode |

**Validation**

- `source` and `destination` are required and must differ.
- Variable names are validated against Go identifier rules.
- `mode` must be within `0`–`0777`.
- At least one variable source (inline or environment) should be supplied for non-static templates.

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

## Plugin Metadata (for Plugin Authors)

Plugin authors expose metadata to the runtime registry via `PluginMetadata()` (see `internal/plugin/metadata.go`). The structure mirrors the JSON below.

```json
{
  "name": "shell_profile",
  "version": "1.0.0",
  "api_version": "1.x",
  "dependencies": [
    {
      "name": "line_in_file",
      "version_constraint": "1.x"
    }
  ],
  "stateful": false,
  "description": "Provides shell profile composition using line_in_file."
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | string | ✅ | Unique plugin identifier; should match the step `type`. |
| `version` | string | ✅ | Semantic version `X.Y.Z`. |
| `api_version` | string | ✅ | Major version compatibility in `N.x` format. |
| `dependencies` | array | ❌ | Declared dependencies on other plugins (empty array by default). |
| `dependencies[].name` | string | ✅ | Name of required plugin. |
| `dependencies[].version_constraint` | string | ❌ | Major version constraint (`N.x`). |
| `stateful` | bool | ❌ | `true` if the registry should create per-dependent instances. |
| `description` | string | ❌ | Human-readable summary used in logs and diagnostics. |

The registry validates these fields at startup, detects missing or incompatible dependencies, and computes an initialisation order using the dependency graph.


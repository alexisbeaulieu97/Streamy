# Quickstart: line_in_file Plugin

**Feature**: Add Built-In Plugin: line_in_file  
**Date**: October 4, 2025  
**Purpose**: User-facing guide and validation scenarios

---

## Overview

The `line_in_file` plugin provides declarative, idempotent management of text file lines. It ensures specific lines exist or are removed, supports pattern-based replacement, and respects Streamy's safety features (dry-run, backups, verbose output).

**Use Cases**:
- Add environment variables to shell profiles
- Update configuration file values
- Remove outdated settings
- Ensure system config files contain required entries

---

## Installation

The `line_in_file` plugin is **built-in** to Streamy (version 0.3.0+). No installation required.

**Verify availability**:
```bash
streamy --version  # Should show 0.3.0 or higher
```

---

## Basic Examples

### Example 1: Add Line to File

Add a PATH export to your `.bashrc`:

```yaml
# config.yaml
steps:
  - id: add_path
    type: line_in_file
    file: "~/.bashrc"
    line: 'export PATH="$PATH:$HOME/bin"'
    state: present
```

**Run**:
```bash
streamy apply config.yaml
```

**Result**: Line added to end of `.bashrc` if not already present.

**Idempotency**: Running again produces no changes:
```bash
streamy apply config.yaml
# Output: ✓ add_path: No changes needed
```

---

### Example 2: Replace Matched Line

Update debug setting in app config:

```yaml
steps:
  - id: disable_debug
    type: line_in_file
    file: "/etc/myapp/config.ini"
    line: "debug=false"
    match: '^debug='
    state: present
```

**Behavior**:
- If file contains `debug=true` → Replaced with `debug=false`
- If file contains `debug=false` → No change (idempotent)
- If no `debug=` line exists → Line appended

---

### Example 3: Remove Line

Remove deprecated environment variable:

```yaml
steps:
  - id: remove_old_var
    type: line_in_file
    file: "~/.profile"
    match: '^export DEPRECATED_VAR='
    state: absent
```

**Behavior**:
- All lines matching pattern are removed
- If no matches → No change (idempotent)

---

## Advanced Features

### Multiple Match Handling

Control behavior when regex matches multiple lines:

```yaml
steps:
  - id: update_path_first
    type: line_in_file
    file: "~/.zshrc"
    line: 'export PATH="/usr/local/bin:$PATH"'
    match: '^export PATH='
    state: present
    on_multiple_matches: first  # Options: first, all, error, prompt
```

**Options**:
- `first`: Replace only the first match
- `all`: Replace all matching lines
- `error`: Fail if multiple matches found
- `prompt`: Ask user interactively (default)

---

### Backup Before Modification

Create timestamped backups of changed files:

```yaml
steps:
  - id: update_hosts
    type: line_in_file
    file: "/etc/hosts"
    line: "127.0.0.1 myapp.local"
    state: present
    backup: true
    backup_dir: "/var/backups/streamy"
```

**Result**: If file changes, creates:
```
/var/backups/streamy/hosts.20251004T153045.bak
```

**Restore from backup**:
```bash
sudo cp /var/backups/streamy/hosts.20251004T153045.bak /etc/hosts
```

---

### Custom Encoding

Handle non-UTF-8 files:

```yaml
steps:
  - id: update_legacy_config
    type: line_in_file
    file: "/legacy/app.conf"
    line: "charset=utf-8"
    encoding: "latin-1"
    state: present
```

**Supported encodings**: `utf-8` (default), `ascii`, `latin-1`, `latin-2`, `windows-1252`, `utf-16`

---

## Dry-Run Mode

Preview changes without modifying files:

```bash
streamy apply config.yaml --dry-run
```

**Example output**:
```
⊙ add_path: Would add 1 line to /home/user/.bashrc
  + export PATH="$PATH:$HOME/bin"

✓ disable_debug: Would replace 1 line in /etc/myapp/config.ini
  - debug=true
  + debug=false

⊙ remove_old_var: Would remove 1 line from /home/user/.profile
  - export DEPRECATED_VAR=old_value
```

**Verbose diff**:
```bash
streamy apply config.yaml --dry-run --verbose
```

Shows full unified diff with context lines.

---

## Complete Example: Shell Profile Setup

```yaml
# setup-shell.yaml
steps:
  # Create backup before any changes
  - id: backup_bashrc
    type: copy
    source: "~/.bashrc"
    dest: "~/.bashrc.backup"
    
  # Add custom bin to PATH
  - id: add_bin_path
    type: line_in_file
    file: "~/.bashrc"
    line: 'export PATH="$HOME/bin:$PATH"'
    match: '^export PATH=.*\$HOME/bin'
    state: present
    depends_on: [backup_bashrc]
    
  # Set editor preference
  - id: set_editor
    type: line_in_file
    file: "~/.bashrc"
    line: 'export EDITOR=vim'
    match: '^export EDITOR='
    state: present
    depends_on: [backup_bashrc]
    
  # Remove old JAVA_HOME
  - id: remove_old_java
    type: line_in_file
    file: "~/.bashrc"
    match: '^export JAVA_HOME=/usr/lib/jvm/java-8'
    state: absent
    depends_on: [backup_bashrc]
    
  # Add new JAVA_HOME
  - id: set_java_home
    type: line_in_file
    file: "~/.bashrc"
    line: 'export JAVA_HOME=/usr/lib/jvm/java-17'
    match: '^export JAVA_HOME='
    state: present
    depends_on: [remove_old_java]
```

**Run with dry-run first**:
```bash
streamy apply setup-shell.yaml --dry-run
# Review changes
streamy apply setup-shell.yaml
```

**DAG execution**: Steps run in dependency order, parallelizing where safe.

---

## Validation Scenarios

### Scenario 1: Fresh Shell Profile Setup

**Given**: Empty `~/.bashrc`

**Config**:
```yaml
steps:
  - id: setup_path
    type: line_in_file
    file: "~/.bashrc"
    line: 'export PATH="$HOME/.local/bin:$PATH"'
```

**Expected Result**:
- File created with single line
- Exit code: 0
- Output: `✓ setup_path: Added 1 line`

**Verify**:
```bash
cat ~/.bashrc
# Should show: export PATH="$HOME/.local/bin:$PATH"

# Run again (idempotency)
streamy apply config.yaml
# Output: ✓ setup_path: No changes needed
```

---

### Scenario 2: Replace Debug Setting

**Given**: File `/tmp/app.conf` contains:
```
port=8080
debug=true
timeout=30
```

**Config**:
```yaml
steps:
  - id: disable_debug
    type: line_in_file
    file: "/tmp/app.conf"
    line: "debug=false"
    match: '^debug='
    state: present
```

**Expected Result**:
- Line 2 replaced: `debug=true` → `debug=false`
- Other lines unchanged
- File permissions preserved
- Exit code: 0

**Verify**:
```bash
cat /tmp/app.conf
# Should show:
# port=8080
# debug=false
# timeout=30
```

---

### Scenario 3: Remove Multiple Matches

**Given**: File `/tmp/profile` contains:
```
export VAR1=value1
export OLD_VAR=value2
export VAR2=value3
export OLD_VAR=value4
```

**Config**:
```yaml
steps:
  - id: cleanup
    type: line_in_file
    file: "/tmp/profile"
    match: '^export OLD_VAR='
    state: absent
```

**Expected Result**:
- Lines 2 and 4 removed
- Exit code: 0
- Output: `✓ cleanup: Removed 2 lines`

**Verify**:
```bash
cat /tmp/profile
# Should show:
# export VAR1=value1
# export VAR2=value3
```

---

### Scenario 4: Multiple Matches with Prompt

**Given**: File contains:
```
export PATH=/usr/bin:$PATH
export PATH=/usr/local/bin:$PATH
```

**Config**:
```yaml
steps:
  - id: update_path
    type: line_in_file
    file: "/tmp/paths"
    line: 'export PATH=/opt/bin:$PATH'
    match: '^export PATH='
    on_multiple_matches: prompt
```

**Expected Result**:
- User prompted:
  ```
  Multiple matches found for pattern '^export PATH=' (2 matches)
  Options:
    first - Replace only the first match
    all   - Replace all matching lines
    error - Fail with error
  Choice:
  ```
- If user selects `first`: Only line 1 replaced
- If user selects `all`: Both lines replaced

**Non-interactive mode** (CI/CD):
```bash
streamy apply config.yaml  # Error: "interactive prompt required but not in TTY"
```

**Solution for automation**:
```yaml
on_multiple_matches: first  # or 'all' or 'error'
```

---

### Scenario 5: Backup Verification

**Given**: File `/tmp/important.conf` exists

**Config**:
```yaml
steps:
  - id: update_config
    type: line_in_file
    file: "/tmp/important.conf"
    line: "new_setting=value"
    state: present
    backup: true
```

**Expected Result**:
- Original file backed up to `/tmp/important.conf.20251004T153045.bak`
- Modified file contains new line
- Backup has identical permissions to original

**Verify**:
```bash
ls -la /tmp/important.conf*
# -rw-r--r-- important.conf
# -rw-r--r-- important.conf.20251004T153045.bak

diff /tmp/important.conf /tmp/important.conf.20251004T153045.bak
# Shows added line
```

---

### Scenario 6: Encoding Handling

**Given**: File `/tmp/legacy.txt` encoded in Latin-1

**Config**:
```yaml
steps:
  - id: add_line_latin1
    type: line_in_file
    file: "/tmp/legacy.txt"
    line: "café=value"
    encoding: "latin-1"
    state: present
```

**Expected Result**:
- File read and written in Latin-1 encoding
- Special characters (é) preserved correctly
- No encoding corruption

**Verify**:
```bash
file -i /tmp/legacy.txt
# Should show: charset=iso-8859-1
```

---

## Error Handling

### Permission Denied

**Config**:
```yaml
steps:
  - id: update_system_file
    type: line_in_file
    file: "/etc/hosts"
    line: "127.0.0.1 myapp"
    state: present
```

**Running as non-root**:
```bash
streamy apply config.yaml
# ✗ update_system_file: Failed
#   Error: failed to read file /etc/hosts: permission denied
#   Suggestion: Run with sudo or adjust file permissions
```

**Solution**:
```bash
sudo streamy apply config.yaml
```

---

### Invalid Regex

**Config**:
```yaml
steps:
  - id: bad_pattern
    type: line_in_file
    file: "~/.bashrc"
    match: '[invalid(regex'
    state: absent
```

**Result**:
```bash
streamy apply config.yaml
# Error parsing config:
#   Step 'bad_pattern' validation failed
#   Field: match
#   Reason: invalid regex pattern: error parsing regexp: missing closing )
#   Location: config.yaml:5
```

---

### State: Absent Without Match

**Config**:
```yaml
steps:
  - id: invalid_absent
    type: line_in_file
    file: "~/.profile"
    line: "some line"
    state: absent
    # Missing 'match' field!
```

**Result**:
```bash
streamy apply config.yaml
# Error parsing config:
#   Step 'invalid_absent' validation failed
#   Field: match
#   Reason: required when state is absent
#   Location: config.yaml:2
```

---

## Performance Tips

1. **Large files**: Plugin streams content, but regex matching requires full scan
   - For multi-GB files, expect proportional execution time
   - Consider `--verbose` to monitor progress

2. **Dry-run optimization**: Always dry-run first for complex configs
   ```bash
   streamy apply config.yaml --dry-run  # Fast preview
   streamy apply config.yaml            # Actual execution
   ```

3. **Backup strategy**: Don't backup on every run (idempotent runs don't create backups)
   - Backup directory fills only when changes occur

4. **Regex performance**: Anchored patterns (`^`, `$`) are faster than unanchored
   ```yaml
   match: '^export PATH='  # Fast: anchored search
   match: 'export PATH'    # Slower: scans whole line
   ```

---

## Troubleshooting

### File Not Found

**Symptom**: `failed to read file: no such file or directory`

**Solutions**:
- For new files: Use `state: present` (auto-creates)
- For dependencies: Ensure file-creation step runs first via `depends_on`

---

### Interactive Prompt in CI/CD

**Symptom**: `interactive prompt required but not in TTY context`

**Solution**: Always specify `on_multiple_matches` in automated configs:
```yaml
on_multiple_matches: first  # or 'all' or 'error'
```

---

### Backup Directory Not Writable

**Symptom**: `failed to write backup: permission denied`

**Solutions**:
- Ensure `backup_dir` exists and is writable
- Or omit `backup_dir` to use file's directory

---

### Symlink Issues

**Symptom**: Changes not appearing where expected

**Explanation**: Plugin follows symlinks by default (modifies target)

**Verify symlink target**:
```bash
ls -l ~/.bashrc
# lrwxrwxrwx ... ~/.bashrc -> /actual/path/bashrc
```

---

## Next Steps

- **Documentation**: See `docs/plugins.md` for full plugin reference
- **Schema**: View JSON schema at `internal/plugins/lineinfile/schema.json`
- **Examples**: More examples in `testdata/configs/lineinfile/`
- **Integration**: Combine with `template`, `command`, and other plugins

---

## Summary

**Plugin Type**: `line_in_file`  
**Required Fields**: `file`, `line`  
**Optional Fields**: `state`, `match`, `on_multiple_matches`, `backup`, `backup_dir`, `encoding`  
**Idempotent**: ✅ Yes  
**Dry-Run Support**: ✅ Yes  
**Backup Support**: ✅ Yes (optional)  
**Dependencies**: None (built-in)

**Ready for Use**: ✅

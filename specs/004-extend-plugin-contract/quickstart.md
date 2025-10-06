# Quickstart Guide: Verification Lifecycle

**Feature**: Extend Plugin Contract with Verify Lifecycle  
**Audience**: Streamy users wanting to understand and use verification

---

## What is Verification?

Verification is a **read-only inspection** of your system to determine if it matches your declared Streamy configuration. Unlike `apply` (which changes your system), `verify` only **checks** and **reports** the current state.

**Use verification to:**
- 🔍 Audit environments before making changes
- 🎯 Detect configuration drift on existing systems
- ⚡ Skip unnecessary work by identifying satisfied steps
- ✅ Confirm your configuration matches expectations

---

## Quick Example

### Step 1: Create a Configuration

`config.yaml`:
```yaml
steps:
  - id: install-git
    type: package
    packages:
      - name: git
  
  - id: create-projects-dir
    type: command
    command: mkdir -p ~/projects
    verify: test -d ~/projects
  
  - id: link-dotfiles
    type: symlink
    source: ~/dotfiles/.gitconfig
    target: ~/.gitconfig
```

### Step 2: Run Verification

```bash
streamy verify config.yaml
```

### Step 3: Review Results

```
Verifying configuration: config.yaml (3 steps)

[1/3] install-git         ✔ satisfied (120ms)
[2/3] create-projects-dir ✖ missing (30ms)
[3/3] link-dotfiles       ⚠ drifted (50ms)

────────────────────────────────────────────────
Summary: 3 steps verified in 200ms
  ✔ 1 satisfied
  ✖ 1 missing
  ⚠ 1 drifted
  🚫 0 blocked
  ? 0 unknown
```

**Interpretation:**
- ✔ **satisfied**: `git` package is already installed → skip during apply
- ✖ **missing**: `~/projects` directory doesn't exist → needs creation
- ⚠ **drifted**: `.gitconfig` symlink exists but points to wrong location → needs fixing

---

## Understanding Verification Statuses

Verification returns one of five statuses:

| Status | Symbol | Meaning | Apply Behavior |
|--------|--------|---------|----------------|
| **satisfied** | ✔ | System matches configuration exactly | **Skip** (no work needed) |
| **missing** | ✖ | Resource doesn't exist | **Create/Install** |
| **drifted** | ⚠ | Resource exists but differs | **Update/Fix** |
| **blocked** | 🚫 | Cannot verify (error/permission) | **Attempt** (may fail) |
| **unknown** | ? | Verification not possible | **Execute** (safe default) |

---

## Common Workflows

### Workflow 1: Pre-Apply Audit

**Goal**: Understand what will change before running `apply`.

```bash
# Check current state
streamy verify config.yaml

# Review what needs work
# ... inspect output ...

# Apply only necessary changes
streamy apply config.yaml
```

**Benefit**: No surprises — you know exactly what will happen.

---

### Workflow 2: Drift Detection

**Goal**: Identify manual changes on an existing system.

```bash
# On a previously configured machine
streamy verify production-config.yaml
```

**Example Output**:
```
[1/10] install-nginx      ✔ satisfied
[2/10] configure-nginx    ⚠ drifted (config file modified)
[3/10] setup-ssl-cert     ✔ satisfied
...
```

**Action**: Use `--verbose` to see what changed:

```bash
streamy verify production-config.yaml --verbose
```

```diff
[2/10] configure-nginx    ⚠ drifted
      Message: file content differs (1 line changed)
      
      --- expected: templates/nginx.conf (rendered)
      +++ actual: /etc/nginx/nginx.conf
      @@ -12,7 +12,7 @@
       http {
           server {
               listen 80;
      -        server_name example.com;
      +        server_name staging.example.com;  # Manually changed!
           }
       }
```

**Benefit**: Spot unauthorized or forgotten manual edits.

---

### Workflow 3: Compliance Auditing

**Goal**: Verify multiple machines match the same configuration.

```bash
# On each machine
ssh server1 'streamy verify config.yaml --json' > server1-verify.json
ssh server2 'streamy verify config.yaml --json' > server2-verify.json
ssh server3 'streamy verify config.yaml --json' > server3-verify.json

# Compare results
diff server1-verify.json server2-verify.json
```

**Benefit**: Detect inconsistencies across fleet.

---

### Workflow 4: CI/CD Integration

**Goal**: Validate environment in continuous integration pipeline.

`.github/workflows/verify.yml`:
```yaml
name: Verify Environment
on: [push, pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Streamy
        run: curl -sSL https://get.streamy.dev | sh
      
      - name: Verify Configuration
        run: streamy verify .streamy/dev-config.yaml
      
      - name: Report Drift
        if: failure()
        run: streamy verify .streamy/dev-config.yaml --verbose
```

**Benefit**: Catch configuration drift in pull requests.

---

## Command Reference

### Basic Verification

```bash
streamy verify <config-file>
```

**Output**: Human-readable table with summary.

---

### Verbose Mode (Show Diffs)

```bash
streamy verify <config-file> --verbose
```

**Use When**: You want to see **what** changed for drifted steps.

**Example Output**:
```diff
[3/5] render-app-config  ⚠ drifted
      
      --- expected: templates/app.conf.tmpl
      +++ actual: /etc/app/config.conf
      @@ -2,3 +2,3 @@
       APP_NAME=MyApp
      -LOG_LEVEL=info
      +LOG_LEVEL=debug
```

---

### JSON Output (Automation)

```bash
streamy verify <config-file> --json
```

**Use When**: Parsing results programmatically (scripts, monitoring, CI/CD).

**Example Output**:
```json
{
  "config_file": "config.yaml",
  "summary": {
    "total_steps": 3,
    "satisfied": 1,
    "missing": 1,
    "drifted": 1,
    "blocked": 0,
    "unknown": 0,
    "duration_ms": 200
  },
  "results": [
    {
      "step_id": "install-git",
      "status": "satisfied",
      "message": "package git is installed (version 2.39.0)",
      "duration_ms": 120
    },
    ...
  ]
}
```

**Parsing Example**:
```bash
# Extract count of missing steps
jq '.summary.missing' verify-output.json

# List IDs of drifted steps
jq -r '.results[] | select(.status == "drifted") | .step_id' verify-output.json
```

---

### Custom Timeout

```bash
streamy verify <config-file> --timeout=60s
```

**Use When**: Some verification checks are slow (large files, network operations).

**Per-Step Override** (in config):
```yaml
- id: verify-large-repo
  type: repo
  path: /opt/monorepo
  verify_timeout: 120s  # Override global timeout
```

---

## Exit Codes

Verification returns exit codes for scripting:

| Exit Code | Meaning | Use Case |
|-----------|---------|----------|
| 0 | All satisfied | No work needed |
| 1 | Some steps need work | Missing/drifted/blocked/unknown steps found |
| 2 | Configuration error | Invalid YAML, schema violation |
| 3 | Execution error | Plugin crash, unexpected failure |

**Example Script**:
```bash
#!/bin/bash
streamy verify config.yaml

case $? in
  0)
    echo "✅ System is up to date"
    ;;
  1)
    echo "⚠️ System needs updates — run 'streamy apply'"
    ;;
  2)
    echo "❌ Configuration is invalid"
    exit 2
    ;;
  3)
    echo "❌ Verification failed unexpectedly"
    exit 3
    ;;
esac
```

---

## Plugin-Specific Verification

Each plugin type verifies differently:

### Package Plugin

**What it checks:**
- Are packages installed?
- Do versions match (if specified)?

**Example**:
```yaml
- id: install-tools
  type: package
  packages:
    - name: git
      version: "2.40.0"  # Checks exact version
    - name: curl         # Any version OK
```

**Possible Statuses**:
- ✔ satisfied: All packages installed (correct versions)
- ✖ missing: One or more packages not installed
- ⚠ drifted: Package installed but wrong version

---

### Symlink Plugin

**What it checks:**
- Does symlink exist?
- Does it point to the correct source?

**Example**:
```yaml
- id: link-config
  type: symlink
  source: ~/dotfiles/.bashrc
  target: ~/.bashrc
```

**Possible Statuses**:
- ✔ satisfied: Symlink exists and points to correct source
- ✖ missing: Symlink doesn't exist
- ⚠ drifted: Symlink exists but points elsewhere
- 🚫 blocked: Permission denied reading symlink

---

### Template Plugin

**What it checks:**
- Does destination file exist?
- Does rendered template match file content?

**Example**:
```yaml
- id: render-nginx-config
  type: template
  source: templates/nginx.conf.tmpl
  destination: /etc/nginx/nginx.conf
  vars:
    SERVER_NAME: example.com
```

**Possible Statuses**:
- ✔ satisfied: Rendered template matches destination file
- ✖ missing: Destination file doesn't exist
- ⚠ drifted: File exists but content differs (shows diff in verbose mode)
- 🚫 blocked: Permission denied reading destination

---

### Command Plugin

**What it checks:**
- Runs optional `verify` command (if specified)
- Exit code 0 = satisfied, non-zero = missing

**Example**:
```yaml
- id: start-service
  type: command
  command: systemctl start myservice
  verify: systemctl is-active myservice --quiet  # Verification command
```

**Possible Statuses**:
- ✔ satisfied: Verify command succeeded (exit 0)
- ✖ missing: Verify command failed (exit non-zero)
- ? unknown: No verify command specified
- 🚫 blocked: Verify command timed out or errored

**Note**: If no `verify` command specified, status is `unknown` (system will re-run command during apply for safety).

---

### Repo Plugin

**What it checks:**
- Does repository directory exist?
- Is remote URL correct?
- Is current branch correct?

**Example**:
```yaml
- id: clone-dotfiles
  type: repo
  url: https://github.com/user/dotfiles.git
  path: ~/dotfiles
  branch: main
```

**Possible Statuses**:
- ✔ satisfied: Repo exists, correct remote, correct branch
- ✖ missing: Directory doesn't exist
- ⚠ drifted: Repo exists but wrong branch or remote
- 🚫 blocked: Permission denied or directory is not a git repo

---

## Troubleshooting

### Q: Verification says "blocked" — what does that mean?

**A:** Verification couldn't complete due to an error (permission denied, file locked, network timeout). Check the error message for details.

**Example**:
```
[2/5] check-ssl-cert  🚫 blocked (80ms)
      Message: permission denied reading /etc/ssl/private/cert.key
```

**Solution**: Fix the underlying issue (grant permissions, unlock file) and re-run verify.

---

### Q: Why does a step show "unknown"?

**A:** The plugin cannot determine verification status. Most common for `command` type without a `verify` clause.

**Example**:
```yaml
- id: run-migration
  type: command
  command: ./migrate.sh
  # No 'verify' specified → status will be 'unknown'
```

**Solution**: Add a `verify` command if possible:
```yaml
- id: run-migration
  type: command
  command: ./migrate.sh
  verify: test -f /var/lib/migrations/completed
```

---

### Q: Verification is slow — how can I speed it up?

**A:** Increase timeout or optimize expensive checks.

**Global Timeout**:
```bash
streamy verify config.yaml --timeout=60s
```

**Per-Step Timeout**:
```yaml
- id: slow-check
  type: template
  source: large-template.tmpl
  destination: /etc/large-config
  verify_timeout: 120s  # Allow more time for this step
```

**Optimization**: For large files, plugins use checksums instead of full content comparison when possible.

---

### Q: Can I verify only specific steps?

**A:** Not in the initial release. Verification always checks all steps in dependency order.

**Future Enhancement**: Filtering by step ID or tags may be added based on user demand.

---

### Q: Does verification modify my system?

**A:** **No, never.** Verification is strictly read-only. It only inspects state and reports findings.

**Guarantee**: All plugins are contract-tested to ensure no side effects during verification.

---

## What's Next?

After running verification:

1. **All satisfied?** → Your system matches the config — no action needed!
2. **Found issues?** → Review drifted/missing steps, then run:
   ```bash
   streamy apply config.yaml
   ```
3. **Want to fix only specific issues?** → Edit config to remove satisfied steps (or wait for future filtering feature).

---

## Summary

| Action | Command | Use Case |
|--------|---------|----------|
| **Basic verification** | `streamy verify config.yaml` | Quick status check |
| **See what changed** | `streamy verify config.yaml --verbose` | Investigate drift |
| **Automation** | `streamy verify config.yaml --json` | CI/CD, scripting |
| **Slow checks** | `streamy verify config.yaml --timeout=60s` | Large repos, files |

**Remember**: Verification is your **safety check** before applying changes. Use it often!

---

**Next Steps**:
- Read the full [verification contract](contracts/cli-verify-contract.md) for advanced details
- Explore [plugin-specific verification](contracts/plugin-verify-contract.md) behavior
- Try verification on your own configurations

**Happy verifying! 🔍**

# Quickstart: Template Plugin

**Goal**: Verify the template plugin works end-to-end with variable substitution, idempotency, and error handling.

**Time**: ~5 minutes  
**Prerequisites**: Streamy binary built with template plugin

## Step 1: Create Template Files

Create a test directory:

```bash
mkdir -p /tmp/streamy-template-test/templates
cd /tmp/streamy-template-test
```

Create template file `templates/app.conf.tmpl`:

```
# Application Configuration
# Generated from template

app_name = {{.APP_NAME}}
environment = {{.ENVIRONMENT}}
debug = {{.DEBUG_MODE}}

{{if eq .ENVIRONMENT "production"}}
# Production settings
max_connections = 100
timeout = 30
{{else}}
# Development settings
max_connections = 10
timeout = 300
{{end}}

# Database
database_url = {{.DATABASE_URL}}
```

Create template file `templates/secret.txt.tmpl`:

```
API_KEY={{.API_KEY}}
SECRET={{.SECRET_TOKEN}}
```

## Step 2: Create Streamy Configuration

Create `streamy.yaml`:

```yaml
steps:
  # Test 1: Basic variable substitution with inline vars
  - id: render-app-config
    type: template
    source: templates/app.conf.tmpl
    destination: config/app.conf
    vars:
      APP_NAME: MyApp
      ENVIRONMENT: development
      DEBUG_MODE: "true"
      DATABASE_URL: postgres://localhost/mydb

  # Test 2: Environment variable substitution
  - id: render-secrets
    type: template
    source: templates/secret.txt.tmpl
    destination: config/secrets.txt
    mode: 0600
    # env: true is default, will read from environment

  # Test 3: Missing variable handling (should fail)
  - id: render-missing-var
    type: template
    source: templates/app.conf.tmpl
    destination: config/missing.conf
    vars:
      APP_NAME: TestApp
      ENVIRONMENT: test
      # Missing: DEBUG_MODE, DATABASE_URL
    enabled: false  # Disabled by default, enable to test failure

  # Test 4: Allow missing variables
  - id: render-optional
    type: template
    source: templates/app.conf.tmpl
    destination: config/optional.conf
    vars:
      APP_NAME: OptionalApp
    allow_missing: true
    enabled: false  # Disabled by default, enable to test
```

## Step 3: Set Environment Variables

```bash
export API_KEY="test-key-12345"
export SECRET_TOKEN="super-secret-token"
```

## Step 4: Run Dry-Run Mode

Preview what will be created:

```bash
streamy apply --dry-run
```

**Expected Output**:
```
[DRY RUN] Step: render-app-config
  Status: Would create config/app.conf
  Source: templates/app.conf.tmpl
  Variables: 4 defined

[DRY RUN] Step: render-secrets
  Status: Would create config/secrets.txt
  Source: templates/secret.txt.tmpl
  Variables: 2 from environment
  Mode: 0600
```

## Step 5: Apply Configuration

Execute the template rendering:

```bash
streamy apply
```

**Expected Output**:
```
✓ render-app-config: Created config/app.conf
✓ render-secrets: Created config/secrets.txt (mode: 0600)

Summary: 2 steps completed, 0 skipped, 0 failed
```

## Step 6: Verify Rendered Files

Check app config:

```bash
cat config/app.conf
```

**Expected Content**:
```
# Application Configuration
# Generated from template

app_name = MyApp
environment = development
debug = true

# Development settings
max_connections = 10
timeout = 300

# Database
database_url = postgres://localhost/mydb
```

Check secrets (with restricted permissions):

```bash
ls -l config/secrets.txt
cat config/secrets.txt
```

**Expected**:
- Permissions: `-rw-------` (0600)
- Content:
  ```
  API_KEY=test-key-12345
  SECRET=super-secret-token
  ```

## Step 7: Test Idempotency

Run apply again without changes:

```bash
streamy apply
```

**Expected Output**:
```
⊘ render-app-config: Skipped (no changes)
⊘ render-secrets: Skipped (no changes)

Summary: 0 steps completed, 2 skipped, 0 failed
```

**Verification**: No files were modified (check timestamps):

```bash
stat config/app.conf
```

## Step 8: Test Variable Override

Modify `streamy.yaml` to override environment variable:

```yaml
  - id: render-secrets
    type: template
    source: templates/secret.txt.tmpl
    destination: config/secrets.txt
    mode: 0600
    vars:
      API_KEY: overridden-key  # This should override env var
```

Run apply:

```bash
streamy apply
```

**Expected**: File updated with new API_KEY value:

```bash
cat config/secrets.txt
# API_KEY=overridden-key  (overridden)
# SECRET=super-secret-token (from env)
```

## Step 9: Test Missing Variable Error

Enable the failing step in `streamy.yaml`:

```yaml
  - id: render-missing-var
    type: template
    source: templates/app.conf.tmpl
    destination: config/missing.conf
    vars:
      APP_NAME: TestApp
      ENVIRONMENT: test
    enabled: true  # Enable to test
```

Run apply:

```bash
streamy apply
```

**Expected Output** (should fail):
```
✓ render-app-config: Skipped (no changes)
✓ render-secrets: Skipped (no changes)
✗ render-missing-var: FAILED
  Error: undefined variable "DEBUG_MODE"
  Template: templates/app.conf.tmpl, line 6
  Suggestion: Add to 'vars' map or set allow_missing: true

Summary: 0 steps completed, 2 skipped, 1 failed
```

## Step 10: Test Template Syntax Error

Create invalid template `templates/broken.tmpl`:

```
{{.VAR1}}
{{if .VAR2}  # Missing closing brace
{{.VAR3}}
```

Add step to `streamy.yaml`:

```yaml
  - id: test-syntax-error
    type: template
    source: templates/broken.tmpl
    destination: config/broken.txt
    vars:
      VAR1: value1
      VAR2: value2
      VAR3: value3
```

Run apply:

```bash
streamy apply
```

**Expected Output**:
```
✗ test-syntax-error: FAILED
  Error: template syntax error in templates/broken.tmpl
  Details: template: broken.tmpl:2:16: unexpected "}" in operand
  Line 2, column 16

Summary: 0 steps completed, 0 skipped, 1 failed
```

## Success Criteria

All tests pass if:

- ✅ Dry-run shows what would be created without making changes
- ✅ Apply creates files with correct rendered content
- ✅ Inline variables override environment variables
- ✅ Conditionals in templates work (if/else based on ENVIRONMENT)
- ✅ File permissions are set correctly (0600 for secrets)
- ✅ Idempotency: re-running apply skips unchanged files
- ✅ Missing variables cause clear errors with line numbers
- ✅ Template syntax errors show precise location (line/column)
- ✅ allow_missing: true permits optional variables

## Cleanup

```bash
rm -rf /tmp/streamy-template-test
```

## Troubleshooting

**Issue**: "template file not found"  
**Solution**: Ensure paths are relative to config file directory or use absolute paths

**Issue**: "permission denied writing destination"  
**Solution**: Check write permissions for destination directory, create if needed

**Issue**: "variable undefined" but variable is set in environment  
**Solution**: Check `env: true` is set (default), verify variable name matches exactly

**Issue**: Files keep getting rewritten (idempotency not working)  
**Solution**: Check for dynamic content (timestamps, random values) in template

## Next Steps

After completing this quickstart:

1. Review plugin implementation in `internal/plugins/template/`
2. Run full test suite: `go test ./internal/plugins/template/`
3. Add your own templates to real projects
4. Read full documentation in `docs/plugins.md`
5. Explore advanced features (template functions, conditionals, loops)

## Performance Validation

Time the operations to ensure they meet performance goals:

```bash
time streamy apply --dry-run  # Should be <50ms
time streamy apply            # Should be <100ms for these small configs
```

If times exceed targets, profile with:

```bash
streamy apply --profile
```

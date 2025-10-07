# Quickstart: Unified Plugin Interface Migration

**Purpose**: Validate that the unified plugin interface refactoring is complete and working correctly.

**Time to Complete**: ~15 minutes

**Prerequisites**:
- Go 1.21+ installed
- Streamy repository cloned on branch `006-unify-and-simplify`
- All dependencies available (`go mod download`)

---

## Step 1: Verify Core Types Compile

**What**: Ensure new interface and types are defined correctly.

```bash
cd /home/alexis/Projects/Streamy

# Check that new types compile
go build -o /dev/null ./internal/plugin/...
go build -o /dev/null ./internal/model/...
```

**Expected Output**:
```
# (No output = success)
```

**If it fails**: Check that `interface.go`, `evaluation_result.go`, and `errors.go` are properly defined.

---

## Step 2: Run Unit Tests for Core Types

**What**: Verify EvaluationResult and error types work correctly.

```bash
# Test model types
go test -v ./internal/model/...

# Test plugin error types
go test -v ./internal/plugin/ -run TestPluginError
```

**Expected Output**:
```
=== RUN   TestEvaluationResult
--- PASS: TestEvaluationResult (0.00s)
=== RUN   TestPluginErrorTypes
--- PASS: TestPluginErrorTypes (0.00s)
PASS
```

**If it fails**: Review error type implementations (Error(), StepID(), Unwrap() methods).

---

## Step 3: Verify One Plugin Compiles (lineinfile)

**What**: Confirm at least one plugin successfully implements the new interface.

```bash
# Build lineinfile plugin (reference implementation)
go build -o /dev/null ./internal/plugins/lineinfile/...
```

**Expected Output**:
```
# (No output = success)
```

**If it fails**: 
- Ensure Evaluate() method signature matches contract
- Ensure Apply() accepts EvaluationResult parameter
- Check that old Check()/DryRun()/Verify() methods are removed

---

## Step 4: Run Plugin Contract Tests

**What**: Validate that refactored plugins pass contract test suite.

```bash
# Run contract tests for lineinfile (first migrated plugin)
go test -v ./internal/plugins/lineinfile/ -run TestContract
```

**Expected Output**:
```
=== RUN   TestLineInFileContract
=== RUN   TestLineInFileContract/Metadata_is_stable
=== RUN   TestLineInFileContract/Schema_returns_struct
=== RUN   TestLineInFileContract/Evaluate_is_read-only
=== RUN   TestLineInFileContract/Evaluate_is_idempotent
=== RUN   TestLineInFileContract/Apply_is_idempotent
=== RUN   TestLineInFileContract/Error_types_are_correct
--- PASS: TestLineInFileContract (0.15s)
PASS
```

**If it fails**: Review the specific failing subtest and fix the plugin implementation.

---

## Step 5: Test Executor Integration (Verify Mode)

**What**: Ensure executor correctly uses Evaluate() in verify mode.

```bash
# Create a test config file
cat > /tmp/test-unified-plugin.yaml <<'EOF'
steps:
  - id: test-line
    type: line_in_file
    line_in_file:
      path: /tmp/test-file.txt
      line: "test line"
      state: present
EOF

# Create test file
echo "existing content" > /tmp/test-file.txt

# Run verify mode (should show drifted)
go run ./cmd/streamy verify --config /tmp/test-unified-plugin.yaml
```

**Expected Output**:
```
Verifying configuration...
✗ test-line: drifted
  Message: line not found in file
  
Summary: 0 satisfied, 1 drifted
```

**If it fails**:
- Check that executor calls plugin.Evaluate()
- Verify EvaluationResult.CurrentState is interpreted correctly
- Ensure verify mode does NOT call Apply()

---

## Step 6: Test Executor Integration (Dry-Run Mode)

**What**: Ensure executor shows preview without applying changes.

```bash
# Run dry-run mode
go run ./cmd/streamy apply --dry-run --config /tmp/test-unified-plugin.yaml
```

**Expected Output**:
```
Dry-run mode (no changes will be applied)...
→ test-line: would apply
  Message: line not found in file
  Diff:
    + test line
  
Summary: 0 would skip, 1 would apply
```

**If it fails**:
- Check that Diff field is displayed
- Verify dry-run mode does NOT call Apply()
- Ensure RequiresAction flag is checked correctly

---

## Step 7: Test Executor Integration (Apply Mode)

**What**: Ensure executor applies changes when needed.

```bash
# Run apply mode
go run ./cmd/streamy apply --config /tmp/test-unified-plugin.yaml

# Verify file was modified
cat /tmp/test-file.txt
```

**Expected Output**:
```
Applying configuration...
✓ test-line: applied
  Message: added line to /tmp/test-file.txt
  
Summary: 0 skipped, 1 applied, 0 failed
```

**File contents**:
```
existing content
test line
```

**If it fails**:
- Check that Apply() is called when RequiresAction is true
- Verify InternalData is passed from Evaluate() to Apply()
- Ensure Apply() modifies state correctly

---

## Step 8: Test Idempotency (Second Apply)

**What**: Verify that running apply again on satisfied state skips the step.

```bash
# Run apply again (file already has the line)
go run ./cmd/streamy apply --config /tmp/test-unified-plugin.yaml
```

**Expected Output**:
```
Applying configuration...
⊙ test-line: skipped
  Message: no changes required
  
Summary: 1 skipped, 0 applied, 0 failed
```

**If it fails**:
- Check that Evaluate() correctly detects satisfied state
- Verify RequiresAction is false when state matches
- Ensure Apply() is NOT called for satisfied state

---

## Step 9: Test Error Handling (Validation Error)

**What**: Verify structured error types work correctly.

```bash
# Create invalid config (missing required field)
cat > /tmp/test-invalid.yaml <<'EOF'
steps:
  - id: invalid
    type: line_in_file
    line_in_file:
      # path missing (required field)
      line: "test"
      state: present
EOF

# Run verify (should fail with ValidationError)
go run ./cmd/streamy verify --config /tmp/test-invalid.yaml 2>&1 | grep -i validation
```

**Expected Output**:
```
Error: Configuration validation failed for step invalid
  validation error: path is required
```

**If it fails**:
- Check that ValidationError is returned for config errors
- Verify error message includes step ID and actionable guidance
- Ensure executor recognizes ValidationError type

---

## Step 10: Run Full Test Suite

**What**: Verify all tests pass (unit + integration).

```bash
# Run all tests
go test ./... -v

# Check for any failures
echo "Exit code: $?"
```

**Expected Output**:
```
[... many test results ...]
PASS
Exit code: 0
```

**If it fails**:
- Review failing test output
- Focus on tests related to plugin interface, executor, or specific plugins
- Fix issues and re-run

---

## Step 11: Performance Benchmark (Optional)

**What**: Verify performance is within acceptable bounds (20% overhead).

```bash
# Run benchmarks
go test -bench=. ./internal/plugins/lineinfile/ -benchmem
```

**Expected Output**:
```
BenchmarkEvaluate-8    10000    120000 ns/op    2048 B/op    15 allocs/op
```

**Acceptance Criteria**:
- Evaluate() should be no more than 20% slower than old Check() method
- Memory allocations should be reasonable for operation type

---

## Step 12: Verify All 8 Plugins

**What**: Confirm all built-in plugins are migrated and working.

```bash
# Compile all plugins
for plugin in symlink copy lineinfile template package repo command internalexec; do
    echo "Testing $plugin..."
    go test -v ./internal/plugins/$plugin/ -run TestContract || echo "FAILED: $plugin"
done
```

**Expected Output**:
```
Testing symlink...
--- PASS: TestSymlinkContract (0.05s)
Testing copy...
--- PASS: TestCopyContract (0.08s)
[... etc for all 8 plugins ...]
```

**If any fail**: That plugin is not yet fully migrated to the new interface.

---

## Success Criteria

✅ **Quickstart is successful if**:
1. All core types compile without errors
2. At least one plugin (lineinfile) passes contract tests
3. Executor correctly handles verify/dry-run/apply modes
4. Apply mode is idempotent (second run skips satisfied steps)
5. Structured errors are caught and displayed correctly
6. Performance is within 20% budget
7. All 8 plugins eventually pass (may complete incrementally)

---

## Cleanup

```bash
# Remove test files
rm /tmp/test-unified-plugin.yaml
rm /tmp/test-invalid.yaml
rm /tmp/test-file.txt
```

---

## Next Steps After Quickstart

If all steps pass:
1. ✅ Core refactoring is complete
2. ✅ Ready to complete remaining plugin migrations
3. ✅ Ready to update documentation (docs/plugins.md)
4. ✅ Ready to merge feature branch

If steps fail:
1. Review specific failure in the step
2. Check contracts/plugin-interface.md for requirements
3. Review data-model.md for entity definitions
4. Fix issues and re-run from failing step

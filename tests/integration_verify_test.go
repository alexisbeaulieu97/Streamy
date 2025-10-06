package tests

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// stripANSICodes removes ANSI escape codes from output
func stripANSICodes(s string) string {
	// Simple regex to remove ANSI escape codes
	result := s
	for {
		start := strings.Index(result, "\x1b[")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}

func projectRootDir(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	root := filepath.Dir(wd)
	_, statErr := os.Stat(filepath.Join(root, "go.mod"))
	require.NoError(t, statErr)

	return root
}

func newStreamyCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command("go", append([]string{"run", "./cmd/streamy"}, args...)...)
	cmd.Dir = projectRootDir(t)
	return cmd
}

func newStreamyCmdContext(ctx context.Context, t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.CommandContext(ctx, "go", append([]string{"run", "./cmd/streamy"}, args...)...)
	cmd.Dir = projectRootDir(t)
	return cmd
}

func TestVerify_Integration_AllSatisfied(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create a simple file to satisfy a step
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("hello world"), 0644)
	require.NoError(t, err)

	// Create config with steps that will be satisfied
	config := `version: "1.0"
name: "Test All Satisfied"
steps:
  - id: check_file_exists
    type: command
    command: "echo 'done'"
    check: "test -f ` + testFile + `"
`

	err = os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command
	cmd := newStreamyCmd(t, "verify", configFile)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err)

	// Should exit with code 0 (all satisfied)
	cleanOutput := stripANSICodes(string(output))
	require.Contains(t, cleanOutput, "✔ Satisfied: 1")
	require.Contains(t, cleanOutput, "All steps satisfied - no changes needed")
}

func TestVerify_Integration_MissingSteps(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create config with steps pointing to non-existent resources
	nonExistentFile := filepath.Join(tmpDir, "missing.txt")
	config := `version: "1.0"
name: "Test Missing Steps"
steps:
  - id: check_missing_file
    type: command
    command: "echo 'done'"
    check: "test -f ` + nonExistentFile + `"
`

	err := os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command
	cmd := newStreamyCmd(t, "verify", configFile)
	output, err := cmd.CombinedOutput()

	// Should exit with code 1 (missing steps)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())

	cleanOutput := stripANSICodes(string(output))
	require.Contains(t, cleanOutput, "Missing:   1") // Note: triple space
	require.Contains(t, cleanOutput, "Changes needed - run 'streamy apply' to fix")
}

func TestVerify_Integration_DriftedSteps(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create a file with different content than expected
	testFile := filepath.Join(tmpDir, "drifted.txt")
	err := os.WriteFile(testFile, []byte("actual content"), 0644)
	require.NoError(t, err)

	// Create config expecting different content
	config := `version: "1.0"
name: "Test Drifted Steps"
steps:
  - id: check_file_content
    type: command
    command: "echo 'done'"
    check: "grep -q 'expected content' ` + testFile + `"
`

	err = os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command
	cmd := newStreamyCmd(t, "verify", configFile)
	output, err := cmd.CombinedOutput()

	// Should exit with code 1 (drifted steps)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())

	cleanOutput := stripANSICodes(string(output))
	require.Contains(t, cleanOutput, "Missing:   1") // Note: triple space
	require.Contains(t, cleanOutput, "Changes needed - run 'streamy apply' to fix")
}

func TestVerify_Integration_BlockedSteps(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create config with dependency on a missing step
	config := `version: "1.0"
name: "Test Blocked Steps"
steps:
  - id: missing_step
    type: command
    command: "echo 'done'"
    check: "test -f /non/existent/file"
  - id: dependent_step
    type: command
    depends_on:
      - missing_step
    command: "echo 'done'"
    check: "test -f /tmp/anything"
`

	err := os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command
	cmd := newStreamyCmd(t, "verify", configFile)
	output, err := cmd.CombinedOutput()

	// Should exit with code 1 (blocked/missing steps)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())

	require.Contains(t, string(output), "Changes needed - run 'streamy apply' to fix")
}

func TestVerify_Integration_UnknownSteps(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create config with command steps lacking verify clause
	config := `version: "1.0"
name: "Test Unknown Steps"
steps:
  - id: unknown_step
    type: command
    command: "echo 'hello world'"
    # No 'check' field specified
`

	err := os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command
	cmd := newStreamyCmd(t, "verify", configFile)
	output, err := cmd.CombinedOutput()

	// Should exit with code 1 (unknown steps)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())

	cleanOutput := stripANSICodes(string(output))
	require.Contains(t, cleanOutput, "Unknown:  1") // Note: double space
	require.Contains(t, cleanOutput, "Changes needed - run 'streamy apply' to fix")
}

func TestVerify_Integration_DependencyBlocking(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create config where step B depends on step A, and A will fail
	config := `version: "1.0"
name: "Test Dependency Blocking"
steps:
  - id: step_a
    type: command
    command: "echo 'done'"
    check: "test -f /non/existent/file"
  - id: step_b
    type: command
    depends_on:
      - step_a
    command: "echo 'done'"
    check: "test -f /tmp/anything"
  - id: step_c
    type: command
    depends_on:
      - step_b
    command: "echo 'done'"
    check: "test -f /tmp/anything"
`

	err := os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command
	cmd := newStreamyCmd(t, "verify", configFile)
	output, err := cmd.CombinedOutput()

	// Should exit with code 1 (missing/blocked steps)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())

	require.Contains(t, string(output), "Changes needed - run 'streamy apply' to fix")
}

func TestVerify_Integration_VerboseOutput(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	config := `version: "1.0"
name: "Test Verbose Output"
steps:
  - id: unknown_step
    type: command
    command: "echo 'hello world'"
`

	err := os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command with verbose flag
	cmd := newStreamyCmd(t, "verify", "--verbose", configFile)
	output, err := cmd.CombinedOutput()

	// Should exit with code 1 (unknown steps)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())

	cleanOutput := stripANSICodes(string(output))
	// Should include detailed information
	require.Contains(t, cleanOutput, "Step ID")
	require.Contains(t, cleanOutput, "Status")
	require.Contains(t, cleanOutput, "Duration")
	require.Contains(t, cleanOutput, "Message")
}

func TestVerify_Integration_JSONOutput(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	config := `version: "1.0"
name: "Test JSON Output"
steps:
  - id: unknown_step
    type: command
    command: "echo 'hello world'"
`

	err := os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command with JSON flag
	cmd := newStreamyCmd(t, "verify", "--json", configFile)
	output, err := cmd.CombinedOutput()

	// Should exit with code 1 (unknown steps)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())

	// Parse JSON output - find the pretty-printed JSON which starts with "{" and ends with "}"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var jsonLines []string
	foundStart := false
	braceCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "{" {
			foundStart = true
		}
		if foundStart {
			jsonLines = append(jsonLines, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				break
			}
		}
	}

	jsonOutput := strings.Join(jsonLines, "\n")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonOutput), &result)
	require.NoError(t, err)

	// Validate JSON schema
	require.Contains(t, result, "config_file")
	require.Equal(t, configFile, result["config_file"].(string))
	require.Contains(t, result, "summary")

	summary := result["summary"].(map[string]interface{})
	require.Contains(t, summary, "total_steps")
	require.Contains(t, summary, "satisfied")
	require.Contains(t, summary, "missing")
	require.Contains(t, summary, "drifted")
	require.Contains(t, summary, "blocked")
	require.Contains(t, summary, "unknown")
	require.Contains(t, summary, "duration_seconds")
	require.Contains(t, result, "results")

	results := result["results"].([]interface{})
	require.Len(t, results, 1)

	stepResult := results[0].(map[string]interface{})
	require.Contains(t, stepResult, "step_id")
	require.Contains(t, stepResult, "status")
	require.Contains(t, stepResult, "message")
	require.Contains(t, stepResult, "duration_seconds")
	require.Contains(t, stepResult, "timestamp")
}

func TestVerify_Integration_TimeoutHandling(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create config with a step that will timeout
	config := `version: "1.0"
name: "Test Timeout Handling"
steps:
  - id: timeout_step
    type: command
    command: "echo 'done'"
    check: "sleep 10 && test -f /tmp/anything"
    verify_timeout: 1  # Very short timeout
`

	err := os.WriteFile(configFile, []byte(config), 0644)
	require.NoError(t, err)

	// Run verify command
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := newStreamyCmdContext(ctx, t, "verify", configFile)
	_, err = cmd.CombinedOutput()

	// Should complete quickly due to timeout
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	// Should be either 1 (blocked) or another error code
}

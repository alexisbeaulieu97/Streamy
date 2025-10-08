package internalexec

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStreaming_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test a simple command that succeeds
	cmd := exec.Command("echo", "hello world")

	result, err := RunStreaming(cmd)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result.Stdout)
	assert.Equal(t, "", result.Stderr)
}

func TestRunStreaming_WithError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test a command that fails with stderr output
	cmd := exec.Command("sh", "-c", "echo 'error message' >&2; exit 1")

	result, err := RunStreaming(cmd)
	require.Error(t, err)
	assert.Equal(t, "", result.Stdout)
	assert.Equal(t, "error message", result.Stderr)
}

func TestRunStreaming_WithStdoutPipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test with custom stdout pipe
	var stdoutBuf bytes.Buffer
	cmd := exec.Command("echo", "piped output")
	cmd.Stdout = &stdoutBuf

	result, err := RunStreaming(cmd)
	require.NoError(t, err)
	assert.Equal(t, "piped output", result.Stdout)
	assert.Equal(t, "piped output\n", stdoutBuf.String())
	assert.Equal(t, "", result.Stderr)
}

func TestRunStreaming_WithStderrPipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test with custom stderr pipe
	var stderrBuf bytes.Buffer
	cmd := exec.Command("sh", "-c", "echo 'error message' >&2; exit 1")
	cmd.Stderr = &stderrBuf

	result, err := RunStreaming(cmd)
	require.Error(t, err)
	assert.Equal(t, "", result.Stdout)
	assert.Equal(t, "error message", result.Stderr)
	assert.Equal(t, "error message\n", stderrBuf.String())
}

func TestRunStreaming_WithBothPipes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test with both stdout and stderr pipes
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command("sh", "-c", "echo 'normal output'; echo 'error message' >&2; exit 1")
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	result, err := RunStreaming(cmd)
	require.Error(t, err)
	assert.Equal(t, "normal output", result.Stdout)
	assert.Equal(t, "error message", result.Stderr)
	assert.Equal(t, "normal output\n", stdoutBuf.String())
	assert.Equal(t, "error message\n", stderrBuf.String())
}

func TestRunStreaming_WithContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test with context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This command should take longer than the timeout
	cmd := exec.CommandContext(ctx, "sleep", "1")

	result, err := RunStreaming(cmd)
	require.Error(t, err)
	// Different OSes report context cancellation differently
	if runtime.GOOS == "linux" {
		assert.Contains(t, err.Error(), "signal: killed")
	} else {
		assert.Contains(t, err.Error(), "context")
	}
	assert.Empty(t, result.Stdout) // No output due to cancellation
}

func TestRunStreaming_OutputTrimming(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test that output is properly trimmed
	cmd := exec.Command("printf", "hello\nworld\n\t")

	result, err := RunStreaming(cmd)
	require.NoError(t, err)
	assert.Equal(t, "hello\nworld", result.Stdout) // Should be trimmed but preserve internal newlines
}

func TestPrimaryOutput(t *testing.T) {
	t.Run("returns stderr when present", func(t *testing.T) {
		result := Result{
			Stdout: "normal output",
			Stderr: "error message",
		}

		assert.Equal(t, "error message", PrimaryOutput(result))
	})

	t.Run("returns stdout when no stderr", func(t *testing.T) {
		result := Result{
			Stdout: "normal output",
			Stderr: "",
		}

		assert.Equal(t, "normal output", PrimaryOutput(result))
	})

	t.Run("returns empty string when both are empty", func(t *testing.T) {
		result := Result{
			Stdout: "",
			Stderr: "",
		}

		assert.Equal(t, "", PrimaryOutput(result))
	})

	t.Run("handles whitespace", func(t *testing.T) {
		result := Result{
			Stdout: "   ",
			Stderr: "",
		}

		assert.Equal(t, "   ", PrimaryOutput(result))
	})
}

func TestRunStreaming_RealWorldUsage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test a more realistic scenario similar to how plugins might use it
	cmd := exec.Command("sh", "-c", `
		echo "Starting process..."
		echo "Processing data" >&2
		echo "Result: success"
	`)

	// Capture timing to ensure it doesn't block indefinitely
	start := time.Now()
	result, err := RunStreaming(cmd)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 5*time.Second, "Command should complete quickly")
	assert.Equal(t, "Starting process...\nResult: success", result.Stdout)
	assert.Equal(t, "Processing data", result.Stderr)
}

func TestRunStreaming_CommandNotFound(t *testing.T) {
	// Test handling of non-existent command
	cmd := exec.Command("this-command-does-not-exist")

	result, err := RunStreaming(cmd)
	require.Error(t, err)
	assert.Empty(t, result.Stdout)
	// stderr might be empty for command not found errors on some systems
	// The error itself is sufficient to verify the failure
}

func TestRunStreaming_NoOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	// Test command that produces no output
	cmd := exec.Command("true")

	result, err := RunStreaming(cmd)
	require.NoError(t, err)
	assert.Equal(t, "", result.Stdout)
	assert.Equal(t, "", result.Stderr)
}

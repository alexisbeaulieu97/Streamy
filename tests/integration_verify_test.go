package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

func TestVerifyAllSatisfied(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	dir := t.TempDir()
	targetFile := filepath.Join(dir, "exists.txt")
	require.NoError(t, os.WriteFile(targetFile, []byte("ok"), 0o644))

	configPath := writeVerifyConfig(t, fmt.Sprintf(`
version: "1.0"
name: "verify"
steps:
  - id: check_file_exists
    type: command
    command: "echo checking"
    check: "test -f %s"
`, targetFile))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline, results, err := h.VerifyUseCase.Verify(ctx, configPath)
	require.NoError(t, err)
	require.NotNil(t, pipeline)
	require.Len(t, results, 1)
	require.Equal(t, domainpipeline.VerificationSatisfied, results[0].Status)
}

func TestVerifyMissingResource(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	missingFile := filepath.Join(t.TempDir(), "missing.txt")

	configPath := writeVerifyConfig(t, fmt.Sprintf(`
version: "1.0"
name: "verify-missing"
steps:
  - id: ensure_file_missing
    type: command
    command: "echo verify"
    check: "test -f %s"
`, missingFile))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline, results, err := h.VerifyUseCase.Verify(ctx, configPath)
	require.NoError(t, err)
	require.NotNil(t, pipeline)
	require.Len(t, results, 1)
	require.Equal(t, domainpipeline.VerificationFailed, results[0].Status)
}

func TestVerifyParseError(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	badPath := writeVerifyConfig(t, "version: 1.0\nsteps:\n  - id: invalid") // missing type

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := h.VerifyUseCase.Verify(ctx, badPath)
	require.Error(t, err)
}

func writeVerifyConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "verify.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}

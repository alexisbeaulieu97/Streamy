package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
)

func TestRunValidations_Success(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "exists.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("export PATH"), 0o644))

	validations := []config.Validation{
		{
			Type: "command_exists",
			CommandExists: &config.CommandExistsValidation{
				Command: "echo",
			},
		},
		{
			Type: "file_exists",
			FileExists: &config.FileExistsValidation{
				Path: filePath,
			},
		},
		{
			Type: "path_contains",
			PathContains: &config.PathContainsValidation{
				File: filePath,
				Text: "PATH",
			},
		},
	}

	results, err := RunValidations(context.Background(), validations)
	require.NoError(t, err)
	require.Len(t, results, len(validations))

	for i, result := range results {
		require.Equal(t, validations[i].Type, result.Validation.Type)
		require.True(t, result.Passed)
	}
}

func TestRunValidations_FailureAggregatesResults(t *testing.T) {
	t.Parallel()

	validations := []config.Validation{
		{
			Type: "command_exists",
			CommandExists: &config.CommandExistsValidation{
				Command: "definitely_missing_command",
			},
		},
		{
			Type: "file_exists",
			FileExists: &config.FileExistsValidation{
				Path: "./missing-file",
			},
		},
	}

	results, err := RunValidations(context.Background(), validations)
	require.Error(t, err)
	require.Len(t, results, len(validations))

	var failedCount int
	for _, r := range results {
		if !r.Passed {
			failedCount++
			require.NotEmpty(t, r.Message)
		}
	}
	require.Equal(t, 2, failedCount)
}

func TestRunValidations_EmptyList(t *testing.T) {
	t.Parallel()

	results, err := RunValidations(context.Background(), []config.Validation{})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestRunValidations_MixedResults(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	existingFile := filepath.Join(tmp, "exists.txt")
	require.NoError(t, os.WriteFile(existingFile, []byte("content"), 0o644))

	validations := []config.Validation{
		{
			Type: "command_exists",
			CommandExists: &config.CommandExistsValidation{
				Command: "echo", // This should pass
			},
		},
		{
			Type: "file_exists",
			FileExists: &config.FileExistsValidation{
				Path: filepath.Join(tmp, "nonexistent.txt"), // This should fail
			},
		},
		{
			Type: "file_exists",
			FileExists: &config.FileExistsValidation{
				Path: existingFile, // This should pass
			},
		},
	}

	results, err := RunValidations(context.Background(), validations)
	require.Error(t, err, "should return error when any validation fails")
	require.Len(t, results, 3)

	require.True(t, results[0].Passed)
	require.False(t, results[1].Passed)
	require.True(t, results[2].Passed)
}

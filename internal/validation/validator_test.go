package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	require "github.com/stretchr/testify/require"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

func TestRunValidations_Success(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "exists.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("export PATH"), 0o644))

	validations := []domain.Validation{
		{
			Type: domain.ValidationCommandExists,
			Config: map[string]any{
				"command": "echo",
			},
		},
		{
			Type: domain.ValidationFileExists,
			Config: map[string]any{
				"path": filePath,
			},
		},
		{
			Type: domain.ValidationPathContains,
			Config: map[string]any{
				"file": filePath,
				"text": "PATH",
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

	validations := []domain.Validation{
		{
			Type: domain.ValidationCommandExists,
			Config: map[string]any{
				"command": "definitely_missing_command",
			},
		},
		{
			Type: domain.ValidationFileExists,
			Config: map[string]any{
				"path": "./missing-file",
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

	results, err := RunValidations(context.Background(), []domain.Validation{})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestRunValidationsWithCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	validations := []domain.Validation{
		{
			Type: domain.ValidationCommandExists,
			Config: map[string]any{
				"command": "echo",
			},
		},
	}

	_, err := RunValidations(ctx, validations)
	require.Error(t, err)
}

func TestRunValidationsWithInvalidConfig(t *testing.T) {
	t.Parallel()

	validations := []domain.Validation{
		{
			Type:   domain.ValidationCommandExists,
			Config: map[string]any{"command": 123},
		},
		{
			Type:   domain.ValidationFileExists,
			Config: map[string]any{"path": 123},
		},
		{
			Type:   domain.ValidationPathContains,
			Config: map[string]any{"file": 123, "text": "abc"},
		},
	}

	_, err := RunValidations(context.Background(), validations)
	require.Error(t, err)
}

func TestStringConfigValue(t *testing.T) {
	t.Parallel()

	_, err := stringConfigValue(nil, "key")
	require.Error(t, err)

	_, err = stringConfigValue(map[string]any{}, "key")
	require.Error(t, err)

	_, err = stringConfigValue(map[string]any{"key": 123}, "key")
	require.Error(t, err)

	_, err = stringConfigValue(map[string]any{"key": " "}, "key")
	require.Error(t, err)
}

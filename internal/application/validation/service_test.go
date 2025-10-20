package validation

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	require "github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/application/pipeline/testutil"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

func TestService_RunValidations_AllPass(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Use the Go toolchain binary for command validation (always available in CI).
	cmdName := "go"

	existingPath := filepath.Join(tmpDir, "exists.txt")
	require.NoError(t, os.WriteFile(existingPath, []byte("content"), 0o644))

	containsPath := filepath.Join(tmpDir, "contains.txt")
	require.NoError(t, os.WriteFile(containsPath, []byte("hello streamy"), 0o644))

	svc := NewService(testutil.NewMockLogger())
	validations := []domainpipeline.Validation{
		{Type: domainpipeline.ValidationCommandExists, Config: map[string]interface{}{"command": cmdName}},
		{Type: domainpipeline.ValidationFileExists, Config: map[string]interface{}{"path": existingPath}},
		{Type: domainpipeline.ValidationPathContains, Config: map[string]interface{}{"file": containsPath, "text": "streamy"}},
	}

	summary, err := svc.RunValidations(context.Background(), validations)
	require.NoError(t, err)
	require.Equal(t, 3, summary.TotalChecks)
	require.Equal(t, 3, summary.PassedChecks)
	require.Zero(t, summary.FailedChecks)
}

func TestService_RunValidations_Failures(t *testing.T) {
	t.Parallel()

	logger := testutil.NewMockLogger()
	svc := NewService(logger)

	validations := []domainpipeline.Validation{
		{Type: domainpipeline.ValidationCommandExists, Config: map[string]interface{}{"command": "nonexistent-command"}},
		{Type: domainpipeline.ValidationFileExists, Config: nil},
	}

	summary, err := svc.RunValidations(context.Background(), validations)
	require.Error(t, err)
	require.Equal(t, 2, summary.TotalChecks)
	require.Equal(t, 2, summary.FailedChecks)
	require.Zero(t, summary.PassedChecks)
	require.Len(t, summary.FailureDetails, 2)

	var derr *domainpipeline.DomainError
	require.ErrorAs(t, err, &derr)
	require.Equal(t, domainpipeline.ErrCodeValidation, derr.Code)
	require.Contains(t, derr.Context, "failed_checks")
	require.Equal(t, 2, derr.Context["failed_checks"])
	require.Contains(t, derr.Context, "failures")
	failures, ok := derr.Context["failures"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, failures, 2)

	found := false

	for _, detail := range summary.FailureDetails {
		if detail["validation_type"] == domainpipeline.ValidationFileExists {
			found = true
			break
		}
	}

	require.True(t, found)

	entries := logger.Entries()
	require.NotEmpty(t, entries)
}

func TestService_RunValidations_UnsupportedType(t *testing.T) {
	t.Parallel()

	svc := NewService(testutil.NewMockLogger())
	validations := []domainpipeline.Validation{
		{Type: domainpipeline.ValidationType("custom"), Config: map[string]interface{}{"foo": "bar"}},
	}

	summary, err := svc.RunValidations(context.Background(), validations)
	require.Error(t, err)
	require.Equal(t, 1, summary.TotalChecks)
	require.Equal(t, 1, summary.FailedChecks)

	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeValidation, domainErr.Code)
	require.NotEmpty(t, domainErr.Context)
}

func TestStringConfig(t *testing.T) {
	t.Parallel()

	t.Run("missing config map", func(t *testing.T) {
		_, err := stringConfig(nil, "path")
		require.Error(t, err)

		var domainErr *domainpipeline.DomainError
		require.ErrorAs(t, err, &domainErr)
		require.Equal(t, domainpipeline.ErrCodeValidation, domainErr.Code)
		require.Equal(t, "path", domainErr.Context["required_key"])
	})

	t.Run("missing key", func(t *testing.T) {
		_, err := stringConfig(map[string]interface{}{}, "path")
		require.Error(t, err)

		var domainErr *domainpipeline.DomainError
		require.ErrorAs(t, err, &domainErr)
		require.Equal(t, domainpipeline.ErrCodeMissing, domainErr.Code)
		require.Equal(t, "path", domainErr.Context["missing_key"])
	})

	t.Run("invalid value", func(t *testing.T) {
		cases := []interface{}{123, ""}
		for _, raw := range cases {
			_, err := stringConfig(map[string]interface{}{"path": raw}, "path")
			require.Error(t, err)

			var domainErr *domainpipeline.DomainError
			require.ErrorAs(t, err, &domainErr)
			require.Equal(t, domainpipeline.ErrCodeValidation, domainErr.Code)
			require.Equal(t, "path", domainErr.Context["invalid_key"])
		}
	})

	t.Run("success", func(t *testing.T) {
		value, err := stringConfig(map[string]interface{}{"path": "ok"}, "path")
		require.NoError(t, err)
		require.Equal(t, "ok", value)
	})
}

func TestValidationWorkerLimit(t *testing.T) {
	original := runtime.GOMAXPROCS(0)
	runtime.GOMAXPROCS(4)
	t.Cleanup(func() {
		runtime.GOMAXPROCS(original)
	})

	require.Equal(t, 0, validationWorkerLimit(0))
	require.Equal(t, 2, validationWorkerLimit(2))
	require.Equal(t, 4, validationWorkerLimit(10))
	require.Equal(t, 4, validationWorkerLimit(100))
}

func TestService_RunValidations_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := NewService(testutil.NewMockLogger())
	summary, err := svc.RunValidations(ctx, []domainpipeline.Validation{{Type: domainpipeline.ValidationCommandExists}})
	require.Error(t, err)
	require.Zero(t, summary.TotalChecks)
	require.Zero(t, summary.PassedChecks)

	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeCancelled, domainErr.Code)
}

func TestSupportedValidationTypesForContext(t *testing.T) {
	t.Parallel()

	expected := []domainpipeline.ValidationType{
		domainpipeline.ValidationCommandExists,
		domainpipeline.ValidationFileExists,
		domainpipeline.ValidationPathContains,
	}

	require.ElementsMatch(t, expected, supportedValidationTypesForContext())
}

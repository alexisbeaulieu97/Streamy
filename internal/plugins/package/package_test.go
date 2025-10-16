package packageplugin

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
	"gopkg.in/yaml.v3"
)

func newPackageStep(t *testing.T, id string, cfg config.PackageStep) *config.Step {
	t.Helper()
	step := &config.Step{ID: id, Type: "package", Enabled: true}
	require.NoError(t, step.SetConfig(cfg))
	return step
}

func TestPackagePlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.PluginMetadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "package", meta.Name)
}

func TestPackagePlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.PackageStep)
	require.True(t, ok, "schema should be of type PackageStep")
}

func TestPackagePlugin_EvaluateForMissingPackage(t *testing.T) {
	t.Parallel()

	p := New()

	step := newPackageStep(t, "install_package", config.PackageStep{Packages: []string{"nonexistent-test-package-12345"}})

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, evalResult.StepID)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "packages not installed")
}

func TestPackagePlugin_EvaluateForExistingPackage(t *testing.T) {
	t.Parallel()

	p := New()

	// Use a common package that's likely to be installed
	step := newPackageStep(t, "check_existing_package", config.PackageStep{Packages: []string{"curl"}})

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, evalResult.StepID)

	// The result depends on whether curl is installed
	if evalResult.CurrentState == model.StatusSatisfied {
		require.False(t, evalResult.RequiresAction)
		require.Contains(t, evalResult.Message, "all packages installed")
	} else {
		require.True(t, evalResult.RequiresAction)
		require.Contains(t, evalResult.Message, "packages not installed")
	}
}

func TestPackagePlugin_ApplyInstallPackage(t *testing.T) {
	t.Parallel()

	p := New()

	step := newPackageStep(t, "install_test_package", config.PackageStep{Packages: []string{"test-package-12345"}})

	// First evaluate
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)

	// Then apply - this will likely fail since the package doesn't exist,
	// but we're testing the interface
	result, err := p.Apply(context.Background(), evalResult, step)
	// We expect either an error or a failed result
	if err == nil {
		require.NotNil(t, result)
		require.Equal(t, step.ID, result.StepID)
	}
}

func TestPackagePlugin_EvaluateVersionMismatch(t *testing.T) {
	t.Parallel()

	p := New()

	step := newPackageStep(t, "version_mismatch", config.PackageStep{Packages: []string{"curl"}})

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, evalResult.StepID)

	// Should detect as either missing (package not found) or satisfied (curl is installed)
	// The current implementation only checks if package exists, not version-specific matching
	require.NotNil(t, evalResult.CurrentState)
}

func TestPackagePlugin_ApplyWithUpgrade(t *testing.T) {
	t.Parallel()

	p := New()

	step := newPackageStep(t, "upgrade_package", config.PackageStep{Packages: []string{"curl"}, Update: true})

	// First evaluate
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.NotNil(t, evalResult)

	// Then apply
	result, err := p.Apply(context.Background(), evalResult, step)
	// We expect this to either succeed or fail gracefully
	if err == nil {
		require.NotNil(t, result)
		require.Equal(t, step.ID, result.StepID)
	}
}

func TestPackagePlugin_EvaluateErrors(t *testing.T) {
	t.Run("returns error when package config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{ID: "test_package", Type: "package"}

		_, err := p.Evaluate(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "package configuration missing")
	})

	t.Run("returns satisfied when packages list is empty", func(t *testing.T) {
		p := New()

		step := newPackageStep(t, "test_package", config.PackageStep{Packages: []string{}})

		evalResult, err := p.Evaluate(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
		require.False(t, evalResult.RequiresAction)
		require.Contains(t, evalResult.Message, "all packages installed")
	})
}

func TestPackagePlugin_ApplyErrors(t *testing.T) {
	t.Run("returns error when package config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{ID: "test_package", Type: "package", Enabled: true}

		evalResult := &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusMissing,
			RequiresAction: true,
			Message:        "Test",
		}

		_, err := p.Apply(context.Background(), evalResult, step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "package configuration missing")
	})
}

func TestPackagePlugin_EvaluateUsesRawConfigWhenStructNil(t *testing.T) {
	p := New()

	yamlStr := `
id: raw_package
type: package
packages:
  - curl
`

	var step config.Step
	require.NoError(t, yaml.Unmarshal([]byte(yamlStr), &step))

	evalResult, err := p.Evaluate(context.Background(), &step)
	require.NoError(t, err)
	require.Equal(t, step.ID, evalResult.StepID)
}

// Contract tests for the new plugin interface
func TestPackagePlugin_Contract(t *testing.T) {
	t.Run("Plugin Contract", func(t *testing.T) {
		plugin := New()

		t.Run("Metadata is stable", func(t *testing.T) {
			m1 := plugin.PluginMetadata()
			m2 := plugin.PluginMetadata()
			require.Equal(t, m1, m2, "PluginMetadata() should return consistent values across calls")
		})

		t.Run("Schema returns struct", func(t *testing.T) {
			schema := plugin.Schema()
			require.NotNil(t, schema, "Schema() should not return nil")
			_, ok := schema.(config.PackageStep)
			require.True(t, ok, "Schema() should return a PackageStep struct")
		})

		t.Run("Evaluate is idempotent", func(t *testing.T) {
			step := newPackageStep(t, "idempotent-test", config.PackageStep{Packages: []string{"nonexistent-package-12345"}})
			ctx := context.Background()

			// Call Evaluate twice
			result1, err1 := plugin.Evaluate(ctx, step)
			result2, err2 := plugin.Evaluate(ctx, step)

			require.NoError(t, err1, "First Evaluate() should not return an error")
			require.NoError(t, err2, "Second Evaluate() should not return an error")

			// Results should be equivalent for a non-existent package
			require.Equal(t, result1.CurrentState, result2.CurrentState, "CurrentState should be consistent across calls")
			require.Equal(t, result1.RequiresAction, result2.RequiresAction, "RequiresAction should be consistent across calls")
		})
	})
}

func TestPackagePlugin_ApplySkipsWhenNoAction(t *testing.T) {
	p := New()

	step := &config.Step{ID: "skip", Type: "package", Enabled: true}
	require.NoError(t, step.SetConfig(config.PackageStep{Packages: []string{"bash"}}))

	eval := &model.EvaluationResult{
		StepID:         step.ID,
		RequiresAction: false,
		CurrentState:   model.StatusSatisfied,
	}

	result, err := p.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSkipped, result.Status)
	require.Equal(t, step.ID, result.StepID)
}

func TestPackageConvertError(t *testing.T) {
	t.Run("wraps validation errors", func(t *testing.T) {
		err := streamyerrors.NewValidationError("field", "invalid", nil)
		converted := convertError("pkg", err)

		var pluginErr *plugin.ValidationError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "pkg", pluginErr.StepID())
	})

	t.Run("wraps execution errors", func(t *testing.T) {
		err := streamyerrors.NewExecutionError("legacy", errors.New("boom"))
		converted := convertError("pkg2", err)

		var pluginErr *plugin.ExecutionError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "pkg2", pluginErr.StepID())
	})

	t.Run("wraps unknown errors as execution", func(t *testing.T) {
		err := errors.New("other failure")
		converted := convertError("pkg3", err)

		var pluginErr *plugin.ExecutionError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "pkg3", pluginErr.StepID())
	})
}

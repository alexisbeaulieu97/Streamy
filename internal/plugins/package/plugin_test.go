package packageplugin

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/internalexec"
)

func TestMetadata(t *testing.T) {
	meta := New().Metadata()
	require.Equal(t, "package", meta.Name)
	require.Equal(t, domainplugin.TypePackage, meta.Type)
}

func TestEvaluateAllPackagesInstalled(t *testing.T) {
	original := runCommand
	defer func() { runCommand = original }()

	runCommand = func(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
		require.Equal(t, "dpkg-query", name)
		return internalexec.Result{}, nil
	}

	step := domainpipeline.Step{
		ID:   "pkg_all_installed",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]interface{}{
			"packages": []interface{}{"curl", "git"},
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationSatisfied), result.CurrentState)

	data, ok := result.InternalData.(*evaluationData)
	require.Truef(t, ok, "internal data type %T value %#v", result.InternalData, result.InternalData)
	require.ElementsMatch(t, []string{"curl", "git"}, data.InstalledPackages)
	require.Empty(t, data.MissingPackages)
}

func TestEvaluateDetectsMissingPackages(t *testing.T) {
	original := runCommand
	defer func() { runCommand = original }()

	runCommand = func(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
		if name != "dpkg-query" {
			return internalexec.Result{}, nil
		}
		if len(args) > 0 && args[len(args)-1] == "git" {
			return internalexec.Result{}, errPackageMissing
		}
		return internalexec.Result{}, nil
	}

	step := domainpipeline.Step{
		ID:   "pkg_missing",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]interface{}{
			"packages": []interface{}{"curl", "git"},
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)

	data, ok := result.InternalData.(*evaluationData)
	require.Truef(t, ok, "internal data type %T value %#v", result.InternalData, result.InternalData)
	require.ElementsMatch(t, []string{"curl"}, data.InstalledPackages)
	require.ElementsMatch(t, []string{"git"}, data.MissingPackages)
	require.Contains(t, result.Diff, "git")
}

func TestApplyInstallsMissingPackages(t *testing.T) {
	var dpkgCalls int
	var installCalls int

	original := runCommand
	defer func() { runCommand = original }()

	runCommand = func(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
		switch name {
		case "dpkg-query":
			dpkgCalls++
			if len(args) > 0 && args[len(args)-1] == "git" {
				return internalexec.Result{}, errPackageMissing
			}
			return internalexec.Result{}, nil
		case "apt-get":
			require.GreaterOrEqual(t, len(args), 1)
			if args[0] == "install" {
				installCalls++
			}
			return internalexec.Result{}, nil
		default:
			return internalexec.Result{}, nil
		}
	}

	step := domainpipeline.Step{
		ID:   "pkg_install",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]interface{}{
			"packages": []interface{}{"curl", "git"},
		},
	}

	plugin := New()

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)

	result, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)
	require.True(t, result.Changed)
	require.Contains(t, result.Message, "git")
	require.Equal(t, 2, dpkgCalls)
	require.Equal(t, 1, installCalls)
}

func TestApplyHandlesExecutionError(t *testing.T) {
	original := runCommand
	defer func() { runCommand = original }()

	runCommand = func(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
		switch name {
		case "dpkg-query":
			return internalexec.Result{}, errPackageMissing
		case "apt-get":
			return internalexec.Result{}, errors.New("apt explode")
		default:
			return internalexec.Result{}, nil
		}
	}

	step := domainpipeline.Step{
		ID:   "pkg_error",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]interface{}{
			"packages": []interface{}{"curl"},
		},
	}

	eval, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	result, applyErr := New().Apply(context.Background(), eval, step)
	require.Nil(t, result)
	require.Error(t, applyErr)
	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, applyErr, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeExecution, domainErr.Code)
}

func TestApplyRespectsAlreadySatisfiedEvaluation(t *testing.T) {
	original := runCommand
	defer func() { runCommand = original }()

	step := domainpipeline.Step{
		ID:   "pkg_satisfied",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]interface{}{
			"packages": []interface{}{"curl"},
		},
	}

	result, err := New().Apply(context.Background(), &domainpipeline.EvaluationResult{RequiresAction: false}, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusAlreadySatisfied, result.Status)
	require.Contains(t, result.Message, "no changes")
}

func TestEnsureEvaluationDataFallback(t *testing.T) {
	original := runCommand
	defer func() { runCommand = original }()

	var dpkgQueries int

	runCommand = func(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
		if name == "dpkg-query" {
			dpkgQueries++
			return internalexec.Result{}, errPackageMissing
		}
		return internalexec.Result{}, nil
	}

	step := domainpipeline.Step{
		ID:   "pkg_missing",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]interface{}{
			"packages": []interface{}{"curl"},
		},
	}

	plugin := New()

	result, err := plugin.Apply(context.Background(), nil, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)
	require.GreaterOrEqual(t, dpkgQueries, 1)
}

func TestApplyRunsUpdateWhenRequested(t *testing.T) {
	original := runCommand
	defer func() { runCommand = original }()

	var updateCalls int
	var installCalls int

	runCommand = func(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
		switch name {
		case "dpkg-query":
			return internalexec.Result{}, errPackageMissing
		case "apt-get":
			require.NotEmpty(t, args)
			if args[0] == "update" {
				updateCalls++
				return internalexec.Result{}, nil
			}
			if args[0] == "install" {
				installCalls++
				return internalexec.Result{}, nil
			}
		}
		return internalexec.Result{}, nil
	}

	step := domainpipeline.Step{
		ID:   "pkg_update",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]interface{}{
			"packages": []interface{}{"curl"},
			"update":   true,
		},
	}

	plugin := New()

	result, err := plugin.Apply(context.Background(), nil, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)
	require.Equal(t, 1, updateCalls)
	require.Equal(t, 1, installCalls)
}

func TestApplyHandlesUpdateTimeout(t *testing.T) {
	original := runCommand
	defer func() { runCommand = original }()

	runCommand = func(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
		if name == "dpkg-query" {
			return internalexec.Result{}, errPackageMissing
		}
		if name == "apt-get" && len(args) > 0 && args[0] == "update" {
			return internalexec.Result{}, context.DeadlineExceeded
		}
		return internalexec.Result{}, nil
	}

	step := domainpipeline.Step{
		ID:   "pkg_update_timeout",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]interface{}{
			"packages": []interface{}{"curl"},
			"update":   true,
		},
	}

	result, err := New().Apply(context.Background(), nil, step)
	require.Nil(t, result)
	require.Error(t, err)
	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeTimeout, domainErr.Code)
}

func TestDecodeConfigValidation(t *testing.T) {
	t.Parallel()

	_, err := decodeConfig(domainpipeline.Step{ID: "invalid"})
	require.Error(t, err)

	_, err = decodeConfig(domainpipeline.Step{ID: "pkg", Config: map[string]interface{}{}})
	require.Error(t, err)

	cfg, err := decodeConfig(domainpipeline.Step{
		ID: "pkg",
		Config: map[string]interface{}{
			"packages": []interface{}{"curl"},
			"update":   "true",
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"curl"}, cfg.Packages)
	require.True(t, cfg.Update)
}

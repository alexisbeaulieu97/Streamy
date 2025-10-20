package tests

import (
	"context"
	"testing"

	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	plugininfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

func TestRegistryResolvesDeclaredDependencies(t *testing.T) {
	reg := plugininfra.NewRegistry()

	line := newStubPlugin(domainplugin.Metadata{
		ID:           "line_in_file",
		Name:         "line_in_file",
		Type:         domainplugin.TypeLineInFile,
		Version:      "1.0.0",
		Dependencies: nil,
	})
	shell := newStubPlugin(domainplugin.Metadata{
		ID:           "template",
		Name:         "template",
		Type:         domainplugin.TypeTemplate,
		Version:      "1.0.0",
		Dependencies: []string{string(domainplugin.TypeLineInFile)},
	})

	require.NoError(t, reg.Register(line))
	require.NoError(t, reg.Register(shell))
	require.NoError(t, reg.ValidateDependencies())

	dep, err := reg.GetForDependent(string(domainplugin.TypeTemplate), domainplugin.TypeLineInFile)
	require.NoError(t, err)
	require.Equal(t, line, dep)

	ctx := context.Background()
	step := domainpipeline.Step{ID: "profile", Type: domainpipeline.StepType(domainplugin.TypeTemplate)}
	eval, err := shell.Evaluate(ctx, step)
	require.NoError(t, err)
	result, err := shell.Apply(ctx, eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)
}

func TestRegistryMissingDependency(t *testing.T) {
	reg := plugininfra.NewRegistry()

	needsLine := newStubPlugin(domainplugin.Metadata{
		ID:           "template",
		Name:         "template",
		Type:         domainplugin.TypeTemplate,
		Version:      "1.0.0",
		Dependencies: []string{string(domainplugin.TypeLineInFile)},
	})

	require.NoError(t, reg.Register(needsLine))
	err := reg.ValidateDependencies()
	require.Error(t, err)

	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeDependency, domainErr.Code)
}

func TestRegistryDetectsCycles(t *testing.T) {
	reg := plugininfra.NewRegistry()

	a := newStubPlugin(domainplugin.Metadata{
		ID:           "template",
		Name:         "template",
		Type:         domainplugin.TypeTemplate,
		Version:      "1.0.0",
		Dependencies: []string{string(domainplugin.TypeLineInFile)},
	})
	b := newStubPlugin(domainplugin.Metadata{
		ID:           "line_in_file",
		Name:         "line_in_file",
		Type:         domainplugin.TypeLineInFile,
		Version:      "1.0.0",
		Dependencies: []string{string(domainplugin.TypeTemplate)},
	})

	require.NoError(t, reg.Register(a))
	require.NoError(t, reg.Register(b))

	err := reg.ValidateDependencies()
	require.Error(t, err)

	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeCycle, domainErr.Code)
}

func TestRegistryUndeclaredAccess(t *testing.T) {
	reg := plugininfra.NewRegistry()

	provider := newStubPlugin(domainplugin.Metadata{
		ID:      "line_in_file",
		Name:    "line_in_file",
		Type:    domainplugin.TypeLineInFile,
		Version: "1.0.0",
	})
	consumer := newStubPlugin(domainplugin.Metadata{
		ID:      "template",
		Name:    "template",
		Type:    domainplugin.TypeTemplate,
		Version: "1.0.0",
	})

	require.NoError(t, reg.Register(provider))
	require.NoError(t, reg.Register(consumer))
	require.NoError(t, reg.ValidateDependencies())

	_, err := reg.GetForDependent(string(domainplugin.TypeTemplate), domainplugin.TypeLineInFile)
	require.Error(t, err)

	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeDependency, domainErr.Code)
}

type stubPlugin struct {
	meta     domainplugin.Metadata
	evaluate func(context.Context, domainpipeline.Step) (*domainpipeline.EvaluationResult, error)
	apply    func(context.Context, *domainpipeline.EvaluationResult, domainpipeline.Step) (*domainpipeline.StepResult, error)
}

func newStubPlugin(meta domainplugin.Metadata) ports.Plugin {
	return &stubPlugin{meta: meta}
}

func (p *stubPlugin) Metadata() domainplugin.Metadata {
	return p.meta
}

func (p *stubPlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if p.evaluate != nil {
		return p.evaluate(ctx, step)
	}

	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   "pending",
		InternalData:   step.Config,
	}, nil
}

func (p *stubPlugin) Apply(ctx context.Context, eval *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if p.apply != nil {
		return p.apply(ctx, eval, step)
	}

	return &domainpipeline.StepResult{
		StepID: step.ID,
		Status: domainpipeline.StatusSuccess,
	}, nil
}

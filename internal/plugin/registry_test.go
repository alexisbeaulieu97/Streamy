package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestRegistry_RegisterAndRetrieve(t *testing.T) {
	ResetRegistry()
	p := &testRegistryPlugin{}

	require.NoError(t, RegisterPlugin("command", p))

	fetched, err := GetPlugin("command")
	require.NoError(t, err)
	require.Equal(t, p, fetched)
}

func TestRegistry_PreventsDuplicateRegistration(t *testing.T) {
	ResetRegistry()
	p := &testRegistryPlugin{}

	require.NoError(t, RegisterPlugin("command", p))
	err := RegisterPlugin("command", &testRegistryPlugin{})
	require.Error(t, err)
	var pluginErr *streamyerrors.PluginError
	require.ErrorAs(t, err, &pluginErr)
}

func TestRegistry_ReturnsErrorForUnknownPlugin(t *testing.T) {
	ResetRegistry()

	_, err := GetPlugin("unknown")
	require.Error(t, err)
	var pluginErr *streamyerrors.PluginError
	require.ErrorAs(t, err, &pluginErr)
}

type testRegistryPlugin struct{}

func (p *testRegistryPlugin) Metadata() Metadata {
	return Metadata{Name: "test", Version: "1.0.0", Type: "command"}
}

func (p *testRegistryPlugin) Schema() interface{} { return nil }

func (p *testRegistryPlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	return false, nil
}

func (p *testRegistryPlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{StepID: step.ID, Status: "success"}, nil
}

func (p *testRegistryPlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{StepID: step.ID, Status: "skipped"}, nil
}

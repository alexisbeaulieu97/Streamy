package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

var _ Plugin = (*testPlugin)(nil)

type testPlugin struct{}

func (p *testPlugin) Metadata() Metadata {
	return Metadata{
		Name:    "test",
		Version: "1.0.0",
		Type:    "command",
	}
}

func (p *testPlugin) Schema() interface{} {
	return struct {
		Command string `yaml:"command"`
	}{}
}

func (p *testPlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	return false, nil
}

func (p *testPlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
	return &model.VerificationResult{
		StepID:  step.ID,
		Status:  model.StatusSatisfied,
		Message: "test verification satisfied",
	}, nil
}

func (p *testPlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{StepID: step.ID, Status: "success", Message: "applied"}, nil
}

func (p *testPlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{StepID: step.ID, Status: "skipped", Message: "dry-run"}, nil
}

func TestPluginMetadata(t *testing.T) {
	p := &testPlugin{}
	meta := p.Metadata()

	require.Equal(t, "test", meta.Name)
	require.Equal(t, "1.0.0", meta.Version)
	require.Equal(t, "command", meta.Type)
}

func TestPluginSchemaProvidesTypeInformation(t *testing.T) {
	p := &testPlugin{}
	schema := p.Schema()
	require.NotNil(t, schema)
}

func TestPluginDryRunAndApply(t *testing.T) {
	p := &testPlugin{}
	step := &config.Step{
		ID:   "run_command",
		Type: "command",
		Command: &config.CommandStep{
			Command: "echo hello",
		},
	}

	dryRun, err := p.DryRun(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "skipped", dryRun.Status)

	applied, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", applied.Status)
	require.Equal(t, step.ID, applied.StepID)
}

func TestPluginCheckSupportsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	p := &testPlugin{}
	_, err := p.Check(ctx, &config.Step{ID: "noop"})
	require.NoError(t, err)
}

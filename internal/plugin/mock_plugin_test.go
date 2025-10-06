package plugin

import (
	"context"
	"sync"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

type MockPluginOption func(*MockPlugin)

type MockPlugin struct {
	mu         sync.Mutex
	metadata   PluginMetadata
	pluginType string
	calls      []string
	schema     interface{}
	checkFn    func(context.Context, *config.Step) (bool, error)
	applyFn    func(context.Context, *config.Step) (*model.StepResult, error)
	dryRunFn   func(context.Context, *config.Step) (*model.StepResult, error)
	verifyFn   func(context.Context, *config.Step) (*model.VerificationResult, error)
}

func NewMockPlugin(name string, opts ...MockPluginOption) *MockPlugin {
	mp := &MockPlugin{
		metadata: PluginMetadata{
			Name:       name,
			Version:    "1.0.0",
			APIVersion: "1.x",
		},
		pluginType: name,
	}

	for _, opt := range opts {
		opt(mp)
	}

	if mp.metadata.Dependencies == nil {
		mp.metadata.Dependencies = []Dependency{}
	}
	return mp
}

func WithDependencies(deps ...Dependency) MockPluginOption {
	copied := make([]Dependency, len(deps))
	copy(copied, deps)
	return func(mp *MockPlugin) {
		mp.metadata.Dependencies = copied
	}
}

func WithStateful(stateful bool) MockPluginOption {
	return func(mp *MockPlugin) {
		mp.metadata.Stateful = stateful
	}
}

func WithDescription(desc string) MockPluginOption {
	return func(mp *MockPlugin) {
		mp.metadata.Description = desc
	}
}

func WithPluginType(t string) MockPluginOption {
	return func(mp *MockPlugin) {
		mp.pluginType = t
	}
}

func WithSchema(schema interface{}) MockPluginOption {
	return func(mp *MockPlugin) {
		mp.schema = schema
	}
}

func WithCheckFunc(fn func(context.Context, *config.Step) (bool, error)) MockPluginOption {
	return func(mp *MockPlugin) {
		mp.checkFn = fn
	}
}

func WithApplyFunc(fn func(context.Context, *config.Step) (*model.StepResult, error)) MockPluginOption {
	return func(mp *MockPlugin) {
		mp.applyFn = fn
	}
}

func WithDryRunFunc(fn func(context.Context, *config.Step) (*model.StepResult, error)) MockPluginOption {
	return func(mp *MockPlugin) {
		mp.dryRunFn = fn
	}
}

func WithVerifyFunc(fn func(context.Context, *config.Step) (*model.VerificationResult, error)) MockPluginOption {
	return func(mp *MockPlugin) {
		mp.verifyFn = fn
	}
}

func (m *MockPlugin) Metadata() Metadata {
	return Metadata{Name: m.metadata.Name, Version: m.metadata.Version, Type: m.pluginType}
}

func (m *MockPlugin) Schema() interface{} {
	m.recordCall("Schema")
	return m.schema
}

func (m *MockPlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	m.recordCall("Check")
	if m.checkFn != nil {
		return m.checkFn(ctx, step)
	}
	return false, nil
}

func (m *MockPlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	m.recordCall("Apply")
	if m.applyFn != nil {
		return m.applyFn(ctx, step)
	}
	return &model.StepResult{StepID: step.ID, Status: "success"}, nil
}

func (m *MockPlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	m.recordCall("DryRun")
	if m.dryRunFn != nil {
		return m.dryRunFn(ctx, step)
	}
	return &model.StepResult{StepID: step.ID, Status: "skipped"}, nil
}

func (m *MockPlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
	m.recordCall("Verify")
	if m.verifyFn != nil {
		return m.verifyFn(ctx, step)
	}
	return &model.VerificationResult{StepID: step.ID, Status: model.StatusSatisfied}, nil
}

func (m *MockPlugin) PluginMetadata() PluginMetadata {
	return m.metadata
}

func (m *MockPlugin) Calls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]string, len(m.calls))
	copy(copied, m.calls)
	return copied
}

func (m *MockPlugin) recordCall(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, name)
}

type InitializingMockPlugin struct {
	*MockPlugin
	initFn func(*PluginRegistry) error
}

func NewInitializingMockPlugin(name string, initFn func(*PluginRegistry) error, opts ...MockPluginOption) *InitializingMockPlugin {
	base := NewMockPlugin(name, opts...)
	return &InitializingMockPlugin{MockPlugin: base, initFn: initFn}
}

func (m *InitializingMockPlugin) Init(registry *PluginRegistry) error {
	m.recordCall("Init")
	if m.initFn != nil {
		return m.initFn(registry)
	}
	return nil
}

package plugin

import (
	"context"
	"errors"
	"sort"
	"testing"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type stubPlugin struct {
	meta domainplugin.Metadata
}

func (s *stubPlugin) Metadata() domainplugin.Metadata { return s.meta }
func (s *stubPlugin) Evaluate(context.Context, domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	return &domainpipeline.EvaluationResult{RequiresAction: false}, nil
}
func (s *stubPlugin) Apply(context.Context, *domainpipeline.EvaluationResult, domainpipeline.Step) (*domainpipeline.StepResult, error) {
	return &domainpipeline.StepResult{StepID: "stub", Status: domainpipeline.StatusAlreadySatisfied}, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	pluginType := domainplugin.Type("command")
	stub := &stubPlugin{meta: domainplugin.Metadata{ID: "stub", Name: "stub", Type: pluginType, Version: "1.0.0"}}

	if err := reg.Register(stub); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	got, err := reg.Get(pluginType)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if got.Metadata().Type != pluginType {
		t.Fatalf("expected plugin type %s, got %s", pluginType, got.Metadata().Type)
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	types := []domainplugin.Type{"command", "copy", "package"}
	expected := append([]domainplugin.Type(nil), types...)
	sort.Slice(expected, func(i, j int) bool { return expected[i] < expected[j] })

	for _, typ := range types {
		stub := &stubPlugin{meta: domainplugin.Metadata{ID: string(typ), Name: string(typ), Type: typ, Version: "1.0.0"}}
		if err := reg.Register(stub); err != nil {
			t.Fatalf("register %s: %v", typ, err)
		}
	}

	plugins := reg.List()
	if len(plugins) != len(types) {
		t.Fatalf("expected %d plugins, got %d", len(types), len(plugins))
	}
	for i, p := range plugins {
		if p.Metadata().Type != expected[i] {
			t.Fatalf("expected plugin order %v, got %v", expected, pluginTypes(plugins))
		}
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	reg := NewRegistry()
	stub := &stubPlugin{meta: domainplugin.Metadata{ID: "stub", Name: "stub", Type: domainplugin.Type("command"), Version: "1.0.0"}}

	if err := reg.Register(stub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := reg.Register(stub); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get(domainplugin.Type("missing"))
	if err == nil {
		t.Fatal("expected error for missing plugin")
	}
	if _, ok := err.(*domainpipeline.DomainError); !ok {
		t.Fatalf("expected domain error, got %T", err)
	}
}

func TestRegistryRegisterInvalidMetadata(t *testing.T) {
	reg := NewRegistry()
	stub := &stubPlugin{meta: domainplugin.Metadata{Name: "invalid", Type: "command", Version: "1.0.0"}}

	if err := reg.Register(stub); err == nil {
		t.Fatal("expected error for invalid metadata")
	}
}

func TestRegistryRegisterFactoryErrors(t *testing.T) {
	reg := NewRegistry()

	if err := reg.RegisterFactory("", func() (ports.Plugin, error) { return nil, nil }); err == nil {
		t.Fatal("expected error for missing type")
	}

	if err := reg.RegisterFactory(domainplugin.TypeCommand, nil); err == nil {
		t.Fatal("expected error for nil factory")
	}

	if err := reg.RegisterFactory(domainplugin.TypeCommand, func() (ports.Plugin, error) { return nil, nil }); err == nil {
		t.Fatal("expected error for nil plugin")
	}

	if err := reg.RegisterFactory(domainplugin.TypeCommand, func() (ports.Plugin, error) {
		return &stubPlugin{meta: domainplugin.Metadata{ID: "mismatch", Name: "mismatch", Type: domainplugin.TypeCopy, Version: "1.0.0"}}, nil
	}); err == nil {
		t.Fatal("expected error for type mismatch")
	}

	expected := errors.New("boom")
	if err := reg.RegisterFactory(domainplugin.TypeCommand, func() (ports.Plugin, error) { return nil, expected }); err == nil || !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
}

func TestRegistryValidateDependencies(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(newStubPlugin(domainplugin.TypeCommand)); err != nil {
		t.Fatalf("register command: %v", err)
	}
	if err := reg.Register(newStubPlugin(domainplugin.TypePackage, domainplugin.TypeCommand)); err != nil {
		t.Fatalf("register package: %v", err)
	}

	if err := reg.ValidateDependencies(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestRegistryValidateDependenciesMissing(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(newStubPlugin(domainplugin.TypePackage, domainplugin.TypeRepo)); err != nil {
		t.Fatalf("register package: %v", err)
	}
	err := reg.ValidateDependencies()
	if err == nil {
		t.Fatal("expected dependency validation error")
	}
	assertDomainErrCode(t, err, domainpipeline.ErrCodeDependency)
}

func TestRegistryValidateDependenciesCycle(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(newStubPlugin(domainplugin.TypePackage, domainplugin.TypeRepo)); err != nil {
		t.Fatalf("register package: %v", err)
	}
	if err := reg.Register(newStubPlugin(domainplugin.TypeRepo, domainplugin.TypePackage)); err != nil {
		t.Fatalf("register repo: %v", err)
	}

	err := reg.ValidateDependencies()
	if err == nil {
		t.Fatal("expected cycle validation error")
	}
	assertDomainErrCode(t, err, domainpipeline.ErrCodeCycle)
}

func TestRegistryInitializePlugins(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(newStubPlugin(domainplugin.TypeCommand)); err != nil {
		t.Fatalf("register command: %v", err)
	}
	if err := reg.Register(newStubPlugin(domainplugin.TypePackage, domainplugin.TypeCommand)); err != nil {
		t.Fatalf("register package: %v", err)
	}

	if err := reg.ValidateDependencies(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if err := reg.InitializePlugins(); err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}
}

func TestRegistryGetForDependent(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(newStubPlugin(domainplugin.TypeCommand)); err != nil {
		t.Fatalf("register command: %v", err)
	}
	if err := reg.Register(newStubPlugin(domainplugin.TypePackage, domainplugin.TypeCommand)); err != nil {
		t.Fatalf("register package: %v", err)
	}

	plugin, err := reg.GetForDependent(string(domainplugin.TypePackage), domainplugin.TypeCommand)
	if err != nil {
		t.Fatalf("unexpected dependency retrieval error: %v", err)
	}
	if plugin.Metadata().Type != domainplugin.TypeCommand {
		t.Fatalf("expected command plugin, got %s", plugin.Metadata().Type)
	}
}

func newStubPlugin(typ domainplugin.Type, deps ...domainplugin.Type) *stubPlugin {
	depNames := make([]string, 0, len(deps))
	for _, dep := range deps {
		depNames = append(depNames, string(dep))
	}
	return &stubPlugin{
		meta: domainplugin.Metadata{
			ID:           string(typ),
			Name:         string(typ),
			Type:         typ,
			Version:      "1.0.0",
			Dependencies: depNames,
		},
	}
}

func assertDomainErrCode(t *testing.T, err error, code domainpipeline.ErrorCode) {
	t.Helper()
	var derr *domainpipeline.DomainError
	if !errors.As(err, &derr) {
		t.Fatalf("expected domain error, got %T", err)
	}
	if derr.Code != code {
		t.Fatalf("expected error code %s, got %s", code, derr.Code)
	}
}

func pluginTypes(plugins []ports.Plugin) []domainplugin.Type {
	types := make([]domainplugin.Type, 0, len(plugins))
	for _, p := range plugins {
		types = append(types, p.Metadata().Type)
	}
	return types
}

package plugin

import (
	"fmt"
	"sort"
	"sync"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Registry implements ports.PluginRegistry with an in-memory map keyed by plugin type.
type Registry struct {
	mu       sync.RWMutex
	plugins  map[domainplugin.Type]ports.Plugin
	metadata map[domainplugin.Type]domainplugin.Metadata
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins:  make(map[domainplugin.Type]ports.Plugin),
		metadata: make(map[domainplugin.Type]domainplugin.Metadata),
	}
}

// Register stores a plugin implementation keyed by its metadata type.
func (r *Registry) Register(p ports.Plugin) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}
	meta := p.Metadata()
	if err := meta.Validate(); err != nil {
		return fmt.Errorf("plugin metadata invalid: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[meta.Type]; exists {
		return fmt.Errorf("plugin for type %q already registered", meta.Type)
	}
	r.plugins[meta.Type] = p
	r.metadata[meta.Type] = meta

	return nil
}

// RegisterFactory registers a plugin using a factory function that produces the implementation.
// The provided stepType must match the Metadata().Type returned by the constructed plugin.
func (r *Registry) RegisterFactory(stepType domainplugin.Type, factory func() (ports.Plugin, error)) error {
	if stepType == "" {
		return fmt.Errorf("plugin type is required")
	}
	if factory == nil {
		return fmt.Errorf("plugin factory is nil for type %q", stepType)
	}

	plugin, err := factory()
	if err != nil {
		return fmt.Errorf("construct plugin %q: %w", stepType, err)
	}
	if plugin == nil {
		return fmt.Errorf("plugin factory returned nil for type %q", stepType)
	}

	meta := plugin.Metadata()
	if meta.Type == "" {
		return fmt.Errorf("plugin metadata type is required for %q", stepType)
	}
	if meta.Type != stepType {
		return fmt.Errorf("plugin metadata type %q does not match registration type %q", meta.Type, stepType)
	}

	return r.Register(plugin)
}

// ValidateDependencies ensures all declared dependencies are present and acyclic.
func (r *Registry) ValidateDependencies() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	graph := make(map[domainplugin.Type][]domainplugin.Type, len(r.metadata))
	for typ, meta := range r.metadata {
		deps := make([]domainplugin.Type, 0, len(meta.Dependencies))
		for _, dep := range meta.Dependencies {
			depType := domainplugin.Type(dep)
			if _, ok := r.metadata[depType]; !ok {
				return &domainpipeline.DomainError{
					Code:    domainpipeline.ErrCodeDependency,
					Message: "plugin dependency not registered",
					Context: map[string]interface{}{
						"plugin_type":     string(typ),
						"dependency_type": string(depType),
					},
				}
			}
			deps = append(deps, depType)
		}
		graph[typ] = deps
	}

	if cycle := detectCycle(graph); len(cycle) > 0 {
		strCycle := make([]string, len(cycle))
		for i, node := range cycle {
			strCycle[i] = string(node)
		}
		return &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeCycle,
			Message: "circular plugin dependency detected",
			Context: map[string]interface{}{"cycle": strCycle},
		}
	}

	return nil
}

// InitializePlugins currently validates the dependency graph and acts as a placeholder for future hooks.
func (r *Registry) InitializePlugins() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	graph := make(map[domainplugin.Type][]domainplugin.Type, len(r.metadata))
	for typ, meta := range r.metadata {
		deps := make([]domainplugin.Type, 0, len(meta.Dependencies))
		for _, dep := range meta.Dependencies {
			deps = append(deps, domainplugin.Type(dep))
		}
		graph[typ] = deps
	}

	if _, err := topologicalOrder(graph); err != nil {
		return err
	}
	return nil
}

// GetForDependent retrieves a dependency for a specific plugin, ensuring the relationship was declared.
func (r *Registry) GetForDependent(dependent string, depType domainplugin.Type) (ports.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dependentType := domainplugin.Type(dependent)
	meta, ok := r.metadata[dependentType]
	if !ok {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeNotFound,
			Message: "dependent plugin not registered",
			Context: map[string]interface{}{"plugin_type": dependent},
		}
	}

	plugin, ok := r.plugins[depType]
	if !ok {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeNotFound,
			Message: "dependency plugin not registered",
			Context: map[string]interface{}{
				"plugin_type":     dependent,
				"dependency_type": string(depType),
			},
		}
	}

	declared := false
	for _, dep := range meta.Dependencies {
		if dep == string(depType) {
			declared = true
			break
		}
	}
	if !declared {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeDependency,
			Message: "undeclared plugin dependency",
			Context: map[string]interface{}{
				"plugin_type":     dependent,
				"dependency_type": string(depType),
			},
		}
	}

	return plugin, nil
}

// Get returns the plugin that handles the provided type.
func (r *Registry) Get(stepType domainplugin.Type) (ports.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[stepType]
	if !ok {
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeNotFound,
			Message: "plugin not registered",
			Context: map[string]interface{}{"plugin_type": stepType},
		}
	}
	return plugin, nil
}

// List returns all registered plugins.
func (r *Registry) List() []ports.Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.Plugin, 0, len(r.plugins))
	types := make([]domainplugin.Type, 0, len(r.plugins))
	for t := range r.plugins {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})

	for _, t := range types {
		result = append(result, r.plugins[t])
	}
	return result
}

var _ ports.PluginRegistry = (*Registry)(nil)

type visitState int

const (
	stateUnvisited visitState = iota
	stateVisiting
	stateVisited
)

func detectCycle(graph map[domainplugin.Type][]domainplugin.Type) []domainplugin.Type {
	state := make(map[domainplugin.Type]visitState, len(graph))
	stack := make([]domainplugin.Type, 0, len(graph))
	var cycle []domainplugin.Type

	var dfs func(domainplugin.Type) bool
	dfs = func(node domainplugin.Type) bool {
		state[node] = stateVisiting
		stack = append(stack, node)

		for _, dep := range graph[node] {
			switch state[dep] {
			case stateUnvisited:
				if dfs(dep) {
					return true
				}
			case stateVisiting:
				idx := indexOf(stack, dep)
				if idx >= 0 {
					cycle = append([]domainplugin.Type(nil), stack[idx:]...)
					cycle = append(cycle, dep)
				} else {
					cycle = []domainplugin.Type{dep}
				}
				return true
			}
		}

		stack = stack[:len(stack)-1]
		state[node] = stateVisited
		return false
	}

	for node := range graph {
		if state[node] == stateUnvisited {
			if dfs(node) {
				return cycle
			}
		}
	}
	return nil
}

func topologicalOrder(graph map[domainplugin.Type][]domainplugin.Type) ([]domainplugin.Type, error) {
	inDegree := make(map[domainplugin.Type]int, len(graph))
	for node := range graph {
		inDegree[node] = 0
	}
	for node, deps := range graph {
		inDegree[node] = len(deps)
		for _, dep := range deps {
			if _, ok := inDegree[dep]; !ok {
				// If dependency not present, graph is invalid; guard earlier ensures presence.
				inDegree[dep] = 0
			}
		}
	}

	queue := make([]domainplugin.Type, 0, len(inDegree))
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	order := make([]domainplugin.Type, 0, len(inDegree))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		for dependent, deps := range graph {
			for _, dep := range deps {
				if dep == node {
					inDegree[dependent]--
					if inDegree[dependent] == 0 {
						queue = append(queue, dependent)
					}
				}
			}
		}
	}

	if len(order) != len(inDegree) {
		cycle := detectCycle(graph)
		strCycle := make([]string, len(cycle))
		for i, node := range cycle {
			strCycle[i] = string(node)
		}
		return nil, &domainpipeline.DomainError{
			Code:    domainpipeline.ErrCodeCycle,
			Message: "circular plugin dependency detected",
			Context: map[string]interface{}{"cycle": strCycle},
		}
	}

	return order, nil
}

func indexOf(stack []domainplugin.Type, target domainplugin.Type) int {
	for i, v := range stack {
		if v == target {
			return i
		}
	}
	return -1
}

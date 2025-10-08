package plugin

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/alexisbeaulieu97/streamy/internal/logger"
)

// PluginRegistry manages plugin registration and dependency resolution.
type PluginRegistry struct {
	mu                sync.RWMutex
	plugins           map[string]Plugin
	metadata          map[string]PluginMetadata
	typeToName        map[string]string // Maps legacy Type to Name for backward compatibility
	dependencyGraph   *DependencyGraph
	statefulInstances map[string]map[string]Plugin
	disabled          map[string]bool
	logger            *logger.Logger
	config            *RegistryConfig
}

// NewPluginRegistry returns a new registry instance.
func NewPluginRegistry(config *RegistryConfig, log *logger.Logger) *PluginRegistry {
	if config == nil {
		config = DefaultConfig()
	}

	return &PluginRegistry{
		plugins:           make(map[string]Plugin),
		metadata:          make(map[string]PluginMetadata),
		typeToName:        make(map[string]string),
		dependencyGraph:   NewDependencyGraph(),
		statefulInstances: make(map[string]map[string]Plugin),
		disabled:          make(map[string]bool),
		logger:            log,
		config:            config,
	}
}

// Register adds a plugin to the registry.
func (r *PluginRegistry) Register(p Plugin) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	meta, err := r.extractMetadata(p)
	if err != nil {
		return err
	}
	if err := meta.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[meta.Name]; exists {
		return fmt.Errorf("plugin '%s' already registered", meta.Name)
	}

	r.plugins[meta.Name] = p
	r.metadata[meta.Name] = meta
	r.dependencyGraph.AddNode(meta.Name)

	// For legacy plugins, map Type -> Name for backward compatibility
	if meta.Description != "" {
		r.typeToName[meta.Description] = meta.Name
	}

	if meta.Stateful {
		r.statefulInstances[meta.Name] = make(map[string]Plugin)
	} else {
		delete(r.statefulInstances, meta.Name)
	}

	for _, dep := range meta.Dependencies {
		r.dependencyGraph.AddEdge(meta.Name, dep.Name)
	}

	delete(r.disabled, meta.Name)
	return nil
}

// ValidateDependencies verifies plugin dependency graphs.
func (r *PluginRegistry) ValidateDependencies() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.disabled = make(map[string]bool)

	var issues []error
	versionConflicts := make(map[string]*ErrVersionConflict)

	for name, meta := range r.metadata {
		for _, dep := range meta.Dependencies {
			target, exists := r.plugins[dep.Name]
			if !exists {
				err := &ErrMissingDependency{Plugin: name, Dependency: dep.Name}
				if r.config.DependencyPolicy == PolicyStrict {
					return err
				}
				r.disableAffectedPlugins(err)
				issues = append(issues, err)
				continue
			}

			depMeta, ok := r.metadata[dep.Name]
			if !ok {
				extracted, extractErr := r.extractMetadata(target)
				if extractErr != nil {
					err := fmt.Errorf("unable to extract metadata for plugin '%s': %w", dep.Name, extractErr)
					if r.config.DependencyPolicy == PolicyStrict {
						return err
					}
					issues = append(issues, err)
					continue
				}
				depMeta = extracted
				r.metadata[dep.Name] = depMeta
			}

			if dep.VersionConstraint != nil && !dep.VersionConstraint.Satisfies(depMeta.Version) {
				vc := versionConflicts[dep.Name]
				if vc == nil {
					vc = &ErrVersionConflict{
						Plugin:        dep.Name,
						ActualVersion: depMeta.Version,
						RequiredBy:    make(map[string]string),
					}
					versionConflicts[dep.Name] = vc
				}
				vc.RequiredBy[name] = dep.VersionConstraint.String()
				if r.config.DependencyPolicy == PolicyStrict {
					return vc
				}
			}
		}
	}

	for _, conflict := range versionConflicts {
		r.disableAffectedPlugins(conflict)
		issues = append(issues, conflict)
	}

	cycle, err := r.dependencyGraph.DetectCycles()
	if err != nil {
		return err
	}
	if len(cycle) > 0 {
		cycleErr := &ErrCircularDependency{Cycle: cycle}
		if r.config.DependencyPolicy == PolicyStrict {
			return cycleErr
		}
		r.disableAffectedPlugins(cycleErr)
		issues = append(issues, cycleErr)
	}

	if len(issues) > 0 {
		for _, issue := range issues {
			r.logWarn(issue.Error())
		}
	}

	return nil
}

// InitializePlugins initializes registered plugins respecting dependencies.
func (r *PluginRegistry) InitializePlugins() error {
	r.mu.RLock()
	order, err := r.dependencyGraph.TopologicalSort()
	if err != nil {
		r.mu.RUnlock()
		return err
	}

	type initTarget struct {
		name   string
		plugin Plugin
	}
	targets := make([]initTarget, 0, len(order))
	for _, name := range order {
		if r.disabled[name] {
			continue
		}
		plugin, exists := r.plugins[name]
		if !exists {
			continue
		}
		targets = append(targets, initTarget{name: name, plugin: plugin})
	}
	r.mu.RUnlock()

	for _, target := range targets {
		if initializer, ok := target.plugin.(PluginInitializer); ok {
			if err := initializer.Init(r); err != nil {
				return fmt.Errorf("init plugin '%s': %w", target.name, err)
			}
		}
	}

	return nil
}

// Get retrieves a plugin by name or type (for legacy plugins).
func (r *PluginRegistry) Get(nameOrType string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try direct name lookup
	plugin, exists := r.plugins[nameOrType]
	if exists && !r.disabled[nameOrType] {
		return plugin, nil
	}

	// For backward compatibility, try type->name mapping
	if name, ok := r.typeToName[nameOrType]; ok {
		plugin, exists = r.plugins[name]
		if exists && !r.disabled[name] {
			return plugin, nil
		}
	}

	return nil, ErrPluginNotFound{Name: nameOrType}
}

// GetForDependent retrieves a dependency for a specific plugin, enforcing policies.
func (r *PluginRegistry) GetForDependent(dependentName, pluginName string) (Plugin, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.plugins[pluginName]
	if !exists || r.disabled[pluginName] {
		return nil, ErrPluginNotFound{Name: pluginName}
	}

	dependent, ok := r.plugins[dependentName]
	if !ok {
		return nil, ErrPluginNotFound{Name: dependentName}
	}

	if !r.isDependencyDeclared(dependent, pluginName) {
		switch r.config.AccessPolicy {
		case AccessStrict:
			return nil, ErrUndeclaredDependency{Caller: dependentName, Dependency: pluginName}
		case AccessWarn:
			r.logWarn(fmt.Sprintf("plugin '%s' accessed undeclared dependency '%s'", dependentName, pluginName))
		}
	}

	meta := r.metadata[pluginName]
	if !meta.Stateful {
		return plugin, nil
	}

	instances := r.statefulInstances[pluginName]
	if instances == nil {
		instances = make(map[string]Plugin)
		r.statefulInstances[pluginName] = instances
	}
	if instance, ok := instances[dependentName]; ok {
		return instance, nil
	}

	instance := r.createPluginInstance(pluginName)
	if instance == nil {
		return nil, fmt.Errorf("stateful plugin '%s' cannot create new instance", pluginName)
	}
	instances[dependentName] = instance
	return instance, nil
}

// List returns the registered plugin names in sorted order.
func (r *PluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		if r.disabled[name] {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *PluginRegistry) isDependencyDeclared(caller Plugin, depName string) bool {
	if caller == nil {
		return false
	}

	// All plugins now implement PluginMetadata() directly
	meta := caller.PluginMetadata()
	for _, dep := range meta.Dependencies {
		if dep.Name == depName {
			return true
		}
	}

	return false
}

func (r *PluginRegistry) disableAffectedPlugins(err error) {
	switch e := err.(type) {
	case *ErrMissingDependency:
		r.disabled[e.Plugin] = true
	case ErrMissingDependency:
		r.disabled[e.Plugin] = true
	case *ErrCircularDependency:
		for _, name := range e.Cycle {
			r.disabled[name] = true
		}
	case ErrCircularDependency:
		for _, name := range e.Cycle {
			r.disabled[name] = true
		}
	case *ErrVersionConflict:
		for dependent := range e.RequiredBy {
			r.disabled[dependent] = true
		}
	case ErrVersionConflict:
		for dependent := range e.RequiredBy {
			r.disabled[dependent] = true
		}
	}
}

func (r *PluginRegistry) createPluginInstance(name string) Plugin {
	original, ok := r.plugins[name]
	if !ok {
		return nil
	}

	value := reflect.ValueOf(original)
	if !value.IsValid() {
		return nil
	}

	typ := value.Type()
	if typ.Kind() != reflect.Ptr {
		return nil
	}

	elem := typ.Elem()
	instance := reflect.New(elem).Interface()
	plugin, ok := instance.(Plugin)
	if !ok {
		return nil
	}
	return plugin
}

func (r *PluginRegistry) extractMetadata(p Plugin) (PluginMetadata, error) {
	// All plugins now implement PluginMetadata() directly
	meta := p.PluginMetadata()
	if meta.Name == "" {
		return PluginMetadata{}, fmt.Errorf("plugin metadata missing name")
	}

	// Set defaults
	if meta.Dependencies == nil {
		meta.Dependencies = []Dependency{}
	}
	if meta.APIVersion == "" {
		meta.APIVersion = "1.x"
	}

	return meta, nil
}

func (r *PluginRegistry) logWarn(msg string) {
	if r.logger == nil {
		return
	}
	r.logger.Warn(msg)
}

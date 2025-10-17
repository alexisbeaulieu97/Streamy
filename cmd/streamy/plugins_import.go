package main

import (
	"context"
	"fmt"
	"time"

	plugininfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	commandplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	copyplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	lineinfileplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"
	packageplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	repoplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	symlinkplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
	templateplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/template"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type pluginConstructor struct {
	name          string
	legacyFactory func() plugin.Plugin
	portsFactory  func() ports.Plugin
}

var builtinPluginConstructors = []pluginConstructor{
	{name: "command", legacyFactory: commandplugin.New, portsFactory: commandplugin.NewPort},
	{name: "copy", legacyFactory: copyplugin.New, portsFactory: copyplugin.NewPort},
	{name: "line_in_file", legacyFactory: lineinfileplugin.New, portsFactory: lineinfileplugin.NewPort},
	{name: "package", legacyFactory: packageplugin.New, portsFactory: packageplugin.NewPort},
	{name: "repo", legacyFactory: repoplugin.New, portsFactory: repoplugin.NewPort},
	{name: "symlink", legacyFactory: symlinkplugin.New, portsFactory: symlinkplugin.NewPort},
	{name: "template", legacyFactory: templateplugin.New, portsFactory: templateplugin.NewPort},
}

// RegisterPlugins wires built-in plugins into the legacy registry (until fully removed).
func RegisterPlugins(registry *plugin.PluginRegistry, log *logger.Logger) error {
	start := time.Now()

	for _, ctor := range builtinPluginConstructors {
		if ctor.legacyFactory == nil {
			return fmt.Errorf("legacy factory missing for plugin %q", ctor.name)
		}
		if err := registry.Register(ctor.legacyFactory()); err != nil {
			return fmt.Errorf("register plugin %q: %w", ctor.name, err)
		}
	}

	if err := registry.ValidateDependencies(); err != nil {
		return fmt.Errorf("validate plugin dependencies: %w", err)
	}

	if err := registry.InitializePlugins(); err != nil {
		return fmt.Errorf("initialize plugins: %w", err)
	}

	if log != nil {
		names := registry.List()
		fields := map[string]any{
			"plugins":      names,
			"plugin_count": len(names),
			"duration_ms":  time.Since(start).Milliseconds(),
		}
		log.WithFields(fields).Info("plugins initialized")
	}

	return nil
}

// RegisterPortsPlugins registers ports-native adapters into the new registry.
func RegisterPortsPlugins(ctx context.Context, reg *plugininfra.Registry, log ports.Logger) error {
	start := time.Now()

	registeredNames := make([]string, 0, len(builtinPluginConstructors))

	for _, ctor := range builtinPluginConstructors {
		if ctor.portsFactory == nil {
			return fmt.Errorf("ports factory missing for plugin %q", ctor.name)
		}
		plugin := ctor.portsFactory()
		if plugin == nil {
			return fmt.Errorf("ports factory for plugin %q returned nil", ctor.name)
		}
		if err := reg.Register(plugin); err != nil {
			return fmt.Errorf("register plugin %q: %w", ctor.name, err)
		}
		meta := plugin.Metadata()
		if meta.Type != "" {
			registeredNames = append(registeredNames, string(meta.Type))
		} else {
			registeredNames = append(registeredNames, ctor.name)
		}
	}

	if log != nil {
		log.Info(ctx, "ports plugins registered",
			"plugins", registeredNames,
			"plugin_count", len(registeredNames),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}

	return nil
}

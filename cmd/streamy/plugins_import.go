package main

import (
	"context"
	"fmt"
	"time"

	plugininfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
	commandplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	copyplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	lineinfileplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"
	packageplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	repoplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	symlinkplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
	templateplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/template"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type portPluginFactory struct {
	name    string
	factory func() ports.Plugin
}

var portPluginFactories = []portPluginFactory{
	{name: "command", factory: commandplugin.New},
	{name: "copy", factory: copyplugin.New},
	{name: "line_in_file", factory: lineinfileplugin.New},
	{name: "package", factory: packageplugin.New},
	{name: "repo", factory: repoplugin.New},
	{name: "symlink", factory: symlinkplugin.New},
	{name: "template", factory: templateplugin.New},
}

// RegisterPortsPlugins registers ports-native adapters into the new registry.
func RegisterPortsPlugins(ctx context.Context, reg *plugininfra.Registry, log ports.Logger) error {
	start := time.Now()

	registeredNames := make([]string, 0, len(portPluginFactories))

	for _, ctor := range portPluginFactories {
		if ctor.factory == nil {
			return fmt.Errorf("ports factory missing for plugin %q", ctor.name)
		}

		plugin := ctor.factory()
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

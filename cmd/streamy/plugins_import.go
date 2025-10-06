package main

import (
	"fmt"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	commandplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	copyplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	lineinfileplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"
	packageplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	repoplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	symlinkplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
	templateplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/template"
)

// RegisterPlugins wires built-in plugins into the provided registry and validates their dependencies.
func RegisterPlugins(registry *plugin.PluginRegistry, log *logger.Logger) error {
	start := time.Now()

	constructors := []struct {
		name    string
		factory func() plugin.Plugin
	}{
		{name: "command", factory: commandplugin.New},
		{name: "copy", factory: copyplugin.New},
		{name: "line_in_file", factory: lineinfileplugin.New},
		{name: "package", factory: packageplugin.New},
		{name: "repo", factory: repoplugin.New},
		{name: "symlink", factory: symlinkplugin.New},
		{name: "template", factory: templateplugin.New},
	}

	for _, ctor := range constructors {
		if err := registry.Register(ctor.factory()); err != nil {
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

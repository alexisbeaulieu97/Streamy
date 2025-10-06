package main

import "github.com/alexisbeaulieu97/streamy/internal/plugin"

var appRegistry *plugin.PluginRegistry

func setAppRegistry(reg *plugin.PluginRegistry) {
    appRegistry = reg
}

func getAppRegistry() *plugin.PluginRegistry {
    return appRegistry
}

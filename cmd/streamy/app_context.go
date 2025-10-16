package main

import (
	pipelineapp "github.com/alexisbeaulieu97/streamy/internal/app/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
)

// AppContext bundles long-lived services created at startup.
type AppContext struct {
	Registry *plugin.PluginRegistry
	Pipeline *pipelineapp.Service
}

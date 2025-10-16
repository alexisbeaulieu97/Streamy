package main

import (
	"fmt"
	"os"

	pipelineapp "github.com/alexisbeaulieu97/streamy/internal/app/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func main() {
	log, err := logger.New(logger.Options{Level: "info", HumanReadable: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}

	cfg := plugin.DefaultConfig()
	registry := plugin.NewPluginRegistry(cfg, log)

	if err := RegisterPlugins(registry, log); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare plugins: %v\n", err)
		os.Exit(1)
	}

	app := &AppContext{
		Registry: registry,
		Pipeline: pipelineapp.NewService(registry),
	}

	if err := newRootCmd(app).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

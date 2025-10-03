package engine

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

// ExecutionContext contains runtime state shared across executor workers.
type ExecutionContext struct {
	Config          *config.Config
	DryRun          bool
	Verbose         bool
	ContinueOnError bool
	WorkerPool      chan struct{}
	Results         map[string]*model.StepResult
	Logger          *logger.Logger
	Context         context.Context
}

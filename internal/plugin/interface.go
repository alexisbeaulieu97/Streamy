package plugin

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

// Metadata describes plugin capabilities.
type Metadata struct {
	Name    string
	Version string
	Type    string
}

// Plugin defines the contract all Streamy plugins must satisfy.
type Plugin interface {
	Metadata() Metadata
	Schema() interface{}
	Check(ctx context.Context, step *config.Step) (bool, error)
	Apply(ctx context.Context, step *config.Step) (*model.StepResult, error)
	DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error)
}

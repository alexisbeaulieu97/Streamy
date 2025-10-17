package dashboard

import (
	"context"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// PipelineService exposes the minimal operations the dashboard requires to
// verify and apply pipelines.
type PipelineService interface {
	Verify(ctx context.Context, opts VerifyOptions) (*registry.ExecutionResult, error)
	Apply(ctx context.Context, opts ApplyOptions) (*registry.ExecutionResult, error)
}

// VerifyOptions configures a verification request.
type VerifyOptions struct {
	ConfigPath string
	Timeout    time.Duration
}

// ApplyOptions configures an apply request.
type ApplyOptions struct {
	ConfigPath      string
	DryRun          bool
	Verbose         bool
	ContinueOnError bool
}

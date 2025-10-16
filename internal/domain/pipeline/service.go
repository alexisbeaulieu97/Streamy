package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/validation"
)

// executor defines the interface for verifying steps, allowing for mocking.
type executor interface {
	VerifySteps(ctx *engine.ExecutionContext, steps []config.Step, defaultTimeout time.Duration) (*model.VerificationSummary, error)
}

// Service exposes pure pipeline operations without binding to registry or CLI concerns.
type Service struct {
	registry    *plugin.PluginRegistry
	newExecutor func(log *logger.Logger) executor
	executePlan func(execCtx *engine.ExecutionContext, plan *engine.ExecutionPlan) ([]model.StepResult, error)
}

// NewService constructs a domain pipeline service.
func NewService(reg *plugin.PluginRegistry) *Service {
	return &Service{
		registry: reg,
		newExecutor: func(log *logger.Logger) executor {
			return engine.NewExecutor(log)
		},
		executePlan: engine.Execute,
	}
}

// PreparedPipeline captures configuration and planning artefacts reused across operations.
type PreparedPipeline struct {
	Path   string
	Config *config.Config
	Graph  *engine.Graph
	Plan   *engine.ExecutionPlan
}

// Prepare loads configuration, builds the DAG and execution plan.
func (s *Service) Prepare(configPath string) (*PreparedPipeline, error) {
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return nil, err
	}

	graph, err := engine.BuildDAG(cfg.Steps)
	if err != nil {
		return nil, err
	}

	plan, err := engine.GeneratePlan(graph)
	if err != nil {
		return nil, err
	}

	return &PreparedPipeline{
		Path:   configPath,
		Config: cfg,
		Graph:  graph,
		Plan:   plan,
	}, nil
}

// VerifyRequest configures a verification run.
type VerifyRequest struct {
	Prepared       *PreparedPipeline
	ConfigPath     string
	Logger         *logger.Logger
	Verbose        bool
	PerStepTimeout time.Duration
	DefaultTimeout time.Duration
}

// VerifyOutcome returns verification details.
type VerifyOutcome struct {
	Prepared *PreparedPipeline
	Summary  *model.VerificationSummary
}

// Verify executes verification for a pipeline, returning the summary alongside any execution error.
func (s *Service) Verify(ctx context.Context, req VerifyRequest) (*VerifyOutcome, error) {
	prepared, err := s.ensurePrepared(req.ConfigPath, req.Prepared)
	if err != nil {
		return nil, err
	}

	if req.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	perStepTimeout := req.PerStepTimeout
	if perStepTimeout <= 0 {
		if prepared.Config != nil && prepared.Config.Settings.Timeout > 0 {
			perStepTimeout = time.Duration(prepared.Config.Settings.Timeout) * time.Second
		} else if req.DefaultTimeout > 0 {
			perStepTimeout = req.DefaultTimeout
		} else {
			perStepTimeout = 30 * time.Second
		}
	}

	execCtx := &engine.ExecutionContext{
		Config:   prepared.Config,
		DryRun:   true,
		Verbose:  req.Verbose,
		Logger:   req.Logger,
		Context:  ctx,
		Registry: s.registry,
	}

	executor := s.newExecutor(req.Logger)
	summary, verifyErr := executor.VerifySteps(execCtx, prepared.Config.Steps, perStepTimeout)

	outcome := &VerifyOutcome{
		Prepared: prepared,
		Summary:  summary,
	}

	if verifyErr != nil {
		return outcome, verifyErr
	}

	return outcome, nil
}

// ApplyRequest configures an apply run.
type ApplyRequest struct {
	Prepared        *PreparedPipeline
	ConfigPath      string
	Logger          *logger.Logger
	DryRunOverride  bool
	VerboseOverride bool
	ContinueOnError bool
	OnStepResult    func(model.StepResult)
	OnValidation    func(validation.ValidationResult)
}

// ApplyOutcome captures apply execution details.
type ApplyOutcome struct {
	Prepared          *PreparedPipeline
	Results           []model.StepResult
	ValidationResults []validation.ValidationResult
	ExecutionErr      error
	ValidationErr     error
}

// Apply executes a pipeline apply operation, returning step and validation results alongside any error.
func (s *Service) Apply(ctx context.Context, req ApplyRequest) (*ApplyOutcome, error) {
	prepared, err := s.ensurePrepared(req.ConfigPath, req.Prepared)
	if err != nil {
		return nil, err
	}

	if req.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if prepared.Config == nil {
		return nil, fmt.Errorf("prepared config is required for apply")
	}

	effectiveDryRun := prepared.Config.Settings.DryRun || req.DryRunOverride
	effectiveVerbose := prepared.Config.Settings.Verbose || req.VerboseOverride
	continueOnError := prepared.Config.Settings.ContinueOnError || req.ContinueOnError

	parallel := prepared.Config.Settings.Parallel
	if parallel <= 0 {
		parallel = 4
	}

	execCtx := &engine.ExecutionContext{
		Config:          prepared.Config,
		DryRun:          effectiveDryRun,
		Verbose:         effectiveVerbose,
		ContinueOnError: continueOnError,
		WorkerPool:      make(chan struct{}, parallel),
		Results:         make(map[string]*model.StepResult),
		Logger:          req.Logger,
		Context:         ctx,
		Registry:        s.registry,
	}

	results, execErr := s.executePlan(execCtx, prepared.Plan)

	for _, res := range results {
		if req.OnStepResult != nil {
			req.OnStepResult(res)
		}
	}

	var validationResults []validation.ValidationResult
	var validationErr error
	if len(prepared.Config.Validations) > 0 {
		validationResults, validationErr = validation.RunValidations(ctx, prepared.Config.Validations)
		for _, vr := range validationResults {
			if req.OnValidation != nil {
				req.OnValidation(vr)
			}
		}
	}

	outcome := &ApplyOutcome{
		Prepared:          prepared,
		Results:           results,
		ValidationResults: validationResults,
		ExecutionErr:      execErr,
		ValidationErr:     validationErr,
	}

	if execErr != nil {
		return outcome, execErr
	}
	if validationErr != nil {
		return outcome, validationErr
	}

	return outcome, nil
}

func (s *Service) ensurePrepared(configPath string, prepared *PreparedPipeline) (*PreparedPipeline, error) {
	if prepared != nil {
		return prepared, nil
	}
	if configPath == "" {
		return nil, fmt.Errorf("config path required")
	}
	return s.Prepare(configPath)
}

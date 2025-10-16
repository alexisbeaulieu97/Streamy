package repoplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type repoPlugin struct{}

// New creates a new repository plugin.
func New() plugin.Plugin {
	return &repoPlugin{}
}

var _ plugin.Plugin = (*repoPlugin)(nil)

// PluginMetadata describes the plugin for the dependency registry.
//
// The empty Dependencies slice documents that repo does not require other plugins.
// APIVersion pins compatibility with other plugins using the registry-provided interface.
func (p *repoPlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:         "repo",
		Type:         "repo",
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: []plugin.Dependency{},
		Stateful:     false,
		Description:  "Manages git repositories with clone and update support.",
	}
}

func (p *repoPlugin) Schema() any {
	return config.RepoStep{}
}

// Evaluation data for repository operations
type repoEvaluationData struct {
	RepoExists   bool
	IsGitRepo    bool
	ActualURL    string
	ExpectedURL  string
	Destination  string
	Branch       string
	Depth        int
	CurrentHead  string
	DesiredHead  string
	CloneOptions *git.CloneOptions
}

func (p *repoPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	// Check context first (only if context is provided)
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	repoCfg, err := loadRepoConfig(step)
	if err != nil {
		return nil, plugin.NewValidationError(step.ID, err)
	}

	// Check destination directory (read-only operation)
	dirExists := true
	if _, err := os.Stat(repoCfg.Destination); err != nil {
		if os.IsNotExist(err) {
			dirExists = false
		} else {
			return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot access destination: %w", err))
		}
	}

	// Check if it's a git repository (read-only operation)
	gitDir := filepath.Join(repoCfg.Destination, ".git")
	isGitRepo := false
	var actualURL string
	var currentHead string

	if dirExists {
		if _, err := os.Stat(gitDir); err == nil {
			// Only treat as git repo when we can open it cleanly; otherwise flag drift.
			repo, err := git.PlainOpen(repoCfg.Destination)
			if err == nil {
				isGitRepo = true

				// Get current HEAD/branch
				head, err := repo.Head()
				if err == nil {
					currentHead = head.Name().Short()
				}

				// Get remote URL
				remote, err := repo.Remote("origin")
				if err == nil && len(remote.Config().URLs) > 0 {
					actualURL = remote.Config().URLs[0]
				}
			}
		}
	}

	// Store evaluation data to avoid recomputation
	cloneOpts := &git.CloneOptions{
		URL: repoCfg.URL,
	}
	if repoCfg.Depth > 0 {
		cloneOpts.Depth = repoCfg.Depth
	}
	if repoCfg.Branch != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(repoCfg.Branch)
		cloneOpts.SingleBranch = true
	}

	internalData := &repoEvaluationData{
		RepoExists:   dirExists,
		IsGitRepo:    isGitRepo,
		ActualURL:    actualURL,
		ExpectedURL:  repoCfg.URL,
		Destination:  repoCfg.Destination,
		Branch:       repoCfg.Branch,
		Depth:        repoCfg.Depth,
		CurrentHead:  currentHead,
		DesiredHead:  repoCfg.Branch,
		CloneOptions: cloneOpts,
	}

	// Determine current state
	if !dirExists {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusMissing,
			RequiresAction: true,
			Message:        fmt.Sprintf("repository directory %s does not exist", repoCfg.Destination),
			Diff:           fmt.Sprintf("Would clone: %s", repoCfg.URL),
			InternalData:   internalData,
		}, nil
	}

	if !isGitRepo {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusDrifted,
			RequiresAction: true,
			Message:        fmt.Sprintf("directory %s exists but is not a git repository", repoCfg.Destination),
			Diff:           fmt.Sprintf("Would remove directory and clone: %s", repoCfg.URL),
			InternalData:   internalData,
		}, nil
	}

	// Check if remote URL matches (only if we were able to determine the actual URL)
	if actualURL != "" && actualURL != repoCfg.URL {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusDrifted,
			RequiresAction: true,
			Message:        fmt.Sprintf("remote URL is %s (expected %s)", actualURL, repoCfg.URL),
			Diff:           fmt.Sprintf("Would reclone with URL: %s", repoCfg.URL),
			InternalData:   internalData,
		}, nil
	}

	// Check if branch matches (if specified)
	if repoCfg.Branch != "" && currentHead != repoCfg.Branch {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusDrifted,
			RequiresAction: true,
			Message:        fmt.Sprintf("current branch is %s (expected %s)", currentHead, repoCfg.Branch),
			Diff:           fmt.Sprintf("Would checkout branch: %s", repoCfg.Branch),
			InternalData:   internalData,
		}, nil
	}

	// Repository is in correct state
	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusSatisfied,
		RequiresAction: false,
		Message:        fmt.Sprintf("git repository exists at %s", repoCfg.Destination),
		InternalData:   internalData,
	}, nil
}

func (p *repoPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	repoCfg, err := loadRepoConfig(step)
	if err != nil {
		return nil, plugin.NewValidationError(step.ID, err)
	}

	// Use evaluation data to avoid recomputation
	var data *repoEvaluationData
	if evalResult != nil {
		if typed, ok := evalResult.InternalData.(*repoEvaluationData); ok {
			data = typed
		}
	}
	if data == nil {
		// Fallback to re-evaluating
		var err error
		evalResult, err = p.Evaluate(ctx, step)
		if err != nil {
			return nil, convertError(step.ID, err)
		}
		typed, ok := evalResult.InternalData.(*repoEvaluationData)
		if !ok || typed == nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: "evaluation failed during apply",
				Error:   fmt.Errorf("evaluation result missing repository evaluation data"),
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("evaluation failed during apply"))
		}
		data = typed
	}

	// Only apply if changes are needed
	if !evalResult.RequiresAction {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusSkipped,
			Message: "no changes needed",
		}, nil
	}

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(repoCfg.Destination), 0o755); err != nil {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: fmt.Sprintf("failed to create destination directory: %v", err),
			Error:   err,
		}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to create destination directory: %w", err))
	}

	// If directory exists but is not a git repo, remove it first
	if data.RepoExists && !data.IsGitRepo {
		if err := os.RemoveAll(repoCfg.Destination); err != nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: fmt.Sprintf("failed to remove existing directory: %v", err),
				Error:   err,
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to remove existing directory: %w", err))
		}
	}

	// Clone the repository
	if _, err := git.PlainCloneContext(ctx, repoCfg.Destination, false, data.CloneOptions); err != nil {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: fmt.Sprintf("failed to clone repository: %v", err),
			Error:   err,
		}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to clone repository: %w", err))
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("cloned %s", repoCfg.URL),
	}, nil
}

// Helper functions

func convertError(stepID string, err error) error {
	// Convert legacy errors to new plugin errors
	var valErr *streamyerrors.ValidationError
	if errors.As(err, &valErr) {
		return plugin.NewValidationError(stepID, valErr.Err)
	}

	var execErr *streamyerrors.ExecutionError
	if errors.As(err, &execErr) {
		return plugin.NewExecutionError(stepID, execErr.Err)
	}

	// Fallback to ExecutionError for unknown error types
	return plugin.NewExecutionError(stepID, err)
}

type repoConfig struct {
	*config.RepoStep
	RawConfig map[string]any
}

func loadRepoConfig(step *config.Step) (*repoConfig, error) {
	if step == nil {
		return nil, fmt.Errorf("step is nil")
	}

	repoPath := strings.TrimSpace(os.Getenv("STREAMY_REPO_PATH"))
	branch := strings.TrimSpace(os.Getenv("STREAMY_REPO_BRANCH"))
	depth := strings.TrimSpace(os.Getenv("STREAMY_REPO_DEPTH"))
	url := strings.TrimSpace(os.Getenv("STREAMY_REPO_URL"))

	raw := step.RawConfig()
	if len(raw) == 0 {
		return nil, fmt.Errorf("repo configuration missing")
	}

	cfg := &config.RepoStep{}
	if err := step.DecodeConfig(cfg); err != nil {
		return nil, fmt.Errorf("repo configuration decode failed: %w", err)
	}

	if repoPath != "" {
		cfg.Destination = repoPath
	}
	if branch != "" {
		cfg.Branch = branch
	}
	if depth != "" {
		parsedDepth, err := strconv.Atoi(depth)
		if err != nil {
			return nil, fmt.Errorf("invalid STREAMY_REPO_DEPTH %q: %w", depth, err)
		}
		if parsedDepth < 0 {
			return nil, fmt.Errorf("invalid STREAMY_REPO_DEPTH %q: must be >= 0", depth)
		}
		cfg.Depth = parsedDepth
	}
	if url != "" {
		cfg.URL = url
	}

	// Validate the configuration after environment overrides
	if err := config.GetValidator().Struct(cfg); err != nil {
		return nil, fmt.Errorf("repo configuration validation failed: %w", err)
	}

	return &repoConfig{
		RepoStep:  cfg,
		RawConfig: raw,
	}, nil
}

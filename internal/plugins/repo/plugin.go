// Package repoplugin reconciles git repositories for pipeline steps.
package repoplugin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Plugin implements cloning and validation of git repositories.
type Plugin struct{}

var _ ports.Plugin = (*Plugin)(nil)

// New constructs a ports-native repo plugin.
func New() ports.Plugin {
	return &Plugin{}
}

// Metadata describes the repo plugin for registry wiring.
func (Plugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "repo",
		Name:        "repo",
		Version:     "1.0.0",
		Type:        domainplugin.TypeRepo,
		Description: "Manages git repositories with clone and update support.",
	}
}

type evaluationData struct {
	RepoExists   bool
	IsGitRepo    bool
	ActualURL    string
	ExpectedURL  string
	Destination  string
	Branch       string
	Depth        int
	CurrentHead  string
	CloneOptions *git.CloneOptions
}

// Evaluate inspects repository state and reports whether a clone/update is required.
func (Plugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("repo evaluation cancelled", map[string]interface{}{"step_id": step.ID})
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data := newEvaluationData(cfg)

	repo, evalResult, inspectErr := inspectDestination(cfg, data, step.ID)
	if inspectErr != nil {
		return nil, inspectErr
	}

	if evalResult != nil {
		return evalResult, nil
	}

	collectRepoMetadata(repo, data)

	if drift := detectRepoDrift(cfg, data); drift != nil {
		return drift, nil
	}

	return repoSatisfied(cfg, data), nil
}

func newEvaluationData(cfg stepConfig) *evaluationData {
	return &evaluationData{
		ExpectedURL:  cfg.URL,
		Destination:  cfg.Destination,
		Branch:       cfg.Branch,
		Depth:        cfg.Depth,
		CloneOptions: buildCloneOptions(cfg),
	}
}

func inspectDestination(cfg stepConfig, data *evaluationData, stepID string) (*git.Repository, *domainpipeline.EvaluationResult, error) {
	info, statErr := os.Stat(cfg.Destination)
	switch {
	case errors.Is(statErr, os.ErrNotExist):
		data.RepoExists = false
		return nil, repoMissing(cfg.URL, data), nil
	case statErr != nil:
		return nil, nil, domainpipeline.NewExecutionError("inspect destination", statErr, map[string]interface{}{
			"step_id":     stepID,
			"destination": cfg.Destination,
		})
	default:
		if !info.IsDir() {
			data.RepoExists = true
			data.IsGitRepo = false

			return nil, repoDrifted("destination exists but is not a directory", data), nil
		}

		data.RepoExists = true
	}

	repo, openErr := git.PlainOpen(cfg.Destination)
	if openErr != nil {
		if errors.Is(openErr, git.ErrRepositoryNotExists) {
			data.IsGitRepo = false

			return nil, repoDrifted(fmt.Sprintf("directory %s exists but is not a git repository", cfg.Destination), data), nil
		}

		return nil, nil, domainpipeline.NewExecutionError("open git repository", openErr, map[string]interface{}{
			"step_id":     stepID,
			"destination": cfg.Destination,
		})
	}

	data.IsGitRepo = true

	return repo, nil, nil
}

func collectRepoMetadata(repo *git.Repository, data *evaluationData) {
	if repo == nil {
		return
	}

	if remote, err := repo.Remote("origin"); err == nil {
		if urls := remote.Config().URLs; len(urls) > 0 {
			data.ActualURL = urls[0]
		}
	}

	if head, err := repo.Head(); err == nil && head.Name().IsBranch() {
		data.CurrentHead = head.Name().Short()
	}
}

func detectRepoDrift(cfg stepConfig, data *evaluationData) *domainpipeline.EvaluationResult {
	if data.ActualURL != "" && !urlsEqual(data.ActualURL, cfg.URL) {
		return repoDrifted(fmt.Sprintf("remote URL is %s (expected %s)", data.ActualURL, cfg.URL), data)
	}

	if cfg.Branch != "" && data.CurrentHead != "" && data.CurrentHead != cfg.Branch {
		return repoDrifted(fmt.Sprintf("current branch is %s (expected %s)", data.CurrentHead, cfg.Branch), data)
	}

	return nil
}

func repoSatisfied(cfg stepConfig, data *evaluationData) *domainpipeline.EvaluationResult {
	return &domainpipeline.EvaluationResult{
		RequiresAction: false,
		CurrentState:   string(domainpipeline.VerificationSatisfied),
		DesiredState:   fmt.Sprintf("git repository exists at %s", cfg.Destination),
		InternalData:   data,
	}
}

// Apply reconciles the repository by cloning when evaluation reported drift/missing state.
func (Plugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("repo apply cancelled", map[string]interface{}{"step_id": step.ID})
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data, err := ensureEvaluationData(ctx, evaluation, step)
	if err != nil {
		return nil, err
	}

	resetNeeded := shouldResetRepository(cfg, data)
	if evaluation != nil && !evaluation.RequiresAction && !resetNeeded {
		return repoAlreadySatisfied(step.ID), nil
	}

	if resetNeeded {
		if resetErr := resetRepositoryDestination(cfg.Destination, step.ID); resetErr != nil {
			return nil, resetErr
		}
	}

	if cloneErr := cloneRepository(ctx, cfg, data.CloneOptions, step.ID); cloneErr != nil {
		return nil, cloneErr
	}

	return repoCloned(step.ID, cfg.URL), nil
}

func shouldResetRepository(cfg stepConfig, data *evaluationData) bool {
	if data == nil {
		return true
	}

	if !data.RepoExists || !data.IsGitRepo {
		return true
	}

	if data.ActualURL != "" && !urlsEqual(data.ActualURL, cfg.URL) {
		return true
	}

	if cfg.Branch != "" && data.CurrentHead != "" && data.CurrentHead != cfg.Branch {
		return true
	}

	return false
}

func resetRepositoryDestination(destination, stepID string) error {
	if err := os.RemoveAll(destination); err != nil {
		return domainpipeline.NewExecutionError("remove repository destination", err, map[string]interface{}{
			"step_id":     stepID,
			"destination": destination,
		})
	}

	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return domainpipeline.NewExecutionError("create parent directory", err, map[string]interface{}{
			"step_id":     stepID,
			"parent_dir":  parent,
			"destination": destination,
		})
	}

	return nil
}

func cloneRepository(ctx context.Context, cfg stepConfig, options *git.CloneOptions, stepID string) error {
	if _, err := git.PlainCloneContext(ctx, cfg.Destination, false, options); err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return domainpipeline.NewCancelledError("clone cancelled", map[string]interface{}{"step_id": stepID})
		case errors.Is(err, context.DeadlineExceeded):
			return domainpipeline.NewTimeoutError("clone timed out", err, map[string]interface{}{"step_id": stepID})
		default:
			return domainpipeline.NewExecutionError("clone repository", err, map[string]interface{}{
				"step_id":     stepID,
				"url":         cfg.URL,
				"destination": cfg.Destination,
			})
		}
	}

	return nil
}

func repoAlreadySatisfied(stepID string) *domainpipeline.StepResult {
	return &domainpipeline.StepResult{
		StepID:  stepID,
		Status:  domainpipeline.StatusAlreadySatisfied,
		Message: "repository already up to date",
	}
}

func repoCloned(stepID, url string) *domainpipeline.StepResult {
	return &domainpipeline.StepResult{
		StepID:  stepID,
		Status:  domainpipeline.StatusSuccess,
		Message: fmt.Sprintf("cloned %s", url),
		Changed: true,
	}
}

func ensureEvaluationData(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*evaluationData, error) {
	if evaluation != nil {
		if data, ok := evaluation.InternalData.(*evaluationData); ok && data != nil {
			return data, nil
		}
	}

	eval, err := (&Plugin{}).Evaluate(ctx, step)
	if err != nil {
		return nil, err
	}

	data, ok := eval.InternalData.(*evaluationData)
	if !ok || data == nil {
		return nil, domainpipeline.NewInternalError("repo evaluation missing internal data", nil, map[string]interface{}{"step_id": step.ID})
	}

	return data, nil
}

type stepConfig struct {
	URL         string
	Destination string
	Branch      string
	Depth       int
}

func decodeConfig(step domainpipeline.Step) (stepConfig, error) {
	if strings.TrimSpace(step.ID) == "" {
		return stepConfig{}, domainpipeline.NewValidationError("repo step missing id", map[string]interface{}{"step": step})
	}

	if step.Config == nil {
		return stepConfig{}, domainpipeline.NewValidationError("repo configuration missing", map[string]interface{}{"step_id": step.ID})
	}

	cfg := stepConfig{}

	url, ok := getString(step.Config, "url")
	if !ok || strings.TrimSpace(url) == "" {
		return stepConfig{}, domainpipeline.NewValidationError("repo url is required", map[string]interface{}{"step_id": step.ID})
	}

	cfg.URL = strings.TrimSpace(url)

	dest, ok := getString(step.Config, "destination")
	if !ok || strings.TrimSpace(dest) == "" {
		return stepConfig{}, domainpipeline.NewValidationError("repo destination is required", map[string]interface{}{"step_id": step.ID})
	}

	cfg.Destination = strings.TrimSpace(dest)

	if branch, ok := getString(step.Config, "branch"); ok {
		cfg.Branch = strings.TrimSpace(branch)
	}

	if depth, ok := step.Config["depth"]; ok {
		parsed, err := parseDepth(depth)
		if err != nil {
			return stepConfig{}, domainpipeline.NewValidationError("repo depth must be a non-negative integer", map[string]interface{}{
				"step_id": step.ID,
				"depth":   depth,
			})
		}

		cfg.Depth = parsed
	}

	applyEnvOverrides(&cfg)

	if cfg.Depth < 0 {
		return stepConfig{}, domainpipeline.NewValidationError("repo depth must be non-negative", map[string]interface{}{
			"step_id": step.ID,
			"depth":   cfg.Depth,
		})
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *stepConfig) {
	if path := strings.TrimSpace(os.Getenv("STREAMY_REPO_PATH")); path != "" {
		cfg.Destination = path
	}

	if branch := strings.TrimSpace(os.Getenv("STREAMY_REPO_BRANCH")); branch != "" {
		cfg.Branch = branch
	}

	if depthStr := strings.TrimSpace(os.Getenv("STREAMY_REPO_DEPTH")); depthStr != "" {
		if parsed, err := strconv.Atoi(depthStr); err == nil {
			cfg.Depth = parsed
		}
	}

	if url := strings.TrimSpace(os.Getenv("STREAMY_REPO_URL")); url != "" {
		cfg.URL = url
	}
}

func parseDepth(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("depth cannot be negative: %d", v)
		}

		return v, nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("depth cannot be negative: %d", v)
		}

		return int(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("depth cannot be negative: %d", v)
		}

		return int(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("depth cannot be negative: %f", v)
		}

		frac, _ := math.Modf(v)
		if frac != 0 {
			return 0, fmt.Errorf("depth must be a whole number: %f", v)
		}

		return int(v), nil
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, nil
		}

		trimmed := strings.TrimSpace(v)

		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("parse depth %q: %w", trimmed, err)
		}

		if parsed < 0 {
			return 0, fmt.Errorf("depth cannot be negative: %d", parsed)
		}

		return parsed, nil
	default:
		return 0, fmt.Errorf("invalid depth type %T", value)
	}
}

func getString(values map[string]interface{}, key string) (string, bool) {
	if values == nil {
		return "", false
	}

	raw, ok := values[key]
	if !ok {
		return "", false
	}

	switch v := raw.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func buildCloneOptions(cfg stepConfig) *git.CloneOptions {
	opts := &git.CloneOptions{
		URL: cfg.URL,
	}
	if cfg.Depth > 0 {
		opts.Depth = cfg.Depth
	}

	if cfg.Branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(cfg.Branch)
		opts.SingleBranch = true
	}

	return opts
}

func repoMissing(url string, data *evaluationData) *domainpipeline.EvaluationResult {
	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   string(domainpipeline.VerificationFailed),
		DesiredState:   fmt.Sprintf("repository directory %s does not exist", data.Destination),
		Diff:           fmt.Sprintf("would clone: %s", url),
		InternalData:   data,
	}
}

func repoDrifted(message string, data *evaluationData) *domainpipeline.EvaluationResult {
	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   string(domainpipeline.VerificationFailed),
		DesiredState:   message,
		Diff:           fmt.Sprintf("would clone: %s", data.ExpectedURL),
		InternalData:   data,
	}
}

func urlsEqual(a, b string) bool {
	return strings.TrimSuffix(strings.TrimSpace(a), ".git") == strings.TrimSuffix(strings.TrimSpace(b), ".git")
}

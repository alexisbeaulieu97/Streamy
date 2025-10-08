package main

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestRunVerifyInternal_ConfigParseError(t *testing.T) {
	restoreVerifyDeps(t)

	var stderr bytes.Buffer
	stderrWriter = &stderr
	parseConfigFunc = func(string) (*config.Config, error) {
		return nil, errors.New("parse failure")
	}

	code, err := runVerifyInternal(verifyOptions{ConfigPath: "broken.yml"})
	require.NoError(t, err)
	require.Equal(t, 2, code)
	require.Contains(t, stderr.String(), "Error parsing configuration")
}

func TestRunVerifyInternal_SuccessTableOutput(t *testing.T) {
	restoreVerifyDeps(t)

	parseConfigFunc = func(string) (*config.Config, error) {
		return &config.Config{
			Steps: []config.Step{
				{ID: "step1", Type: "command"},
			},
		}, nil
	}
	newLoggerFunc = func(opts logger.Options) (*logger.Logger, error) {
		opts.Writer = io.Discard
		return logger.New(opts)
	}
	getRegistryFunc = func() *plugin.PluginRegistry { return nil }

	summary := &model.VerificationSummary{
		TotalSteps: 1,
		Satisfied:  1,
		Results: []*model.VerificationResult{
			{
				StepID:    "step1",
				Status:    model.StatusSatisfied,
				Message:   "ok",
				Duration:  time.Second,
				Timestamp: time.Now(),
			},
		},
		Duration: time.Second,
	}
	newExecutorFunc = func(*logger.Logger) verificationExecutor {
		return &fakeVerificationExecutor{summary: summary, err: nil}
	}

	var tableCalls int
	printTableOutputFunc = func(*model.VerificationSummary) {
		tableCalls++
	}

	code, err := runVerifyInternal(verifyOptions{
		ConfigPath: "streamy.yml",
		Timeout:    time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, 0, code)
	require.Equal(t, 1, tableCalls)
}

func TestRunVerifyInternal_VerboseOutput(t *testing.T) {
	restoreVerifyDeps(t)

	parseConfigFunc = func(string) (*config.Config, error) {
		return &config.Config{Steps: []config.Step{}}, nil
	}
	newLoggerFunc = func(opts logger.Options) (*logger.Logger, error) {
		opts.Writer = io.Discard
		return logger.New(opts)
	}
	newExecutorFunc = func(*logger.Logger) verificationExecutor {
		return &fakeVerificationExecutor{
			summary: &model.VerificationSummary{
				TotalSteps: 0,
				Satisfied:  0,
				Results:    []*model.VerificationResult{},
				Duration:   0,
			},
			err: nil,
		}
	}

	var verboseCalls int
	printVerboseOutputFunc = func(*model.VerificationSummary) {
		verboseCalls++
	}

	code, err := runVerifyInternal(verifyOptions{
		ConfigPath: "streamy.yml",
		Verbose:    true,
	})
	require.NoError(t, err)
	require.Equal(t, 0, code)
	require.Equal(t, 1, verboseCalls)
}

func TestRunVerifyInternal_JSONOutputError(t *testing.T) {
	restoreVerifyDeps(t)

	parseConfigFunc = func(string) (*config.Config, error) {
		return &config.Config{Steps: []config.Step{}}, nil
	}
	newLoggerFunc = func(opts logger.Options) (*logger.Logger, error) {
		opts.Writer = io.Discard
		return logger.New(opts)
	}
	newExecutorFunc = func(*logger.Logger) verificationExecutor {
		return &fakeVerificationExecutor{
			summary: &model.VerificationSummary{
				TotalSteps: 0,
				Satisfied:  0,
				Results:    []*model.VerificationResult{},
				Duration:   0,
			},
			err: nil,
		}
	}

	printJSONOutputFunc = func(*model.VerificationSummary, string) error {
		return errors.New("write failure")
	}

	code, err := runVerifyInternal(verifyOptions{
		ConfigPath: "streamy.yml",
		JSON:       true,
	})
	require.NoError(t, err)
	require.Equal(t, 3, code)
}

func TestRunVerifyInternal_ValidationError(t *testing.T) {
	restoreVerifyDeps(t)

	parseConfigFunc = func(string) (*config.Config, error) {
		return &config.Config{Steps: []config.Step{}}, nil
	}
	newLoggerFunc = func(opts logger.Options) (*logger.Logger, error) {
		opts.Writer = io.Discard
		return logger.New(opts)
	}
	newExecutorFunc = func(*logger.Logger) verificationExecutor {
		return &fakeVerificationExecutor{
			err: streamyerrors.NewValidationError("step", "invalid", nil),
		}
	}

	var stderr bytes.Buffer
	stderrWriter = &stderr

	code, err := runVerifyInternal(verifyOptions{ConfigPath: "streamy.yml"})
	require.NoError(t, err)
	require.Equal(t, 2, code)
	require.Contains(t, stderr.String(), "Configuration error")
}

func TestRunVerifyInternal_ExecutionError(t *testing.T) {
	restoreVerifyDeps(t)

	parseConfigFunc = func(string) (*config.Config, error) {
		return &config.Config{Steps: []config.Step{}}, nil
	}
	newLoggerFunc = func(opts logger.Options) (*logger.Logger, error) {
		opts.Writer = io.Discard
		return logger.New(opts)
	}
	newExecutorFunc = func(*logger.Logger) verificationExecutor {
		return &fakeVerificationExecutor{
			err: errors.New("boom"),
		}
	}

	var stderr bytes.Buffer
	stderrWriter = &stderr

	code, err := runVerifyInternal(verifyOptions{ConfigPath: "streamy.yml"})
	require.NoError(t, err)
	require.Equal(t, 3, code)
	require.Contains(t, stderr.String(), "Verification error")
}

type fakeVerificationExecutor struct {
	summary *model.VerificationSummary
	err     error
}

func (f *fakeVerificationExecutor) VerifySteps(_ *engine.ExecutionContext, _ []config.Step, _ time.Duration) (*model.VerificationSummary, error) {
	return f.summary, f.err
}

func restoreVerifyDeps(t *testing.T) {
	origParse := parseConfigFunc
	origLogger := newLoggerFunc
	origExecutor := newExecutorFunc
	origRegistry := getRegistryFunc
	origExit := exitFunc
	origStderr := stderrWriter
	origTable := printTableOutputFunc
	origVerbose := printVerboseOutputFunc
	origJSON := printJSONOutputFunc

	t.Cleanup(func() {
		parseConfigFunc = origParse
		newLoggerFunc = origLogger
		newExecutorFunc = origExecutor
		getRegistryFunc = origRegistry
		exitFunc = origExit
		stderrWriter = origStderr
		printTableOutputFunc = origTable
		printVerboseOutputFunc = origVerbose
		printJSONOutputFunc = origJSON
	})
}

package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	streamconfig "github.com/alexisbeaulieu97/streamy/internal/config"
	streamengine "github.com/alexisbeaulieu97/streamy/internal/engine"
	streamlogger "github.com/alexisbeaulieu97/streamy/internal/logger"
	streammodel "github.com/alexisbeaulieu97/streamy/internal/model"
	streamplugin "github.com/alexisbeaulieu97/streamy/internal/plugin"

	linefileinplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"
)

func TestIntegration_LineInFile_FreshProfile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	profile := filepath.Join(dir, ".bashrc")

	cfg := baseLineInFileConfig([]streamconfig.Step{
		{
			ID:      "add_path",
			Type:    "line_in_file",
			Enabled: true,
			LineInFile: &streamconfig.LineInFileStep{
				File:  profile,
				Line:  "export PATH=\"$PATH:~/bin\"",
				State: "present",
			},
		},
	})

	results := runLineInFilePlan(t, cfg, false)
	require.Len(t, results, 1)
	assert.Equal(t, streammodel.StatusSuccess, resultByID(t, results, "add_path").Status)

	content, err := os.ReadFile(profile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "export PATH=\"$PATH:~/bin\"")

	results = runLineInFilePlan(t, cfg, false)
	require.Len(t, results, 1)
	assert.Equal(t, streammodel.StatusSkipped, resultByID(t, results, "add_path").Status)
}

func TestIntegration_LineInFile_ReplaceDebug(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "app.ini")
	writeTempFile(t, configPath, "debug=true\nmode=prod\n")

	cfg := baseLineInFileConfig([]streamconfig.Step{
		{
			ID:      "replace_debug",
			Type:    "line_in_file",
			Enabled: true,
			LineInFile: &streamconfig.LineInFileStep{
				File:              configPath,
				Line:              "debug=false",
				State:             "present",
				Match:             "^debug=",
				OnMultipleMatches: "first",
			},
		},
	})

	results := runLineInFilePlan(t, cfg, false)
	require.Len(t, results, 1)
	assert.Equal(t, streammodel.StatusSuccess, resultByID(t, results, "replace_debug").Status)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "debug=false\nmode=prod\n", string(content))
}

func TestIntegration_LineInFile_RemoveMultiple(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "profile")
	writeTempFile(t, configPath, "export OLD_VAR=1\nexport OLD_VAR=2\nexport KEEP=1\n")

	cfg := baseLineInFileConfig([]streamconfig.Step{
		{
			ID:      "remove_old",
			Type:    "line_in_file",
			Enabled: true,
			LineInFile: &streamconfig.LineInFileStep{
				File:  configPath,
				State: "absent",
				Match: "^export OLD_VAR=",
			},
		},
	})

	results := runLineInFilePlan(t, cfg, false)
	require.Len(t, results, 1)
	assert.Equal(t, streammodel.StatusSuccess, resultByID(t, results, "remove_old").Status)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "export KEEP=1\n", string(content))
}

func TestIntegration_LineInFile_BackupVerify(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "settings.conf")
	writeTempFile(t, configPath, "option=old\n")

	cfg := baseLineInFileConfig([]streamconfig.Step{
		{
			ID:      "update_option",
			Type:    "line_in_file",
			Enabled: true,
			LineInFile: &streamconfig.LineInFileStep{
				File:      configPath,
				Line:      "option=new",
				State:     "present",
				Match:     "^option=",
				Backup:    true,
				BackupDir: filepath.Join(dir, "backups"),
			},
		},
	})

	results := runLineInFilePlan(t, cfg, false)
	require.Len(t, results, 1)
	assert.Equal(t, streammodel.StatusSuccess, resultByID(t, results, "update_option").Status)

	backups, err := filepath.Glob(filepath.Join(dir, "backups", "settings.conf.*.bak"))
	require.NoError(t, err)
	require.Len(t, backups, 1)
	data, err := os.ReadFile(backups[0])
	require.NoError(t, err)
	assert.Equal(t, "option=old\n", string(data))
}

func TestIntegration_LineInFile_CompleteShellSetup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	writeTempFile(t, profile, "export JAVA_HOME=/usr/lib/jvm\n")

	cfg := baseLineInFileConfig([]streamconfig.Step{
		{
			ID:      "add_path",
			Type:    "line_in_file",
			Enabled: true,
			LineInFile: &streamconfig.LineInFileStep{
				File:  profile,
				Line:  "export PATH=\"$PATH:/opt/dev/bin\"",
				State: "present",
			},
		},
		{
			ID:        "set_editor",
			Type:      "line_in_file",
			Enabled:   true,
			DependsOn: []string{"add_path"},
			LineInFile: &streamconfig.LineInFileStep{
				File:  profile,
				Line:  "export EDITOR=vim",
				State: "present",
			},
		},
		{
			ID:        "remove_old_java",
			Type:      "line_in_file",
			Enabled:   true,
			DependsOn: []string{"set_editor"},
			LineInFile: &streamconfig.LineInFileStep{
				File:  profile,
				State: "absent",
				Match: "^export JAVA_HOME=",
			},
		},
		{
			ID:        "set_java",
			Type:      "line_in_file",
			Enabled:   true,
			DependsOn: []string{"remove_old_java"},
			LineInFile: &streamconfig.LineInFileStep{
				File:  profile,
				Line:  "export JAVA_HOME=/opt/java",
				State: "present",
			},
		},
	})

	results := runLineInFilePlan(t, cfg, false)
	require.Len(t, results, 4)
	for _, id := range []string{"add_path", "set_editor", "remove_old_java", "set_java"} {
		assert.Equal(t, streammodel.StatusSuccess, resultByID(t, results, id).Status)
	}

	content, err := os.ReadFile(profile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "export PATH=\"$PATH:/opt/dev/bin\"")
	assert.Contains(t, string(content), "export EDITOR=vim")
	assert.Contains(t, string(content), "export JAVA_HOME=/opt/java")
	assert.NotContains(t, string(content), "export JAVA_HOME=/usr/lib/jvm")
}

func TestIntegration_LineInFile_DryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	profile := filepath.Join(dir, ".profile")
	writeTempFile(t, profile, "alias ll='ls -al'\n")

	cfg := baseLineInFileConfig([]streamconfig.Step{
		{
			ID:      "remove_alias",
			Type:    "line_in_file",
			Enabled: true,
			LineInFile: &streamconfig.LineInFileStep{
				File:  profile,
				State: "absent",
				Match: "^alias ll",
			},
		},
	})

	results := runLineInFilePlan(t, cfg, true)
	require.Len(t, results, 1)
	res := resultByID(t, results, "remove_alias")
	assert.Equal(t, streammodel.StatusWouldUpdate, res.Status)

	content, err := os.ReadFile(profile)
	require.NoError(t, err)
	assert.Equal(t, "alias ll='ls -al'\n", string(content))
}

func baseLineInFileConfig(steps []streamconfig.Step) *streamconfig.Config {
	return &streamconfig.Config{
		Version: "1.0",
		Name:    "line-in-file-integration",
		Steps:   steps,
	}
}

func runLineInFilePlan(t *testing.T, cfg *streamconfig.Config, dryRun bool) []streammodel.StepResult {
	t.Helper()

	graph := buildDAG(t, cfg)
	plan := generatePlan(t, graph)

	logger := testLogger(t)
	if logger == nil {
		var err error
		logger, err = streamlogger.New(streamlogger.Options{Level: "info", HumanReadable: false})
		require.NoError(t, err)
	}

	registry := streamplugin.NewPluginRegistry(streamplugin.DefaultConfig(), logger)
	require.NoError(t, registry.Register(linefileinplugin.New()))
	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())

	execCtx := &streamengine.ExecutionContext{
		Config:     cfg,
		DryRun:     dryRun,
		WorkerPool: make(chan struct{}, len(cfg.Steps)),
		Results:    make(map[string]*streammodel.StepResult),
		Logger:     logger,
		Context:    context.Background(),
		Registry:   registry,
	}

	results, err := streamengine.Execute(execCtx, plan)
	require.NoError(t, err)
	return results
}

func resultByID(t *testing.T, results []streammodel.StepResult, id string) streammodel.StepResult {
	t.Helper()
	for _, res := range results {
		if res.StepID == id {
			return res
		}
	}
	t.Fatalf("result for step %s not found", id)
	return streammodel.StepResult{}
}

func writeTempFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

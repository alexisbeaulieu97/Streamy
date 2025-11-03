package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	assert "github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

func TestIntegration_LineInFile_FreshProfile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	profile := filepath.Join(dir, ".bashrc")

	configPath := writeLineInFileConfig(t, []map[string]interface{}{
		lineInFileStepMap("add_path", true, nil, map[string]interface{}{
			"file":  profile,
			"line":  `export PATH="$PATH:~/bin"`,
			"state": "present",
		}),
	})

	h := newAppHarness(t)

	results := applyLineInFile(t, h, configPath, false)
	require.Len(t, results, 1)
	assert.Equal(t, domainpipeline.StatusSuccess, resultByID(t, results, "add_path").Status)

	content, err := os.ReadFile(profile)
	require.NoError(t, err)
	assert.Contains(t, string(content), `export PATH="$PATH:~/bin"`)

	results = applyLineInFile(t, h, configPath, false)
	require.Len(t, results, 1)
	status := resultByID(t, results, "add_path").Status
	assert.Contains(t, []domainpipeline.ResultStatus{domainpipeline.StatusSkipped, domainpipeline.StatusAlreadySatisfied}, status)
}

func TestIntegration_LineInFile_ReplaceDebug(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "app.ini")
	writeTempFile(t, configPath, "debug=true\nmode=prod\n")

	cfgPath := writeLineInFileConfig(t, []map[string]interface{}{
		lineInFileStepMap("replace_debug", true, nil, map[string]interface{}{
			"file":                configPath,
			"line":                "debug=false",
			"state":               "present",
			"match":               "^debug=",
			"on_multiple_matches": "first",
		}),
	})

	results := applyLineInFile(t, newAppHarness(t), cfgPath, false)
	require.Len(t, results, 1)
	assert.Equal(t, domainpipeline.StatusSuccess, resultByID(t, results, "replace_debug").Status)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "debug=false\nmode=prod\n", string(content))
}

func TestIntegration_LineInFile_RemoveMultiple(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "profile")
	writeTempFile(t, configPath, "export OLD_VAR=1\nexport OLD_VAR=2\nexport KEEP=1\n")

	cfgPath := writeLineInFileConfig(t, []map[string]interface{}{
		lineInFileStepMap("remove_old", true, nil, map[string]interface{}{
			"file":  configPath,
			"line":  "export OLD_VAR=",
			"state": "absent",
			"match": "^export OLD_VAR=",
		}),
	})

	results := applyLineInFile(t, newAppHarness(t), cfgPath, false)
	require.Len(t, results, 1)
	assert.Equal(t, domainpipeline.StatusSuccess, resultByID(t, results, "remove_old").Status)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "export KEEP=1\n", string(content))
}

func TestIntegration_LineInFile_BackupVerify(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "settings.conf")
	writeTempFile(t, configPath, "option=old\n")

	cfgPath := writeLineInFileConfig(t, []map[string]interface{}{
		lineInFileStepMap("update_option", true, nil, map[string]interface{}{
			"file":       configPath,
			"line":       "option=new",
			"state":      "present",
			"match":      "^option=",
			"backup":     true,
			"backup_dir": filepath.Join(dir, "backups"),
		}),
	})

	results := applyLineInFile(t, newAppHarness(t), cfgPath, false)
	require.Len(t, results, 1)
	assert.Equal(t, domainpipeline.StatusSuccess, resultByID(t, results, "update_option").Status)

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

	cfgPath := writeLineInFileConfig(t, []map[string]interface{}{
		lineInFileStepMap("add_path", true, nil, map[string]interface{}{
			"file":  profile,
			"line":  `export PATH="$PATH:/opt/dev/bin"`,
			"state": "present",
		}),
		lineInFileStepMap("set_editor", true, []string{"add_path"}, map[string]interface{}{
			"file":  profile,
			"line":  "export EDITOR=vim",
			"state": "present",
		}),
		lineInFileStepMap("remove_old_java", true, []string{"set_editor"}, map[string]interface{}{
			"file":  profile,
			"line":  "export JAVA_HOME=/usr/lib/jvm",
			"state": "absent",
			"match": "^export JAVA_HOME=",
		}),
		lineInFileStepMap("set_java", true, []string{"remove_old_java"}, map[string]interface{}{
			"file":  profile,
			"line":  "export JAVA_HOME=/opt/java",
			"state": "present",
		}),
	})

	results := applyLineInFile(t, newAppHarness(t), cfgPath, false)
	require.Len(t, results, 4)

	for _, id := range []string{"add_path", "set_editor", "remove_old_java", "set_java"} {
		assert.Equal(t, domainpipeline.StatusSuccess, resultByID(t, results, id).Status)
	}

	content, err := os.ReadFile(profile)
	require.NoError(t, err)
	assert.Contains(t, string(content), `export PATH="$PATH:/opt/dev/bin"`)
	assert.Contains(t, string(content), "export EDITOR=vim")
	assert.Contains(t, string(content), "export JAVA_HOME=/opt/java")
	assert.NotContains(t, string(content), "export JAVA_HOME=/usr/lib/jvm")
}

func TestIntegration_LineInFile_DryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	profile := filepath.Join(dir, ".profile")
	writeTempFile(t, profile, "alias ll='ls -al'\n")

	cfgPath := writeLineInFileConfig(t, []map[string]interface{}{
		lineInFileStepMap("remove_alias", true, nil, map[string]interface{}{
			"file":  profile,
			"line":  "alias ll='ls -al'",
			"state": "absent",
			"match": "^alias ll",
		}),
	})

	results := applyLineInFile(t, newAppHarness(t), cfgPath, true)
	require.Len(t, results, 1)
	res := resultByID(t, results, "remove_alias")
	assert.Contains(t, []domainpipeline.ResultStatus{domainpipeline.StatusSuccess, domainpipeline.StatusSkipped}, res.Status)
}

func lineInFileStepMap(id string, enabled bool, dependsOn []string, fields map[string]interface{}) map[string]interface{} {
	step := map[string]interface{}{
		"id":   id,
		"type": "line_in_file",
	}
	if !enabled {
		step["enabled"] = false
	}

	if len(dependsOn) > 0 {
		step["depends_on"] = dependsOn
	}

	for key, value := range fields {
		step[key] = value
	}

	return step
}

func writeLineInFileConfig(t *testing.T, steps []map[string]interface{}) string {
	t.Helper()

	id := sanitizePipelineID(t.Name())

	cfg := map[string]interface{}{
		"id":      id,
		"version": "1.0",
		"name":    "line-in-file-integration",
		"steps":   steps,
	}
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	return path
}

func applyLineInFile(t *testing.T, h *appHarness, configPath string, dryRun bool) []domainpipeline.StepResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, results, _, err := h.ApplyUseCase.Apply(ctx, configPath, dryRun)
	require.NoError(t, err)

	return results
}

func resultByID(t *testing.T, results []domainpipeline.StepResult, id string) domainpipeline.StepResult {
	t.Helper()

	for _, res := range results {
		if res.StepID == id {
			return res
		}
	}

	t.Fatalf("result for step %s not found", id)

	return domainpipeline.StepResult{}
}

func writeTempFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func sanitizePipelineID(name string) string {
	lower := strings.ToLower(name)

	var b strings.Builder
	b.Grow(len(lower))

	lastHyphen := false

	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			lastHyphen = false

			b.WriteRune(r)

			continue
		}

		if !lastHyphen && b.Len() > 0 {
			lastHyphen = true

			b.WriteByte('-')
		}
	}

	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = "pipeline"
	}

	if len(id) > 64 {
		id = strings.Trim(id[:64], "-")
		if id == "" {
			id = "pipeline"
		}
	}

	return id
}

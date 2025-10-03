package packageplugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestPackagePlugin_CheckReportsInstalledPackages(t *testing.T) {
	binDir := t.TempDir()
	writeScript(t, binDir, "dpkg-query", `#!/bin/sh
if echo "$@" | grep -q missing_pkg; then
  exit 1
fi
echo "install ok installed"
exit 0
`)

	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

	p := New()
	require.Implements(t, (*pluginpkg.Plugin)(nil), p)

	step := &config.Step{
		ID:   "install_tools",
		Type: "package",
		Package: &config.PackageStep{
			Packages: []string{"git", "curl"},
		},
	}

	installed, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.True(t, installed, "expected packages to be considered installed")
}

func TestPackagePlugin_CheckDetectsMissingPackage(t *testing.T) {
	binDir := t.TempDir()
	writeScript(t, binDir, "dpkg-query", `#!/bin/sh
echo "$@" | grep -q missing_pkg
if [ $? -eq 0 ]; then
  exit 1
fi
echo "install ok installed"
exit 0
`)

	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

	p := New()

	step := &config.Step{
		ID:   "install_tools",
		Type: "package",
		Package: &config.PackageStep{
			Packages: []string{"git", "missing_pkg"},
		},
	}

	installed, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, installed, "expected missing package to be detected")
}

func TestPackagePlugin_ApplyRunsAptGet(t *testing.T) {
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "apt.log")
	writeScript(t, binDir, "dpkg-query", `#!/bin/sh
echo "install ok installed"
exit 0
`)
	writeScript(t, binDir, "apt-get", `#!/bin/sh
echo "$0 $@" >> "`+logPath+`"
exit 0
`)

	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

	p := New()

	step := &config.Step{
		ID:   "install_tools",
		Type: "package",
		Package: &config.PackageStep{
			Packages: []string{"git", "curl"},
		},
	}

	result, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, result.StepID)
	require.Equal(t, "success", strings.ToLower(result.Status))

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	output := string(data)
	require.Contains(t, output, "apt-get install")
	require.Contains(t, output, "git")
	require.Contains(t, output, "curl")
}

func TestPackagePlugin_DryRunSkipsExecution(t *testing.T) {
	binDir := t.TempDir()
	writeScript(t, binDir, "dpkg-query", `#!/bin/sh
echo "install ok installed"
exit 0
`)
	writeScript(t, binDir, "apt-get", `#!/bin/sh
exit 1
`)

	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

	p := New()

	step := &config.Step{
		ID:   "install_tools",
		Type: "package",
		Package: &config.PackageStep{
			Packages: []string{"git"},
		},
	}

	result, err := p.DryRun(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "install_tools", result.StepID)
	require.Equal(t, "skipped", strings.ToLower(result.Status))
	require.Contains(t, strings.ToLower(result.Message), "dry")
}

func writeScript(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o755))
}

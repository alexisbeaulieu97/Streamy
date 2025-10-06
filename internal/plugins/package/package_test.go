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

func TestPackagePlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.Metadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "package", meta.Type)
}

func TestPackagePlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.PackageStep)
	require.True(t, ok, "schema should be of type PackageStep")
}

func TestPackagePlugin_ApplyInstallsMultiplePackages(t *testing.T) {
	binDir := t.TempDir()
	writeScript(t, binDir, "apt-get", `#!/bin/sh
echo "Installing: $@"
exit 0
`)

	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

	step := &config.Step{
		ID:   "install_packages",
		Type: "package",
		Package: &config.PackageStep{
			Packages: []string{"git", "curl", "vim"},
			Manager:  "apt",
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", strings.ToLower(res.Status))
}

func TestPackagePlugin_CheckMultiplePackages(t *testing.T) {
	binDir := t.TempDir()
	writeScript(t, binDir, "dpkg-query", `#!/bin/sh
# Simulate git and curl installed, but not vim
if echo "$@" | grep -q "vim"; then
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
		ID:   "check_packages",
		Type: "package",
		Package: &config.PackageStep{
			Packages: []string{"git", "curl"},
			Manager:  "apt",
		},
	}

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.True(t, ok, "expected all packages to be installed")

	step.Package.Packages = []string{"git", "curl", "vim"}
	ok, err = p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok, "expected Check to return false when some packages missing")
}

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

func TestPackagePlugin_Verify(t *testing.T) {
	t.Run("returns satisfied when all packages installed", func(t *testing.T) {
		binDir := t.TempDir()
		writeScript(t, binDir, "dpkg-query", `#!/bin/sh
echo "install ok installed"
exit 0
`)

		originalPath := os.Getenv("PATH")
		t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
		require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

		p := New()

		step := &config.Step{
			ID:   "install_packages",
			Type: "package",
			Package: &config.PackageStep{
				Packages: []string{"git", "curl"},
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
		require.Contains(t, result.Message, "all packages installed")
	})

	t.Run("returns missing when packages not installed", func(t *testing.T) {
		binDir := t.TempDir()
		writeScript(t, binDir, "dpkg-query", `#!/bin/sh
if echo "$@" | grep -q "vim"; then
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
			ID:   "install_packages",
			Type: "package",
			Package: &config.PackageStep{
				Packages: []string{"git", "vim"},
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "missing", string(result.Status))
		require.Contains(t, result.Message, "packages not installed")
		require.Contains(t, result.Message, "vim")
	})

	t.Run("returns blocked when context is cancelled", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "install_packages",
			Type: "package",
			Package: &config.PackageStep{
				Packages: []string{"git"},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := p.Verify(ctx, step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "blocked", string(result.Status))
		require.Contains(t, result.Message, "cancelled")
		require.NotNil(t, result.Error)
	})

	t.Run("returns error when package config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:      "install_packages",
			Type:    "package",
			Package: nil,
		}

		_, err := p.Verify(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "package configuration missing")
	})
}

func TestPackagePlugin_Check_Errors(t *testing.T) {
	t.Run("returns error when package config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:      "check_packages",
			Type:    "package",
			Package: nil,
		}

		_, err := p.Check(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "package configuration missing")
	})
}

func TestPackagePlugin_Apply_Errors(t *testing.T) {
	t.Run("returns error when package config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:      "install_packages",
			Type:    "package",
			Package: nil,
		}

		_, err := p.Apply(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "package configuration missing")
	})

	t.Run("returns error when apt-get fails", func(t *testing.T) {
		binDir := t.TempDir()
		writeScript(t, binDir, "apt-get", `#!/bin/sh
echo "Failed to install"
exit 1
`)

		originalPath := os.Getenv("PATH")
		t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
		require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

		p := New()

		step := &config.Step{
			ID:   "install_packages",
			Type: "package",
			Package: &config.PackageStep{
				Packages: []string{"nonexistent"},
			},
		}

		result, err := p.Apply(context.Background(), step)
		require.Error(t, err)
		require.NotNil(t, result)
		require.Equal(t, "failed", result.Status)
	})
}

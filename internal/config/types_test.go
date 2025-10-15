package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestStepUnmarshalYAML(t *testing.T) {
	t.Parallel()

	t.Run("unmarshals package step", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: install_git
type: package
packages:
  - git
  - curl
manager: apt
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.Equal(t, "install_git", step.ID)
		require.Equal(t, "package", step.Type)
		require.True(t, step.Enabled)
		var cfg PackageStep
		require.NoError(t, step.DecodeConfig(&cfg))
		require.Equal(t, []string{"git", "curl"}, cfg.Packages)
		require.Equal(t, "apt", cfg.Manager)
	})

	t.Run("unmarshals repo step", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: clone_repo
type: repo
url: https://github.com/example/repo.git
destination: /tmp/repo
branch: main
depth: 1
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.Equal(t, "clone_repo", step.ID)
		require.Equal(t, "repo", step.Type)
		var cfg RepoStep
		require.NoError(t, step.DecodeConfig(&cfg))
		require.Equal(t, "https://github.com/example/repo.git", cfg.URL)
		require.Equal(t, "/tmp/repo", cfg.Destination)
		require.Equal(t, "main", cfg.Branch)
		require.Equal(t, 1, cfg.Depth)
	})

	t.Run("unmarshals symlink step", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: link_config
type: symlink
source: /etc/config
target: /home/user/.config
force: true
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.Equal(t, "link_config", step.ID)
		require.Equal(t, "symlink", step.Type)
		var cfg SymlinkStep
		require.NoError(t, step.DecodeConfig(&cfg))
		require.Equal(t, "/etc/config", cfg.Source)
		require.Equal(t, "/home/user/.config", cfg.Target)
		require.True(t, cfg.Force)
	})

	t.Run("unmarshals copy step", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: copy_files
type: copy
source: /tmp/src
destination: /tmp/dst
recursive: true
overwrite: true
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.Equal(t, "copy_files", step.ID)
		require.Equal(t, "copy", step.Type)
		var cfg CopyStep
		require.NoError(t, step.DecodeConfig(&cfg))
		require.Equal(t, "/tmp/src", cfg.Source)
		require.Equal(t, "/tmp/dst", cfg.Destination)
		require.True(t, cfg.Recursive)
		require.True(t, cfg.Overwrite)
	})

	t.Run("unmarshals command step", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: run_script
type: command
command: echo hello
check: which echo
shell: /bin/bash
workdir: /tmp
env:
  FOO: bar
  BAZ: qux
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.Equal(t, "run_script", step.ID)
		require.Equal(t, "command", step.Type)
		var cfg CommandStep
		require.NoError(t, step.DecodeConfig(&cfg))
		require.Equal(t, "echo hello", cfg.Command)
		require.Equal(t, "which echo", cfg.Check)
		require.Equal(t, "/bin/bash", cfg.Shell)
		require.Equal(t, "/tmp", cfg.WorkDir)
		require.Equal(t, map[string]string{"FOO": "bar", "BAZ": "qux"}, cfg.Env)

		raw := step.RawConfig()
		require.Equal(t, map[string]any{
			"command": "echo hello",
			"check":   "which echo",
			"shell":   "/bin/bash",
			"workdir": "/tmp",
			"env": map[string]any{
				"FOO": "bar",
				"BAZ": "qux",
			},
		}, raw)

		// Mutating the returned map must not change the internal state.
		raw["command"] = "mutated"
		fresh := step.RawConfig()
		require.Equal(t, "echo hello", fresh["command"])

		var decoded CommandStep
		err = step.DecodeConfig(&decoded)
		require.NoError(t, err)
		require.Equal(t, "echo hello", decoded.Command)
		require.Equal(t, "which echo", decoded.Check)
		require.Equal(t, "/bin/bash", decoded.Shell)
		require.Equal(t, "/tmp", decoded.WorkDir)
		require.Equal(t, map[string]string{"FOO": "bar", "BAZ": "qux"}, decoded.Env)
	})

	t.Run("unmarshals step with enabled=false", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: disabled_step
type: command
enabled: false
command: echo skip
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.Equal(t, "disabled_step", step.ID)
		require.False(t, step.Enabled)
	})

	t.Run("defaults enabled to true when not specified", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: default_enabled
type: command
command: echo test
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.True(t, step.Enabled)
	})

	t.Run("unmarshals step with depends_on", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: dependent_step
type: command
command: echo after
depends_on:
  - step1
  - step2
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.Equal(t, []string{"step1", "step2"}, step.DependsOn)
	})

	t.Run("unmarshals step with empty depends_on", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: independent_step
type: command
command: echo standalone
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)
		require.Empty(t, step.DependsOn)
	})

	t.Run("handles malformed yaml", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
id: bad_step
type: package
packages: not_a_list
`
		var step Step
		err := yaml.Unmarshal([]byte(yamlStr), &step)
		require.NoError(t, err)

		err = ValidateStep(step)
		require.Error(t, err)
	})
}

func TestStepSetConfigRoundTrip(t *testing.T) {
	t.Parallel()

	mode := uint32(0o644)

	tests := []struct {
		name     string
		stepID   string
		stepType string
		cfg      any
	}{
		{
			name:     "package step",
			stepID:   "pkg",
			stepType: "package",
			cfg:      PackageStep{Packages: []string{"git", "curl"}, Manager: "apt", Update: true},
		},
		{
			name:     "repo step",
			stepID:   "repo",
			stepType: "repo",
			cfg:      RepoStep{URL: "https://example.com/repo.git", Destination: "/tmp/repo", Branch: "main", Depth: 1},
		},
		{
			name:     "symlink step",
			stepID:   "symlink",
			stepType: "symlink",
			cfg:      SymlinkStep{Source: "/tmp/source", Target: "/tmp/target", Force: true},
		},
		{
			name:     "copy step",
			stepID:   "copy",
			stepType: "copy",
			cfg:      CopyStep{Source: "/tmp/source", Destination: "/tmp/destination", Recursive: true, Overwrite: true, PreserveMode: true, PreserveModeSet: true},
		},
		{
			name:     "command step",
			stepID:   "command",
			stepType: "command",
			cfg:      CommandStep{Command: "echo hello", Check: "which echo", Shell: "/bin/bash", WorkDir: "/tmp", Env: map[string]string{"FOO": "bar"}},
		},
		{
			name:     "template step",
			stepID:   "template",
			stepType: "template",
			cfg:      TemplateStep{Source: "./template.tmpl", Destination: "./output", Vars: map[string]string{"NAME": "Streamy"}, Env: true, AllowMissing: true, Mode: &mode},
		},
		{
			name:     "line in file step",
			stepID:   "line",
			stepType: "line_in_file",
			cfg:      LineInFileStep{File: "/tmp/file", Line: "some line", State: "present", Match: "", Backup: true, BackupDir: "/tmp/backup", Encoding: "utf-8"},
		},
	}

	// Core step-level keys that must never leak into RawConfig.
	forbiddenKeys := []string{"id", "name", "type", "depends_on", "enabled", "verify_timeout"}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			step := Step{ID: tt.stepID, Type: tt.stepType}
			require.NoError(t, step.SetConfig(tt.cfg))

			decodedPtr := reflect.New(reflect.TypeOf(tt.cfg))
			require.NoError(t, step.DecodeConfig(decodedPtr.Interface()))
			require.Equal(t, tt.cfg, decodedPtr.Elem().Interface())

			raw := step.RawConfig()
			for _, key := range forbiddenKeys {
				require.NotContains(t, raw, key, "rawConfig leaked core step field")
			}
		})
	}
}

func TestCopyStepUnmarshalYAML(t *testing.T) {
	t.Parallel()

	t.Run("defaults preserve_mode to true when not specified", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /tmp/src
destination: /tmp/dst
`
		var copy CopyStep
		err := yaml.Unmarshal([]byte(yamlStr), &copy)
		require.NoError(t, err)
		require.True(t, copy.PreserveMode)
		require.False(t, copy.PreserveModeSet)
	})

	t.Run("respects preserve_mode=false when specified", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /tmp/src
destination: /tmp/dst
preserve_mode: false
`
		var copy CopyStep
		err := yaml.Unmarshal([]byte(yamlStr), &copy)
		require.NoError(t, err)
		require.False(t, copy.PreserveMode)
		require.True(t, copy.PreserveModeSet)
	})

	t.Run("respects preserve_mode=true when specified", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /tmp/src
destination: /tmp/dst
preserve_mode: true
`
		var copy CopyStep
		err := yaml.Unmarshal([]byte(yamlStr), &copy)
		require.NoError(t, err)
		require.True(t, copy.PreserveMode)
		require.True(t, copy.PreserveModeSet)
	})

	t.Run("handles case variations in preserve_mode key", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /tmp/src
destination: /tmp/dst
Preserve_Mode: false
`
		var copy CopyStep
		err := yaml.Unmarshal([]byte(yamlStr), &copy)
		require.NoError(t, err)
		require.False(t, copy.PreserveMode)
		require.True(t, copy.PreserveModeSet)
	})
}

func TestStepMap(t *testing.T) {
	t.Parallel()

	t.Run("creates map from empty slice", func(t *testing.T) {
		t.Parallel()
		m := StepMap(nil)
		require.Empty(t, m)
	})

	t.Run("creates map from single step", func(t *testing.T) {
		t.Parallel()
		step := Step{ID: "step1", Type: "command"}
		require.NoError(t, step.SetConfig(CommandStep{Command: "echo test"}))
		m := StepMap([]Step{step})
		require.Len(t, m, 1)
		require.Contains(t, m, "step1")
		require.Equal(t, "command", m["step1"].Type)
	})

	t.Run("creates map from multiple steps", func(t *testing.T) {
		t.Parallel()
		step1 := Step{ID: "step1", Type: "command"}
		require.NoError(t, step1.SetConfig(CommandStep{Command: "echo 1"}))
		step2 := Step{ID: "step2", Type: "command"}
		require.NoError(t, step2.SetConfig(CommandStep{Command: "echo 2"}))
		step3 := Step{ID: "step3", Type: "package"}
		require.NoError(t, step3.SetConfig(PackageStep{Packages: []string{"git"}}))
		m := StepMap([]Step{step1, step2, step3})
		require.Len(t, m, 3)
		require.Contains(t, m, "step1")
		require.Contains(t, m, "step2")
		require.Contains(t, m, "step3")
		require.Equal(t, "command", m["step1"].Type)
		require.Equal(t, "command", m["step2"].Type)
		require.Equal(t, "package", m["step3"].Type)
	})

	t.Run("later steps override earlier with same ID", func(t *testing.T) {
		t.Parallel()
		first := Step{ID: "duplicate", Type: "command"}
		require.NoError(t, first.SetConfig(CommandStep{Command: "first"}))
		second := Step{ID: "duplicate", Type: "command"}
		require.NoError(t, second.SetConfig(CommandStep{Command: "second"}))
		m := StepMap([]Step{first, second})
		require.Len(t, m, 1)
		var cfg CommandStep
		step := m["duplicate"]
		require.NoError(t, (&step).DecodeConfig(&cfg))
		require.Equal(t, "second", cfg.Command)
	})

	t.Run("preserves step details in map", func(t *testing.T) {
		t.Parallel()
		base := Step{
			ID:        "complex_step",
			Name:      "Complex Step",
			Type:      "command",
			DependsOn: []string{"dep1", "dep2"},
			Enabled:   false,
		}
		require.NoError(t, base.SetConfig(CommandStep{Command: "complex command"}))
		m := StepMap([]Step{base})
		step := m["complex_step"]
		require.Equal(t, "Complex Step", step.Name)
		require.Equal(t, []string{"dep1", "dep2"}, step.DependsOn)
		require.False(t, step.Enabled)
		var cfg CommandStep
		require.NoError(t, (&step).DecodeConfig(&cfg))
		require.Equal(t, "complex command", cfg.Command)
	})
}

func TestHasYAMLKey(t *testing.T) {
	t.Parallel()

	t.Run("returns true when key exists", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
key1: value1
key2: value2
`
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		require.NoError(t, err)

		// The root node is a document node, content[0] is the mapping
		require.True(t, hasYAMLKey(node.Content[0], "key1"))
		require.True(t, hasYAMLKey(node.Content[0], "key2"))
	})

	t.Run("returns false when key does not exist", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
key1: value1
key2: value2
`
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		require.NoError(t, err)

		require.False(t, hasYAMLKey(node.Content[0], "key3"))
		require.False(t, hasYAMLKey(node.Content[0], "missing"))
	})

	t.Run("returns false for nil node", func(t *testing.T) {
		t.Parallel()
		require.False(t, hasYAMLKey(nil, "anykey"))
	})

	t.Run("returns false for non-mapping node", func(t *testing.T) {
		t.Parallel()
		yamlStr := `["item1", "item2"]`
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		require.NoError(t, err)

		// The root is a sequence node, not a mapping
		require.False(t, hasYAMLKey(node.Content[0], "anykey"))
	})

	t.Run("handles case-insensitive key matching", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
MyKey: value
`
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		require.NoError(t, err)

		require.True(t, hasYAMLKey(node.Content[0], "mykey"))
		require.True(t, hasYAMLKey(node.Content[0], "MyKey"))
		require.True(t, hasYAMLKey(node.Content[0], "MYKEY"))
	})

	t.Run("works with empty mapping", func(t *testing.T) {
		t.Parallel()
		yamlStr := `{}`
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		require.NoError(t, err)

		require.False(t, hasYAMLKey(node.Content[0], "anykey"))
	})

	t.Run("works with nested structures", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
outer:
  inner: value
top_level: data
`
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		require.NoError(t, err)

		// Check top-level keys
		require.True(t, hasYAMLKey(node.Content[0], "outer"))
		require.True(t, hasYAMLKey(node.Content[0], "top_level"))
		// Nested keys won't be found at top level
		require.False(t, hasYAMLKey(node.Content[0], "inner"))
	})

	t.Run("handles keys with underscores and dashes", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
snake_case_key: value1
kebab-case-key: value2
`
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		require.NoError(t, err)

		require.True(t, hasYAMLKey(node.Content[0], "snake_case_key"))
		require.True(t, hasYAMLKey(node.Content[0], "kebab-case-key"))
	})

	t.Run("handles numeric values", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
count: 42
enabled: true
`
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		require.NoError(t, err)

		require.True(t, hasYAMLKey(node.Content[0], "count"))
		require.True(t, hasYAMLKey(node.Content[0], "enabled"))
		require.False(t, hasYAMLKey(node.Content[0], "missing"))
	})
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()

	t.Run("config with minimal fields", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
version: "1.0"
name: "Minimal"
steps:
  - id: step1
    type: command
    command: "echo test"
`
		var cfg Config
		err := yaml.Unmarshal([]byte(yamlStr), &cfg)
		require.NoError(t, err)
		require.Equal(t, "1.0", cfg.Version)
		require.Equal(t, "Minimal", cfg.Name)
		require.Empty(t, cfg.Description)
		require.Len(t, cfg.Steps, 1)
	})
}

func TestValidateTemplateConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("valid template configuration", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source:      "/path/to/template.tmpl",
			Destination: "/path/to/output.txt",
			Vars:        map[string]string{"VAR1": "value1", "VAR2": "value2"},
		})
		require.NoError(t, err)
	})

	t.Run("error when source is empty", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Destination: "/path/to/output.txt",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "template source is required")
	})

	t.Run("error when source is whitespace", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source:      "   ",
			Destination: "/path/to/output.txt",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "template source is required")
	})

	t.Run("error when destination is empty", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source: "/path/to/template.tmpl",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "template destination is required")
	})

	t.Run("error when destination is whitespace", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source:      "/path/to/template.tmpl",
			Destination: "   ",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "template destination is required")
	})

	t.Run("error when source equals destination", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source:      "/path/to/file.txt",
			Destination: "/path/to/file.txt",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "template destination must differ from source")
	})

	t.Run("error when source equals destination with whitespace", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source:      "/path/to/file.txt",
			Destination: "  /path/to/file.txt  ",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "template destination must differ from source")
	})

	t.Run("error when variable name is invalid - starts with number", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source:      "/path/to/template.tmpl",
			Destination: "/path/to/output.txt",
			Vars:        map[string]string{"123VAR": "value1"},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "template variable \"123VAR\" is invalid")
	})

	t.Run("error when variable name is invalid - contains special chars", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source:      "/path/to/template.tmpl",
			Destination: "/path/to/output.txt",
			Vars:        map[string]string{"VAR!": "value1"},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "template variable \"VAR!\" is invalid")
	})

	t.Run("valid variable names", func(t *testing.T) {
		t.Parallel()
		err := validateTemplateConfiguration("test-template", TemplateStep{
			Source:      "/path/to/template.tmpl",
			Destination: "/path/to/output.txt",
			Vars:        map[string]string{"VAR_1": "value1", "VAR_TWO": "value2"},
		})
		require.NoError(t, err)
	})
}

func TestTemplateStepUnmarshalYAML(t *testing.T) {
	t.Parallel()

	t.Run("defaults env to true when not specified", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /path/to/template.tmpl
destination: /path/to/output.txt
vars:
  VAR1: value1
`
		var template TemplateStep
		err := yaml.Unmarshal([]byte(yamlStr), &template)
		require.NoError(t, err)
		require.True(t, template.Env)
		require.Equal(t, map[string]string{"VAR1": "value1"}, template.Vars)
	})

	t.Run("respects env=false when specified", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /path/to/template.tmpl
destination: /path/to/output.txt
env: false
`
		var template TemplateStep
		err := yaml.Unmarshal([]byte(yamlStr), &template)
		require.NoError(t, err)
		require.False(t, template.Env)
	})

	t.Run("respects env=true when specified", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /path/to/template.tmpl
destination: /path/to/output.txt
env: true
`
		var template TemplateStep
		err := yaml.Unmarshal([]byte(yamlStr), &template)
		require.NoError(t, err)
		require.True(t, template.Env)
	})

	t.Run("initializes vars map when not provided", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /path/to/template.tmpl
destination: /path/to/output.txt
`
		var template TemplateStep
		err := yaml.Unmarshal([]byte(yamlStr), &template)
		require.NoError(t, err)
		require.NotNil(t, template.Vars)
		require.Empty(t, template.Vars)
	})

	t.Run("preserves existing vars when provided", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /path/to/template.tmpl
destination: /path/to/output.txt
vars:
  VAR1: value1
  VAR2: value2
`
		var template TemplateStep
		err := yaml.Unmarshal([]byte(yamlStr), &template)
		require.NoError(t, err)
		require.Equal(t, map[string]string{"VAR1": "value1", "VAR2": "value2"}, template.Vars)
	})

	t.Run("handles full template configuration", func(t *testing.T) {
		t.Parallel()
		yamlStr := `
source: /path/to/template.tmpl
destination: /path/to/output.txt
env: false
allow_missing: true
vars:
  NAME: test
  VERSION: 1.0
mode: 0644
`
		var template TemplateStep
		err := yaml.Unmarshal([]byte(yamlStr), &template)
		require.NoError(t, err)
		require.Equal(t, "/path/to/template.tmpl", template.Source)
		require.Equal(t, "/path/to/output.txt", template.Destination)
		require.False(t, template.Env)
		require.True(t, template.AllowMissing)
		require.Equal(t, map[string]string{"NAME": "test", "VERSION": "1.0"}, template.Vars)
		require.NotNil(t, template.Mode)
		require.Equal(t, uint32(0o644), *template.Mode)
	})
}

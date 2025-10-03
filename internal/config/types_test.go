package config

import (
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
		require.NotNil(t, step.Package)
		require.Equal(t, []string{"git", "curl"}, step.Package.Packages)
		require.Equal(t, "apt", step.Package.Manager)
		require.Nil(t, step.Repo)
		require.Nil(t, step.Symlink)
		require.Nil(t, step.Copy)
		require.Nil(t, step.Command)
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
		require.NotNil(t, step.Repo)
		require.Equal(t, "https://github.com/example/repo.git", step.Repo.URL)
		require.Equal(t, "/tmp/repo", step.Repo.Destination)
		require.Equal(t, "main", step.Repo.Branch)
		require.Equal(t, 1, step.Repo.Depth)
		require.Nil(t, step.Package)
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
		require.NotNil(t, step.Symlink)
		require.Equal(t, "/etc/config", step.Symlink.Source)
		require.Equal(t, "/home/user/.config", step.Symlink.Target)
		require.True(t, step.Symlink.Force)
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
		require.NotNil(t, step.Copy)
		require.Equal(t, "/tmp/src", step.Copy.Source)
		require.Equal(t, "/tmp/dst", step.Copy.Destination)
		require.True(t, step.Copy.Recursive)
		require.True(t, step.Copy.Overwrite)
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
		require.NotNil(t, step.Command)
		require.Equal(t, "echo hello", step.Command.Command)
		require.Equal(t, "which echo", step.Command.Check)
		require.Equal(t, "/bin/bash", step.Command.Shell)
		require.Equal(t, "/tmp", step.Command.WorkDir)
		require.Equal(t, map[string]string{"FOO": "bar", "BAZ": "qux"}, step.Command.Env)
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
		require.Error(t, err)
	})
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
		steps := []Step{}
		m := StepMap(steps)
		require.Empty(t, m)
	})

	t.Run("creates map from single step", func(t *testing.T) {
		t.Parallel()
		steps := []Step{
			{ID: "step1", Type: "command", Command: &CommandStep{Command: "echo test"}},
		}
		m := StepMap(steps)
		require.Len(t, m, 1)
		require.Contains(t, m, "step1")
		require.Equal(t, "command", m["step1"].Type)
	})

	t.Run("creates map from multiple steps", func(t *testing.T) {
		t.Parallel()
		steps := []Step{
			{ID: "step1", Type: "command", Command: &CommandStep{Command: "echo 1"}},
			{ID: "step2", Type: "command", Command: &CommandStep{Command: "echo 2"}},
			{ID: "step3", Type: "package", Package: &PackageStep{Packages: []string{"git"}}},
		}
		m := StepMap(steps)
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
		steps := []Step{
			{ID: "duplicate", Type: "command", Command: &CommandStep{Command: "first"}},
			{ID: "duplicate", Type: "command", Command: &CommandStep{Command: "second"}},
		}
		m := StepMap(steps)
		require.Len(t, m, 1)
		require.Equal(t, "second", m["duplicate"].Command.Command)
	})

	t.Run("preserves step details in map", func(t *testing.T) {
		t.Parallel()
		steps := []Step{
			{
				ID:        "complex_step",
				Name:      "Complex Step",
				Type:      "command",
				DependsOn: []string{"dep1", "dep2"},
				Enabled:   false,
				Command:   &CommandStep{Command: "complex command"},
			},
		}
		m := StepMap(steps)
		step := m["complex_step"]
		require.Equal(t, "Complex Step", step.Name)
		require.Equal(t, []string{"dep1", "dep2"}, step.DependsOn)
		require.False(t, step.Enabled)
		require.Equal(t, "complex command", step.Command.Command)
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

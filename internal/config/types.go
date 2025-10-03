package config

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the full Streamy configuration document.
type Config struct {
	Version     string       `yaml:"version" validate:"required,semver"`
	Name        string       `yaml:"name" validate:"required,min=1,max=100"`
	Description string       `yaml:"description,omitempty"`
	Settings    Settings     `yaml:"settings,omitempty"`
	Steps       []Step       `yaml:"steps" validate:"required,min=1,dive"`
	Validations []Validation `yaml:"validations,omitempty" validate:"omitempty,dive"`
}

// Settings holds global execution parameters.
type Settings struct {
	Parallel        int  `yaml:"parallel,omitempty" validate:"omitempty,min=1,max=32"`
	Timeout         int  `yaml:"timeout,omitempty" validate:"omitempty,min=1,max=360000"`
	ContinueOnError bool `yaml:"continue_on_error,omitempty"`
	DryRun          bool `yaml:"dry_run,omitempty"`
	Verbose         bool `yaml:"verbose,omitempty"`
}

// Step describes an individual unit of work in the DAG.
type Step struct {
	ID        string   `yaml:"id" validate:"required,step_id"`
	Name      string   `yaml:"name,omitempty"`
	Type      string   `yaml:"type" validate:"required,oneof=package repo symlink copy command"`
	DependsOn []string `yaml:"depends_on,omitempty"`
	Enabled   bool     `yaml:"enabled,omitempty"`

	Package *PackageStep `yaml:",inline,omitempty"`
	Repo    *RepoStep    `yaml:",inline,omitempty"`
	Symlink *SymlinkStep `yaml:",inline,omitempty"`
	Copy    *CopyStep    `yaml:",inline,omitempty"`
	Command *CommandStep `yaml:",inline,omitempty"`
}

// UnmarshalYAML customises step decoding to populate type-specific structures without conflicts.
func (s *Step) UnmarshalYAML(value *yaml.Node) error {
	type baseStep struct {
		ID        string   `yaml:"id"`
		Name      string   `yaml:"name"`
		Type      string   `yaml:"type"`
		DependsOn []string `yaml:"depends_on"`
		Enabled   *bool    `yaml:"enabled"`
	}

	var base baseStep
	if err := value.Decode(&base); err != nil {
		return err
	}

	s.ID = base.ID
	s.Name = base.Name
	s.Type = base.Type
	s.DependsOn = append([]string(nil), base.DependsOn...)
	if base.Enabled != nil {
		s.Enabled = *base.Enabled
	} else {
		s.Enabled = true
	}

	s.Package = nil
	s.Repo = nil
	s.Symlink = nil
	s.Copy = nil
	s.Command = nil

	switch base.Type {
	case "package":
		var pkg PackageStep
		if err := value.Decode(&pkg); err != nil {
			return err
		}
		s.Package = &pkg
	case "repo":
		var repo RepoStep
		if err := value.Decode(&repo); err != nil {
			return err
		}
		s.Repo = &repo
	case "symlink":
		var link SymlinkStep
		if err := value.Decode(&link); err != nil {
			return err
		}
		s.Symlink = &link
	case "copy":
		var cp CopyStep
		if err := value.Decode(&cp); err != nil {
			return err
		}
		s.Copy = &cp
	case "command":
		var cmd CommandStep
		if err := value.Decode(&cmd); err != nil {
			return err
		}
		s.Command = &cmd
	}

	return nil
}

// UnmarshalYAML applies defaults for copy steps.
func (c *CopyStep) UnmarshalYAML(value *yaml.Node) error {
	type rawCopy CopyStep
	var temp rawCopy
	if err := value.Decode(&temp); err != nil {
		return err
	}
	*c = CopyStep(temp)
	c.PreserveModeSet = hasYAMLKey(value, "preserve_mode")
	if !c.PreserveModeSet {
		c.PreserveMode = true
	}
	return nil
}

// PackageStep installs one or more system packages.
type PackageStep struct {
	Packages []string `yaml:"packages" validate:"required,min=1,dive,min=1,max=100"`
	Manager  string   `yaml:"manager,omitempty"`
	Update   bool     `yaml:"update,omitempty"`
}

// RepoStep clones a git repository.
type RepoStep struct {
	URL         string `yaml:"url" validate:"required,url"`
	Destination string `yaml:"destination" validate:"required"`
	Branch      string `yaml:"branch,omitempty"`
	Depth       int    `yaml:"depth,omitempty" validate:"omitempty,min=0"`
}

// SymlinkStep creates a symbolic link.
type SymlinkStep struct {
	Source string `yaml:"source" validate:"required"`
	Target string `yaml:"target" validate:"required,nefield=Source"`
	Force  bool   `yaml:"force,omitempty"`
}

// CopyStep copies files or directories.
type CopyStep struct {
	Source          string `yaml:"source" validate:"required"`
	Destination     string `yaml:"destination" validate:"required,nefield=Source"`
	Overwrite       bool   `yaml:"overwrite,omitempty"`
	Recursive       bool   `yaml:"recursive,omitempty"`
	PreserveMode    bool   `yaml:"preserve_mode,omitempty"`
	PreserveModeSet bool   `yaml:"-"`
}

// CommandStep executes an arbitrary shell command.
type CommandStep struct {
	Command string            `yaml:"command" validate:"required,min=1"`
	Check   string            `yaml:"check,omitempty"`
	Shell   string            `yaml:"shell,omitempty"`
	WorkDir string            `yaml:"workdir,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// Validation represents a post-execution validation.
type Validation struct {
	Type string `yaml:"type" validate:"required,oneof=command_exists file_exists path_contains"`

	CommandExists *CommandExistsValidation `yaml:",inline,omitempty"`
	FileExists    *FileExistsValidation    `yaml:",inline,omitempty"`
	PathContains  *PathContainsValidation  `yaml:",inline,omitempty"`
}

// CommandExistsValidation ensures a command exists on PATH.
type CommandExistsValidation struct {
	Command string `yaml:"command" validate:"required"`
}

// FileExistsValidation ensures a file or directory exists.
type FileExistsValidation struct {
	Path string `yaml:"path" validate:"required"`
}

// PathContainsValidation ensures a file contains specific text.
type PathContainsValidation struct {
	File string `yaml:"file" validate:"required"`
	Text string `yaml:"text" validate:"required"`
}

// StepMap builds a lookup table for steps by ID.
func StepMap(steps []Step) map[string]Step {
	out := make(map[string]Step, len(steps))
	for _, step := range steps {
		out[step.ID] = step
	}
	return out
}

func hasYAMLKey(node *yaml.Node, key string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(node.Content); i += 2 {
		k := node.Content[i]
		if strings.EqualFold(k.Value, key) {
			return true
		}
	}
	return false
}

package config

import (
	"fmt"
	"regexp"
	"strings"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
	"gopkg.in/yaml.v3"
)

var templateVarNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

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
	ID            string   `yaml:"id" validate:"required,step_id"`
	Name          string   `yaml:"name,omitempty"`
	Type          string   `yaml:"type" validate:"required,oneof=package repo symlink copy command template line_in_file"`
	DependsOn     []string `yaml:"depends_on,omitempty"`
	Enabled       bool     `yaml:"enabled,omitempty"`
	VerifyTimeout int      `yaml:"verify_timeout,omitempty" validate:"omitempty,min=1,max=600"`

	rawConfig map[string]any
}

// UnmarshalYAML customises step decoding to populate type-specific structures without conflicts.
func (s *Step) UnmarshalYAML(value *yaml.Node) error {
	type baseStep struct {
		ID            string   `yaml:"id"`
		Name          string   `yaml:"name"`
		Type          string   `yaml:"type"`
		DependsOn     []string `yaml:"depends_on"`
		Enabled       *bool    `yaml:"enabled"`
		VerifyTimeout *int     `yaml:"verify_timeout"`
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
	if base.VerifyTimeout != nil {
		s.VerifyTimeout = *base.VerifyTimeout
	} else {
		s.VerifyTimeout = 0
	}

	s.rawConfig = extractRawConfig(value)
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

func validateTemplateConfiguration(stepID string, cfg TemplateStep) error {
	if strings.TrimSpace(cfg.Source) == "" {
		return streamyerrors.NewValidationError(stepID, "template source is required", nil)
	}

	if strings.TrimSpace(cfg.Destination) == "" {
		return streamyerrors.NewValidationError(stepID, "template destination is required", nil)
	}

	if strings.TrimSpace(cfg.Source) == strings.TrimSpace(cfg.Destination) {
		return streamyerrors.NewValidationError(stepID, "template destination must differ from source", nil)
	}

	for name := range cfg.Vars {
		if !templateVarNamePattern.MatchString(name) {
			return streamyerrors.NewValidationError(stepID, fmt.Sprintf("template variable %q is invalid; must match %s", name, templateVarNamePattern.String()), nil)
		}
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
	URL         string `yaml:"url" validate:"required,git_url"`
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

// TemplateStep renders a destination file from a template source with variable substitution.
type TemplateStep struct {
	Source       string            `yaml:"source" validate:"required"`
	Destination  string            `yaml:"destination" validate:"required,nefield=Source"`
	Vars         map[string]string `yaml:"vars,omitempty"`
	Env          bool              `yaml:"env,omitempty"`
	AllowMissing bool              `yaml:"allow_missing,omitempty"`
	Mode         *uint32           `yaml:"mode,omitempty" validate:"omitempty,min=0,max=511"`
}

// LineInFileStep manages individual lines in text files.
type LineInFileStep struct {
	File              string `yaml:"file" validate:"required"`
	Line              string `yaml:"line" validate:"required"`
	State             string `yaml:"state,omitempty"`
	Match             string `yaml:"match,omitempty"`
	OnMultipleMatches string `yaml:"on_multiple_matches,omitempty"`
	Backup            bool   `yaml:"backup,omitempty"`
	BackupDir         string `yaml:"backup_dir,omitempty"`
	Encoding          string `yaml:"encoding,omitempty"`
}

// UnmarshalYAML applies defaults for template steps and ensures maps are initialised.
func (t *TemplateStep) UnmarshalYAML(value *yaml.Node) error {
	type rawTemplate TemplateStep
	var temp rawTemplate
	if err := value.Decode(&temp); err != nil {
		return err
	}

	if temp.Vars == nil {
		temp.Vars = make(map[string]string)
	}

	if !hasYAMLKey(value, "env") {
		temp.Env = true
	}

	*t = TemplateStep(temp)
	return nil
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

// RawConfig returns a shallow copy of the plugin-specific configuration block.
func (s *Step) RawConfig() map[string]any {
	if s == nil || s.rawConfig == nil {
		return map[string]any{}
	}

	raw := make(map[string]any, len(s.rawConfig))
	for k, v := range s.rawConfig {
		raw[k] = v
	}
	return raw
}

// SetConfig replaces the step's plugin-specific configuration payload.
func (s *Step) SetConfig(cfg any) error {
	if s == nil {
		return fmt.Errorf("step is nil")
	}
	if cfg == nil {
		s.rawConfig = map[string]any{}
		return nil
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.rawConfig = raw
	return nil
}

// DecodeConfig unmarshals the plugin-specific configuration into dst without touching legacy fields.
func (s *Step) DecodeConfig(dst any) error {
	if dst == nil {
		return fmt.Errorf("destination cannot be nil")
	}
	if s == nil {
		return fmt.Errorf("step is nil")
	}
	if len(s.rawConfig) == 0 {
		return nil
	}

	data, err := yaml.Marshal(s.rawConfig)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, dst)
}

func extractRawConfig(node *yaml.Node) map[string]any {
	if node == nil {
		return map[string]any{}
	}

	var raw map[string]any
	if err := node.Decode(&raw); err != nil || raw == nil {
		return map[string]any{}
	}

	// Base keys to remove case-insensitively
	baseKeys := map[string]bool{
		"id":             true,
		"name":           true,
		"type":           true,
		"depends_on":     true,
		"enabled":        true,
		"verify_timeout": true,
	}

	// Remove base keys case-insensitively
	for key := range raw {
		if baseKeys[strings.ToLower(key)] {
			delete(raw, key)
		}
	}

	return raw
}

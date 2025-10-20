// Package config provides YAML parsing and validation utilities.
package config

import (
	"fmt"
	"strings"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Version     string             `yaml:"version"`
	Name        string             `yaml:"name"`
	Description string             `yaml:"description,omitempty"`
	Settings    settingsConfig     `yaml:"settings,omitempty"`
	Steps       []stepConfig       `yaml:"steps"`
	Validations []validationConfig `yaml:"validations,omitempty"`
}

func (c *fileConfig) toPipeline() (*domain.Pipeline, error) {
	pipeline := &domain.Pipeline{
		Version:     c.Version,
		Name:        c.Name,
		Description: c.Description,
		Settings: domain.Settings{
			Parallel:        c.Settings.Parallel,
			Timeout:         c.Settings.Timeout,
			ContinueOnError: c.Settings.ContinueOnError,
			DryRun:          c.Settings.DryRun,
			Verbose:         c.Settings.Verbose,
		},
	}

	if len(c.Steps) > 0 {
		pipeline.Steps = make([]domain.Step, 0, len(c.Steps))
		for _, step := range c.Steps {
			domainStep, err := step.toDomain()
			if err != nil {
				return nil, err
			}

			pipeline.Steps = append(pipeline.Steps, domainStep)
		}
	}

	if len(c.Validations) > 0 {
		pipeline.Validations = make([]domain.Validation, 0, len(c.Validations))
		for _, validation := range c.Validations {
			domainValidation, err := validation.toDomain()
			if err != nil {
				return nil, err
			}

			pipeline.Validations = append(pipeline.Validations, domainValidation)
		}
	}

	if err := pipeline.Validate(); err != nil {
		return nil, fmt.Errorf("validate pipeline %q: %w", pipeline.Name, err)
	}

	return pipeline, nil
}

type settingsConfig struct {
	Parallel        int  `yaml:"parallel,omitempty"`
	Timeout         int  `yaml:"timeout,omitempty"`
	ContinueOnError bool `yaml:"continue_on_error,omitempty"`
	DryRun          bool `yaml:"dry_run,omitempty"`
	Verbose         bool `yaml:"verbose,omitempty"`
}

type stepConfig struct {
	ID            string   `yaml:"id"`
	Name          string   `yaml:"name,omitempty"`
	Type          string   `yaml:"type"`
	DependsOn     []string `yaml:"depends_on,omitempty"`
	Enabled       bool     `yaml:"enabled"`
	VerifyTimeout int      `yaml:"verify_timeout,omitempty"`
	rawConfig     map[string]any
}

func (s *stepConfig) UnmarshalYAML(value *yaml.Node) error {
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
		return fmt.Errorf("decode step config: %w", err)
	}

	s.ID = strings.TrimSpace(base.ID)
	s.Name = strings.TrimSpace(base.Name)
	s.Type = strings.TrimSpace(base.Type)

	s.DependsOn = append([]string(nil), base.DependsOn...)
	if base.Enabled != nil {
		s.Enabled = *base.Enabled
	} else {
		s.Enabled = true
	}

	if base.VerifyTimeout != nil {
		s.VerifyTimeout = *base.VerifyTimeout
	}

	s.rawConfig = extractRawConfig(value)

	return nil
}

func (s stepConfig) toDomain() (domain.Step, error) {
	step := domain.Step{
		ID:            s.ID,
		Name:          s.Name,
		Type:          domain.StepType(s.Type),
		DependsOn:     append([]string(nil), s.DependsOn...),
		Enabled:       s.Enabled,
		VerifyTimeout: s.VerifyTimeout,
		Config:        cloneMap(s.rawConfig),
	}
	if err := step.Validate(); err != nil {
		return domain.Step{}, fmt.Errorf("validate step %q: %w", step.ID, err)
	}

	return step, nil
}

type validationConfig struct {
	Type   string
	Config map[string]any
}

func (v *validationConfig) UnmarshalYAML(value *yaml.Node) error {
	type rawValidation struct {
		Type string `yaml:"type"`
	}

	var raw rawValidation
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("decode validation config: %w", err)
	}

	v.Type = strings.TrimSpace(raw.Type)
	v.Config = extractValidationConfig(value)

	return nil
}

func (v validationConfig) toDomain() (domain.Validation, error) {
	validation := domain.Validation{
		Type:   domain.ValidationType(v.Type),
		Config: cloneMap(v.Config),
	}
	if err := validation.Validate(); err != nil {
		return domain.Validation{}, fmt.Errorf("validate validation %q: %w", validation.Type, err)
	}

	return validation, nil
}

func extractRawConfig(node *yaml.Node) map[string]any {
	if node == nil {
		return map[string]any{}
	}

	var raw map[string]any
	if err := node.Decode(&raw); err != nil || raw == nil {
		return map[string]any{}
	}

	baseKeys := map[string]bool{
		"id":             true,
		"name":           true,
		"type":           true,
		"depends_on":     true,
		"enabled":        true,
		"verify_timeout": true,
	}

	for key := range raw {
		if baseKeys[strings.ToLower(key)] {
			delete(raw, key)
		}
	}

	return raw
}

func extractValidationConfig(node *yaml.Node) map[string]any {
	if node == nil {
		return map[string]any{}
	}

	var raw map[string]any
	if err := node.Decode(&raw); err != nil || raw == nil {
		return map[string]any{}
	}

	for key := range raw {
		if strings.EqualFold(key, "type") {
			delete(raw, key)
		}
	}

	return raw
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}

	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}

	return out
}

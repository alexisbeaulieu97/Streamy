package config

import (
	"errors"
	"testing"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

func TestStepConfigUnmarshalYAMLDefaults(t *testing.T) {
	yamlContent := `
 id: " setup "
 name: " Setup Step "
 type: " command "
 depends_on:
   - build
 verify_timeout: 15
 enabled: false
 command: echo hi
`

	var cfg stepConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
		t.Fatalf("unmarshal step config: %v", err)
	}

	if cfg.ID != "setup" {
		t.Fatalf("expected trimmed id, got %q", cfg.ID)
	}

	if cfg.Name != "Setup Step" {
		t.Fatalf("expected trimmed name, got %q", cfg.Name)
	}

	if cfg.Type != "command" {
		t.Fatalf("expected trimmed type, got %q", cfg.Type)
	}

	if cfg.Enabled {
		t.Fatal("expected explicit enabled false to be preserved")
	}

	if cfg.VerifyTimeout != 15 {
		t.Fatalf("expected verify timeout 15, got %d", cfg.VerifyTimeout)
	}

	if len(cfg.DependsOn) != 1 || cfg.DependsOn[0] != "build" {
		t.Fatalf("unexpected depends_on: %+v", cfg.DependsOn)
	}

	if cfg.rawConfig == nil {
		t.Fatal("expected rawConfig to be populated")
	}

	if cfg.rawConfig["command"] != "echo hi" {
		t.Fatalf("expected raw config to retain custom fields, got %v", cfg.rawConfig)
	}
}

func TestValidationConfigUnmarshalYAMLTrimsType(t *testing.T) {
	yamlContent := ` type: " command_exists "
 command: go
`

	var val validationConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &val); err != nil {
		t.Fatalf("unmarshal validation config: %v", err)
	}

	if val.Type != "command_exists" {
		t.Fatalf("expected trimmed type, got %q", val.Type)
	}

	if val.Config["command"] != "go" {
		t.Fatalf("expected command key in config, got %+v", val.Config)
	}

	if _, ok := val.Config["type"]; ok {
		t.Fatalf("expected type field removed from config: %+v", val.Config)
	}
}

func TestFileConfigToPipelineSuccess(t *testing.T) {
	cfg := fileConfig{
		ID:      "demo",
		Version: "1.0",
		Name:    "demo",
		Dependencies: []string{
			"db@1.0",
			"network@2.0",
		},
		Settings: settingsConfig{
			Parallel:        8,
			Timeout:         120,
			ContinueOnError: true,
			DryRun:          true,
			Verbose:         true,
		},
		Steps: []stepConfig{{
			ID:        "setup",
			Name:      "Setup",
			Type:      "command",
			Enabled:   true,
			rawConfig: map[string]any{"command": "echo hi"},
		}},
		Validations: []validationConfig{{
			Type:   "command_exists",
			Config: map[string]any{"command": "go"},
		}},
	}

	pipeline, err := cfg.toPipeline()
	if err != nil {
		t.Fatalf("expected pipeline conversion success, got %v", err)
	}

	if pipeline.Name != "demo" || pipeline.Version != "1.0" {
		t.Fatalf("unexpected pipeline metadata: %+v", pipeline)
	}

	eff := pipeline.EffectiveSettings()
	if eff.Parallel != 8 || eff.Timeout != 120 || !eff.DryRun || !eff.Verbose {
		t.Fatalf("unexpected effective settings: %+v", eff)
	}

	if len(pipeline.Steps) != 1 {
		t.Fatalf("expected single step, got %d", len(pipeline.Steps))
	}

	step := pipeline.Steps[0]
	if step.ID != "setup" || step.Config["command"] != "echo hi" {
		t.Fatalf("unexpected step data: %+v", step)
	}

	if len(pipeline.Validations) != 1 || pipeline.Validations[0].Type != domain.ValidationCommandExists {
		t.Fatalf("unexpected validations: %+v", pipeline.Validations)
	}
}

func TestFileConfigToPipelineValidationFailure(t *testing.T) {
	cfg := fileConfig{
		ID:      "demo",
		Version: "1.0",
		Name:    "demo",
		Steps: []stepConfig{{
			ID:        "",
			Type:      "command",
			Enabled:   true,
			rawConfig: map[string]any{"command": "echo"},
		}},
	}

	_, err := cfg.toPipeline()
	if err == nil {
		t.Fatal("expected validation error for missing step id")
	}
}

func TestExtractRawConfigFiltersBaseKeys(t *testing.T) {
	yamlContent := `
 id: build
 type: command
 enabled: true
 verify_timeout: 10
 custom: value
`

	var node ast.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &node); err != nil {
		t.Fatalf("unmarshal node: %v", err)
	}

	raw := extractRawConfig(node)
	if len(raw) != 1 || raw["custom"] != "value" {
		t.Fatalf("expected only custom field, got %+v", raw)
	}
}

func TestExtractRawConfigNilNode(t *testing.T) {
	raw := extractRawConfig(nil)
	if len(raw) != 0 {
		t.Fatalf("expected empty map, got %+v", raw)
	}
}

func TestExtractValidationConfigRemovesType(t *testing.T) {
	yamlContent := `
 type: file_exists
 path: /tmp/demo
`

	var node ast.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &node); err != nil {
		t.Fatalf("unmarshal node: %v", err)
	}

	raw := extractValidationConfig(node)
	if len(raw) != 1 || raw["path"] != "/tmp/demo" {
		t.Fatalf("expected only path key, got %+v", raw)
	}

	if _, ok := raw["type"]; ok {
		t.Fatal("type key should be removed from validation config")
	}
}

func TestFileConfigToPipelineRequiresIDAndVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  fileConfig
	}{
		{
			name: "missing id",
			cfg: fileConfig{
				Version: "1.0",
				Name:    "demo",
				Steps: []stepConfig{{
					ID:        "setup",
					Type:      "command",
					Enabled:   true,
					rawConfig: map[string]any{"command": "echo"},
				}},
			},
		},
		{
			name: "missing version",
			cfg: fileConfig{
				ID:   "demo",
				Name: "demo",
				Steps: []stepConfig{{
					ID:        "setup",
					Type:      "command",
					Enabled:   true,
					rawConfig: map[string]any{"command": "echo"},
				}},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.cfg.toPipeline()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			var domainErr *domain.DomainError
			if !errors.As(err, &domainErr) {
				t.Fatalf("expected domain error, got %T", err)
			}

			if domainErr.Code != domain.ErrCodeMissing && domainErr.Code != domain.ErrCodeValidation {
				t.Fatalf("unexpected domain error code: %s", domainErr.Code)
			}
		})
	}
}

func TestFileConfigToPipelineRejectsNonCanonicalDependencies(t *testing.T) {
	cfg := fileConfig{
		ID:      "demo",
		Version: "1.0",
		Name:    "demo",
		Dependencies: []string{
			"db", // missing version portion
		},
		Steps: []stepConfig{{
			ID:        "setup",
			Type:      "command",
			Enabled:   true,
			rawConfig: map[string]any{"command": "echo"},
		}},
	}

	_, err := cfg.toPipeline()
	if err == nil {
		t.Fatal("expected validation error for non-canonical dependency, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected domain error, got %T", err)
	}

	if domainErr.Code != domain.ErrCodeValidation {
		t.Fatalf("unexpected error code: %s", domainErr.Code)
	}
}

func TestCloneMapProducesCopy(t *testing.T) {
	src := map[string]any{"a": "b"}
	clone := cloneMap(src)

	clone["a"] = "changed"

	if src["a"] != "b" {
		t.Fatalf("expected original map untouched, got %v", src["a"])
	}
}

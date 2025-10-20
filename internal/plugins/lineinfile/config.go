// Package lineinfileplugin parses configuration for the line-in-file plugin.
package lineinfileplugin

import (
	"fmt"
	"regexp"
	"strings"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

const (
	statePresent             = "present"
	stateAbsent              = "absent"
	onMultipleFirst          = "first"
	onMultipleAll            = "all"
	onMultipleError          = "error"
	onMultiplePrompt         = "prompt"
	defaultOnMultipleMatches = onMultiplePrompt
)

var allowedStates = map[string]struct{}{
	statePresent: {},
	stateAbsent:  {},
}

var allowedOnMultiple = map[string]struct{}{
	onMultipleFirst:  {},
	onMultipleAll:    {},
	onMultipleError:  {},
	onMultiplePrompt: {},
}

// LineInFileConfig represents the validated configuration used by the plugin.
type LineInFileConfig struct {
	File              string
	Line              string
	State             string
	Match             string
	OnMultipleMatches string
	Backup            bool
	BackupDir         string
	Encoding          string

	pattern *regexp.Regexp
}

// newConfigFromStep extracts and validates the line_in_file configuration.
func newConfigFromDomainStep(step domainpipeline.Step) (*LineInFileConfig, error) {
	config, err := extractLineInFileConfig(step)
	if err != nil {
		return nil, err
	}

	applyLineInFileDefaults(config)

	if err := validateLineInFileConfig(config); err != nil {
		return nil, err
	}

	if err := compileMatchPattern(config); err != nil {
		return nil, err
	}

	if err := validateEncoding(config); err != nil {
		return nil, err
	}

	return config, nil
}

func extractLineInFileConfig(step domainpipeline.Step) (*LineInFileConfig, error) {
	if step.Config == nil {
		//nolint:wrapcheck // returning domain validation error
		return nil, streamyerrors.NewValidationError(step.ID, "lineinfile configuration missing", nil)
	}

	file, ok := getStringValue(step.Config["file"])
	if !ok || strings.TrimSpace(file) == "" {
		//nolint:wrapcheck // returning domain validation error
		return nil, streamyerrors.NewValidationError("file", "file path is required", nil)
	}

	backup, err := getBoolValue(step.Config["backup"])
	if err != nil {
		//nolint:wrapcheck // returning domain validation error
		return nil, streamyerrors.NewValidationError("backup", err.Error(), err)
	}

	line, _ := getStringValue(step.Config["line"])
	state, _ := getStringValue(step.Config["state"])
	match, _ := getStringValue(step.Config["match"])
	onMultiple, _ := getStringValue(step.Config["on_multiple_matches"])
	backupDir, _ := getStringValue(step.Config["backup_dir"])
	encoding, _ := getStringValue(step.Config["encoding"])

	return &LineInFileConfig{
		File:              strings.TrimSpace(file),
		Line:              line,
		State:             strings.TrimSpace(strings.ToLower(state)),
		Match:             match,
		OnMultipleMatches: strings.TrimSpace(strings.ToLower(onMultiple)),
		Backup:            backup,
		BackupDir:         strings.TrimSpace(backupDir),
		Encoding:          strings.TrimSpace(strings.ToLower(encoding)),
	}, nil
}

func applyLineInFileDefaults(cfg *LineInFileConfig) {
	if cfg.State == "" {
		cfg.State = statePresent
	}

	if cfg.OnMultipleMatches == "" {
		cfg.OnMultipleMatches = defaultOnMultipleMatches
	}
}

func validateLineInFileConfig(cfg *LineInFileConfig) error {
	if cfg.State != stateAbsent && strings.TrimSpace(cfg.Line) == "" {
		//nolint:wrapcheck // returning domain validation error
		return streamyerrors.NewValidationError("line", "line is required", nil)
	}

	if _, ok := allowedStates[cfg.State]; !ok {
		//nolint:wrapcheck // returning domain validation error
		return streamyerrors.NewValidationError("state", "must be 'present' or 'absent'", nil)
	}

	if _, ok := allowedOnMultiple[cfg.OnMultipleMatches]; !ok {
		//nolint:wrapcheck // returning domain validation error
		return streamyerrors.NewValidationError("on_multiple_matches", "must be one of: first, all, error, prompt", nil)
	}

	if cfg.State == stateAbsent && strings.TrimSpace(cfg.Match) == "" {
		//nolint:wrapcheck // returning domain validation error
		return streamyerrors.NewValidationError("match", "required when state is absent", nil)
	}

	return nil
}

func compileMatchPattern(cfg *LineInFileConfig) error {
	if strings.TrimSpace(cfg.Match) == "" {
		return nil
	}

	pattern, err := regexp.Compile(cfg.Match)
	if err != nil {
		//nolint:wrapcheck // returning domain validation error
		return streamyerrors.NewValidationError("match", fmt.Sprintf("invalid regex pattern: %v", err), err)
	}

	cfg.pattern = pattern

	return nil
}

func validateEncoding(cfg *LineInFileConfig) error {
	if cfg.Encoding == "" {
		return nil
	}

	if !isSupportedEncoding(cfg.Encoding) {
		//nolint:wrapcheck // returning domain validation error
		return streamyerrors.NewValidationError("encoding", fmt.Sprintf("unsupported encoding: %s", cfg.Encoding), nil)
	}

	return nil
}

func isSupportedEncoding(name string) bool {
	switch strings.ToLower(name) {
	case "", "utf-8", "utf8", "latin-1", "latin1", "iso-8859-1", "windows-1252", "ascii":
		return true
	}

	return false
}

func getStringValue(value interface{}) (string, bool) {
	if value == nil {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func getBoolValue(value interface{}) (bool, error) {
	if value == nil {
		return false, nil
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "", "false", "0", "no":
			return false, nil
		case "true", "1", "yes":
			return true, nil
		default:
			return false, fmt.Errorf("invalid boolean value %q", v)
		}
	default:
		return false, fmt.Errorf("invalid boolean type %T", v)
	}
}

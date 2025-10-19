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
	if step.Config == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "lineinfile configuration missing", nil)
	}

	file, ok := getStringValue(step.Config["file"])
	if !ok || strings.TrimSpace(file) == "" {
		return nil, streamyerrors.NewValidationError("file", "file path is required", nil)
	}

	line, _ := getStringValue(step.Config["line"])
	state, _ := getStringValue(step.Config["state"])
	match, _ := getStringValue(step.Config["match"])
	onMultiple, _ := getStringValue(step.Config["on_multiple_matches"])
	backup, err := getBoolValue(step.Config["backup"])
	if err != nil {
		return nil, streamyerrors.NewValidationError("backup", err.Error(), err)
	}
	backupDir, _ := getStringValue(step.Config["backup_dir"])
	encoding, _ := getStringValue(step.Config["encoding"])

	normalized := &LineInFileConfig{
		File:              strings.TrimSpace(file),
		Line:              line,
		State:             strings.TrimSpace(strings.ToLower(state)),
		Match:             match,
		OnMultipleMatches: strings.TrimSpace(strings.ToLower(onMultiple)),
		Backup:            backup,
		BackupDir:         strings.TrimSpace(backupDir),
		Encoding:          strings.TrimSpace(strings.ToLower(encoding)),
	}

	if normalized.State == "" {
		normalized.State = statePresent
	}
	if normalized.OnMultipleMatches == "" {
		normalized.OnMultipleMatches = defaultOnMultipleMatches
	}

	if normalized.State != stateAbsent && strings.TrimSpace(normalized.Line) == "" {
		return nil, streamyerrors.NewValidationError("line", "line is required", nil)
	}

	if _, ok := allowedStates[normalized.State]; !ok {
		return nil, streamyerrors.NewValidationError("state", "must be 'present' or 'absent'", nil)
	}

	if _, ok := allowedOnMultiple[normalized.OnMultipleMatches]; !ok {
		return nil, streamyerrors.NewValidationError("on_multiple_matches", "must be one of: first, all, error, prompt", nil)
	}

	if normalized.State == stateAbsent && strings.TrimSpace(normalized.Match) == "" {
		return nil, streamyerrors.NewValidationError("match", "required when state is absent", nil)
	}

	if strings.TrimSpace(normalized.Match) != "" {
		pattern, err := regexp.Compile(normalized.Match)
		if err != nil {
			return nil, streamyerrors.NewValidationError("match", fmt.Sprintf("invalid regex pattern: %v", err), err)
		}
		normalized.pattern = pattern
	}

	if normalized.Encoding != "" && !isSupportedEncoding(normalized.Encoding) {
		return nil, streamyerrors.NewValidationError("encoding", fmt.Sprintf("unsupported encoding: %s", normalized.Encoding), nil)
	}

	return normalized, nil
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

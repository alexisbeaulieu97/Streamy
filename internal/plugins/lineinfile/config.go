package lineinfileplugin

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alexisbeaulieu97/streamy/internal/config"
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
func newConfigFromStep(step *config.Step) (*LineInFileConfig, error) {
	if step == nil {
		return nil, streamyerrors.NewValidationError("", "lineinfile configuration missing", nil)
	}

	if len(step.RawConfig()) == 0 {
		return nil, streamyerrors.NewValidationError(step.ID, "lineinfile configuration missing", nil)
	}
	var decoded config.LineInFileStep
	if err := step.DecodeConfig(&decoded); err != nil {
		return nil, streamyerrors.NewValidationError(step.ID, fmt.Sprintf("failed to decode lineinfile config: %v", err), err)
	}
	cfg := &decoded

	normalized := &LineInFileConfig{
		File:              strings.TrimSpace(cfg.File),
		Line:              cfg.Line,
		State:             strings.TrimSpace(strings.ToLower(cfg.State)),
		Match:             cfg.Match,
		OnMultipleMatches: strings.TrimSpace(strings.ToLower(cfg.OnMultipleMatches)),
		Backup:            cfg.Backup,
		BackupDir:         strings.TrimSpace(cfg.BackupDir),
		Encoding:          strings.TrimSpace(strings.ToLower(cfg.Encoding)),
	}

	if normalized.State == "" {
		normalized.State = statePresent
	}
	if normalized.OnMultipleMatches == "" {
		normalized.OnMultipleMatches = defaultOnMultipleMatches
	}

	if normalized.File == "" {
		return nil, streamyerrors.NewValidationError("file", "file path is required", nil)
	}

	if strings.TrimSpace(normalized.Line) == "" && normalized.State != stateAbsent {
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

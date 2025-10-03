package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

var yamlLineRegex = regexp.MustCompile(`line (\d+)`)

// ParseConfig loads a configuration file from disk, validates it, and returns the resulting model.
func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, streamyerrors.NewParseError(path, 0, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, streamyerrors.NewParseError(path, extractLine(err), err)
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func extractLine(err error) int {
	if err == nil {
		return 0
	}

	matches := yamlLineRegex.FindStringSubmatch(err.Error())
	if len(matches) != 2 {
		return 0
	}

	var line int
	_, scanErr := fmt.Sscanf(matches[1], "%d", &line)
	if scanErr != nil {
		return 0
	}

	return line
}

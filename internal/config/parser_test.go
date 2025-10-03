package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	validYAML := `version: "1.0"
name: "Test Config"
description: "Sample config for parser tests"
settings:
  parallel: 4
steps:
  - id: step_one
    type: command
    command: "echo hello"
`

	invalidYAML := `version: [1, 0]
name: "Broken"
steps:
  - id: missing_type
`

	missingRequired := `version: "1.0"
name: "No Steps"
`

	badVersion := `version: "beta"
name: "Bad Version"
steps:
  - id: step
    type: command
    command: "echo"
`

	cases := []struct {
		name      string
		contents  string
		wantError error
		assert    func(t *testing.T, cfg *Config, err error)
	}{
		{
			name:     "valid configuration is parsed",
			contents: validYAML,
			assert: func(t *testing.T, cfg *Config, err error) {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				require.Equal(t, "Test Config", cfg.Name)
				require.Len(t, cfg.Steps, 1)
				require.Equal(t, "step_one", cfg.Steps[0].ID)
			},
		},
		{
			name:      "invalid yaml returns parse error",
			contents:  invalidYAML,
			wantError: &streamyerrors.ParseError{},
			assert: func(t *testing.T, cfg *Config, err error) {
				require.Error(t, err)
				var parseErr *streamyerrors.ParseError
				require.ErrorAs(t, err, &parseErr)
				require.Contains(t, parseErr.Message, "cannot unmarshal")
			},
		},
		{
			name:      "missing required fields returns validation error",
			contents:  missingRequired,
			wantError: &streamyerrors.ValidationError{},
			assert: func(t *testing.T, cfg *Config, err error) {
				require.Error(t, err)
				var validationErr *streamyerrors.ValidationError
				require.ErrorAs(t, err, &validationErr)
				require.Contains(t, validationErr.Message, "steps")
			},
		},
		{
			name:      "schema version must follow major.minor",
			contents:  badVersion,
			wantError: &streamyerrors.ValidationError{},
			assert: func(t *testing.T, cfg *Config, err error) {
				require.Error(t, err)
				var validationErr *streamyerrors.ValidationError
				require.ErrorAs(t, err, &validationErr)
				require.Contains(t, validationErr.Message, "version")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := writeTempConfig(t, tc.contents)
			cfg, err := ParseConfig(path)
			if tc.wantError == nil {
				tc.assert(t, cfg, err)
				return
			}

			tc.assert(t, cfg, err)
			require.Error(t, err)
		})
	}
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type logEntry map[string]any

func TestLoggerInfoWithFields(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	log, err := New(Options{Level: "info", HumanReadable: false, Writer: buf})
	require.NoError(t, err)

	log = log.WithFields(map[string]any{"step": "install_git", "phase": "setup"})
	log.Info("starting execution")

	var entry logEntry
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	require.Equal(t, "starting execution", entry["message"])
	require.Equal(t, "install_git", entry["step"])
	require.Equal(t, "setup", entry["phase"])
	require.Equal(t, "info", entry["level"])
}

func TestLoggerDebugRespectsLevel(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	log, err := New(Options{Level: "info", HumanReadable: false, Writer: buf})
	require.NoError(t, err)

	log.Debug("this should not appear")
	require.Equal(t, "", strings.TrimSpace(buf.String()))
}

func TestLoggerErrorIncludesContext(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	log, err := New(Options{Level: "debug", HumanReadable: false, Writer: buf})
	require.NoError(t, err)

	log = log.WithFields(map[string]any{"step": "clone_repo"})
	log.Error(errors.New("boom"), "failed")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 1)

	var entry logEntry
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &entry))
	require.Equal(t, "failed", entry["message"])
	require.Equal(t, "clone_repo", entry["step"])
	require.Equal(t, "boom", entry["error"])
}

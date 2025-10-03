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

func TestLoggerWarn(t *testing.T) {
	t.Parallel()

	t.Run("writes warning message", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		log, err := New(Options{Level: "warn", HumanReadable: false, Writer: buf})
		require.NoError(t, err)

		log.Warn("something suspicious")

		var entry logEntry
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		require.Equal(t, "something suspicious", entry["message"])
		require.Equal(t, "warn", entry["level"])
	})

	t.Run("respects log level", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		log, err := New(Options{Level: "error", HumanReadable: false, Writer: buf})
		require.NoError(t, err)

		log.Warn("this should not appear")
		require.Equal(t, "", strings.TrimSpace(buf.String()))
	})
}

func TestLoggerNilHandling(t *testing.T) {
	t.Parallel()

	t.Run("nil logger Info does not panic", func(t *testing.T) {
		t.Parallel()
		var log *Logger
		require.NotPanics(t, func() {
			log.Info("test")
		})
	})

	t.Run("nil logger Debug does not panic", func(t *testing.T) {
		t.Parallel()
		var log *Logger
		require.NotPanics(t, func() {
			log.Debug("test")
		})
	})

	t.Run("nil logger Warn does not panic", func(t *testing.T) {
		t.Parallel()
		var log *Logger
		require.NotPanics(t, func() {
			log.Warn("test")
		})
	})

	t.Run("nil logger Error does not panic", func(t *testing.T) {
		t.Parallel()
		var log *Logger
		require.NotPanics(t, func() {
			log.Error(errors.New("test"), "test")
		})
	})

	t.Run("nil logger WithFields returns nil", func(t *testing.T) {
		t.Parallel()
		var log *Logger
		result := log.WithFields(map[string]any{"key": "value"})
		require.Nil(t, result)
	})
}

func TestLoggerErrorWithNilError(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	log, err := New(Options{Level: "error", HumanReadable: false, Writer: buf})
	require.NoError(t, err)

	log.Error(nil, "error without underlying cause")

	var entry logEntry
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	require.Equal(t, "error without underlying cause", entry["message"])
	require.Equal(t, "error", entry["level"])
}

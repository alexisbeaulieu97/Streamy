package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Options describes logger configuration supplied at creation time.
type Options struct {
	Level         string
	HumanReadable bool
	Writer        io.Writer
}

// Logger wraps zerolog to provide a simplified API for the application.
type Logger struct {
	base zerolog.Logger
}

// New creates a configured Logger instance based on Options.
func New(opts Options) (*Logger, error) {
	writer := opts.Writer
	if writer == nil {
		writer = os.Stdout
	}

	level := zerolog.InfoLevel
	if opts.Level != "" {
		parsed, err := zerolog.ParseLevel(strings.ToLower(opts.Level))
		if err != nil {
			return nil, err
		}
		level = parsed
	}

	var output io.Writer = writer
	if opts.HumanReadable {
		console := zerolog.NewConsoleWriter()
		console.Out = writer
		console.TimeFormat = time.RFC3339
		output = console
	}

	logger := zerolog.New(output).Level(level).With().Timestamp().Logger()
	return &Logger{base: logger}, nil
}

// WithFields returns a derived logger that always writes the supplied fields.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	if l == nil {
		return nil
	}

	builder := l.base.With()
	for key, value := range fields {
		builder = builder.Interface(key, value)
	}

	derived := Logger{base: builder.Logger()}
	return &derived
}

// Info writes an informational log entry.
func (l *Logger) Info(msg string) {
	if l == nil {
		return
	}
	l.base.Info().Msg(msg)
}

// Debug writes a debug-level log entry if enabled.
func (l *Logger) Debug(msg string) {
	if l == nil {
		return
	}
	l.base.Debug().Msg(msg)
}

// Warn writes a warning level log entry.
func (l *Logger) Warn(msg string) {
	if l == nil {
		return
	}
	l.base.Warn().Msg(msg)
}

// Error writes an error log entry including the supplied error context.
func (l *Logger) Error(err error, msg string) {
	if l == nil {
		return
	}
	event := l.base.Error()
	if err != nil {
		event = event.Err(err)
	}
	event.Msg(msg)
}

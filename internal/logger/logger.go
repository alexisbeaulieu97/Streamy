package logger

import (
	"context"
	"io"
	"sort"
	"strings"

	cblog "github.com/charmbracelet/log"

	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Options describes logger configuration supplied at creation time.
type Options struct {
	Level         string
	HumanReadable bool
	Writer        io.Writer
	Layer         string
	Component     string
}

// Logger retains the legacy API while delegating to the new charmbracelet/log adapter.
type Logger struct {
	base ports.Logger
}

// New creates a configured Logger instance based on Options.
func New(opts Options) (*Logger, error) {
	layer := opts.Layer
	if layer == "" {
		layer = "legacy"
	}
	component := opts.Component
	if component == "" {
		component = "legacy"
	}

	infraOpts := logginginfra.Options{
		Writer:    opts.Writer,
		Level:     opts.Level,
		Layer:     layer,
		Component: component,
	}

	// When the legacy logger was configured without human readable output it produced JSON.
	// Mirror that behaviour by selecting the JSON formatter explicitly.
	if !opts.HumanReadable {
		infraOpts.Formatter = cblog.JSONFormatter
	}

	logger, err := logginginfra.New(infraOpts)
	if err != nil {
		return nil, err
	}

	return &Logger{base: logger}, nil
}

// WithFields returns a derived logger that always writes the supplied fields.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	if l == nil || l.base == nil || len(fields) == 0 {
		return l
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	args := make([]interface{}, 0, len(fields)*2)
	for _, key := range keys {
		args = append(args, key, fields[key])
	}

	return &Logger{base: l.base.With(args...)}
}

// Info writes an informational log entry.
func (l *Logger) Info(msg string) {
	l.log(func(ctx context.Context, message string) {
		l.base.Info(ctx, message)
	}, msg)
}

// Debug writes a debug-level log entry if enabled.
func (l *Logger) Debug(msg string) {
	l.log(func(ctx context.Context, message string) {
		l.base.Debug(ctx, message)
	}, msg)
}

// Warn writes a warning level log entry.
func (l *Logger) Warn(msg string) {
	l.log(func(ctx context.Context, message string) {
		l.base.Warn(ctx, message)
	}, msg)
}

// Error writes an error log entry including the supplied error context.
func (l *Logger) Error(err error, msg string) {
	if l == nil || l.base == nil {
		return
	}
	fields := []interface{}{}
	if err != nil {
		fields = append(fields, "error", err)
	}
	l.base.Error(context.Background(), msg, fields...)
}

func (l *Logger) log(fn func(context.Context, string), msg string) {
	if l == nil || l.base == nil || fn == nil {
		return
	}
	fn(context.Background(), strings.TrimSpace(msg))
}

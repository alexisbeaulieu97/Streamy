package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	cblog "github.com/charmbracelet/log"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Options configures the charmbracelet/log adapter.
type Options struct {
	Writer       io.Writer
	Level        string
	TimeFormat   string
	ReportCaller bool
	Formatter    cblog.Formatter
	Layer        string
	Component    string
	Fields       map[string]interface{}
}

const defaultLayer = "infrastructure"

// Logger implements ports.Logger using charmbracelet/log.
type Logger struct {
	logger *cblog.Logger
	fields []interface{}
	layer  string
}

// New creates a Logger adapter with the supplied options.
func New(opts Options) (*Logger, error) {
	writer := opts.Writer
	if writer == nil {
		writer = os.Stdout
	}

	level := cblog.InfoLevel
	if opts.Level != "" {
		parsed, err := cblog.ParseLevel(strings.ToLower(opts.Level))
		if err != nil {
			return nil, fmt.Errorf("parse log level: %w", err)
		}

		level = parsed
	}

	base := cblog.NewWithOptions(writer, cblog.Options{
		Level:           level,
		TimeFormat:      opts.TimeFormat,
		ReportTimestamp: true,
		ReportCaller:    opts.ReportCaller,
		Formatter:       opts.Formatter,
		Fields:          mapToFields(opts.Fields),
	})

	var fields []interface{}
	if opts.Component != "" {
		fields = append(fields, "component", opts.Component)
	}

	layer := opts.Layer
	if layer == "" {
		layer = defaultLayer
	}

	return &Logger{
		logger: base,
		fields: fields,
		layer:  layer,
	}, nil
}

// Debug emits a debug log entry.
func (l *Logger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, cblog.DebugLevel, msg, fields...)
}

// Info emits an info log entry.
func (l *Logger) Info(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, cblog.InfoLevel, msg, fields...)
}

// Warn emits a warning log entry.
func (l *Logger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, cblog.WarnLevel, msg, fields...)
}

// Error emits an error log entry.
func (l *Logger) Error(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, cblog.ErrorLevel, msg, fields...)
}

// With derives a new logger with persistent fields.
func (l *Logger) With(fields ...interface{}) ports.Logger {
	if l == nil {
		return &NoOpLogger{}
	}

	next := make([]interface{}, 0, len(l.fields)+len(fields))
	next = append(next, l.fields...)
	next = append(next, fields...)

	return &Logger{
		logger: l.logger,
		fields: next,
		layer:  l.layer,
	}
}

func (l *Logger) log(ctx context.Context, level cblog.Level, msg string, fields ...interface{}) {
	if l == nil || l.logger == nil {
		return
	}

	extras := map[string]interface{}{
		"layer": l.layer,
	}
	if id := ports.GetCorrelationID(ctx); id != "" {
		extras["correlation_id"] = id
	}

	payload := mergeFields(l.fields, fields, extras)

	switch level {
	case cblog.DebugLevel:
		l.logger.Debug(msg, payload...)
	case cblog.WarnLevel:
		l.logger.Warn(msg, payload...)
	case cblog.ErrorLevel:
		l.logger.Error(msg, payload...)
	default:
		l.logger.Info(msg, payload...)
	}
}

// SetLevel updates the underlying logger level at runtime.
func (l *Logger) SetLevel(level string) error {
	if l == nil || l.logger == nil {
		return nil
	}

	if level == "" {
		return errors.New("log level cannot be empty")
	}

	parsed, err := cblog.ParseLevel(strings.ToLower(level))
	if err != nil {
		return fmt.Errorf("parse log level: %w", err)
	}

	l.logger.SetLevel(parsed)

	return nil
}

func mapToFields(input map[string]interface{}) []interface{} {
	if len(input) == 0 {
		return nil
	}

	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	res := make([]interface{}, 0, len(input)*2)
	for _, k := range keys {
		res = append(res, k, input[k])
	}

	return res
}

func mergeFields(base, additions []interface{}, extras map[string]interface{}) []interface{} {
	store := make(map[string]interface{})
	order := make([]string, 0)

	addPair := func(key string, value interface{}) {
		if key == "" {
			return
		}

		if _, exists := store[key]; !exists {
			order = append(order, key)
		}

		store[key] = value
	}

	process := func(values []interface{}) {
		for i := 0; i+1 < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				continue
			}

			addPair(key, values[i+1])
		}
	}

	process(base)
	process(additions)

	if len(extras) > 0 {
		extraKeys := make([]string, 0, len(extras))
		for key, value := range extras {
			if value == nil {
				continue
			}

			if s, ok := value.(string); ok && s == "" {
				continue
			}

			extraKeys = append(extraKeys, key)
		}

		sort.Strings(extraKeys)

		for _, key := range extraKeys {
			addPair(key, extras[key])
		}
	}

	if rawErr, ok := store["error"]; ok {
		if errVal := toError(rawErr); errVal != nil {
			chain := flattenErrorChain(errVal)
			addPair("error", errVal.Error())

			if len(chain) > 0 {
				addPair("error_chain", chain)
				addPair("error_cause", chain[len(chain)-1])
			}
		}
	}

	result := make([]interface{}, 0, len(order)*2)
	for _, key := range order {
		result = append(result, key, store[key])
	}

	return result
}

func toError(value interface{}) error {
	switch v := value.(type) {
	case error:
		return v
	case fmt.Stringer:
		return errors.New(v.String())
	default:
		return nil
	}
}

func flattenErrorChain(err error) []string {
	if err == nil {
		return nil
	}

	seen := map[string]struct{}{}

	var (
		chain []string
		walk  func(error)
	)

	walk = func(e error) {
		if e == nil {
			return
		}

		msg := e.Error()
		if _, exists := seen[msg]; !exists {
			seen[msg] = struct{}{}
			chain = append(chain, msg)
		}

		if unwrapper, ok := e.(interface{ Unwrap() []error }); ok {
			for _, inner := range unwrapper.Unwrap() {
				walk(inner)
			}

			return
		}

		walk(errors.Unwrap(e))
	}

	walk(err)

	return chain
}

// compile-time assurance
var _ ports.Logger = (*Logger)(nil)

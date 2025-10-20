package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

var domainSuggestions = map[domainpipeline.ErrorCode]string{
	domainpipeline.ErrCodeValidation: "Review the configuration for typos or missing required fields.",
	domainpipeline.ErrCodeDuplicate:  "Ensure each step ID is unique within the pipeline.",
	domainpipeline.ErrCodeDependency: "Verify that referenced dependencies exist and do not form cycles.",
	domainpipeline.ErrCodeCycle:      "Break the circular dependency by adjusting step ordering.",
	domainpipeline.ErrCodeNotFound:   "Confirm the referenced resource or plugin is registered and spelled correctly.",
	domainpipeline.ErrCodeMissing:    "Provide the required field before retrying.",
	domainpipeline.ErrCodeState:      "Check the current system state and rerun after resolving conflicts.",
	domainpipeline.ErrCodeConflict:   "Resolve conflicting operations before re-running the pipeline.",
	domainpipeline.ErrCodeExecution:  "Run with --verbose or inspect plugin logs to diagnose the failure.",
	domainpipeline.ErrCodePlugin:     "Ensure the plugin is implemented correctly and supports the requested operation.",
	domainpipeline.ErrCodeTimeout:    "Increase timeouts or reduce work per step to avoid long-running operations.",
	domainpipeline.ErrCodeCancelled:  "Re-run the command once outstanding cancellations or interrupts are cleared.",
	domainpipeline.ErrCodeConfig:     "Fix configuration syntax errors and try again.",
	domainpipeline.ErrCodeInternal:   "Retry the command and report the issue if it persists.",
}

// FormatError renders rich, multi-line error output suitable for CLI display.
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var builder strings.Builder
	formatError(&builder, err, 0)

	return strings.TrimRight(builder.String(), "\n")
}

func formatError(builder *strings.Builder, err error, depth int) {
	indent := strings.Repeat("  ", depth)

	if agg, ok := err.(*applicationpipeline.AggregateError); ok {
		message := agg.Message
		if message == "" {
			message = "multiple errors occurred"
		}

		builder.WriteString(indent)
		builder.WriteString(message)
		builder.WriteString("\n")

		for i, child := range agg.Errors {
			if child == nil {
				continue
			}

			fmt.Fprintf(builder, "%s  %d.\n", indent, i+1)
			formatError(builder, child, depth+1)
		}

		return
	}

	if formatDomainError(builder, err, depth) {
		return
	}

	if cerr, ok := err.(*commandError); ok {
		fmt.Fprintf(builder, "%sFailed to %s: %s\n", indent, cerr.operation, cerr.context)

		if strings.TrimSpace(cerr.suggestion) != "" {
			builder.WriteString(indent)
			builder.WriteString("  Suggestion: ")
			builder.WriteString(cerr.suggestion)
			builder.WriteString("\n")
		}

		if cerr.cause != nil {
			builder.WriteString(indent)
			builder.WriteString("  Cause:\n")
			formatError(builder, cerr.cause, depth+2)
		}

		return
	}

	builder.WriteString(indent)
	builder.WriteString(err.Error())
	builder.WriteString("\n")
}

func formatDomainError(builder *strings.Builder, err error, depth int) bool {
	var derr *domainpipeline.DomainError
	if !errors.As(err, &derr) {
		return false
	}

	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(builder, "%s[%s] %s", indent, derr.Code, derr.Message)

	if suggestion := domainSuggestions[derr.Code]; suggestion != "" {
		builder.WriteString("\n")
		builder.WriteString(indent)
		builder.WriteString("  Suggestion: ")
		builder.WriteString(suggestion)
	}

	if len(derr.Context) > 0 {
		keys := make([]string, 0, len(derr.Context))
		for key := range derr.Context {
			keys = append(keys, key)
		}

		sort.Strings(keys)
		builder.WriteString("\n")
		builder.WriteString(indent)
		builder.WriteString("  Context:")

		for _, key := range keys {
			builder.WriteString("\n")
			builder.WriteString(indent)
			builder.WriteString("    ")
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(formatContextValue(derr.Context[key]))
		}
	}

	if derr.Cause != nil {
		builder.WriteString("\n")
		builder.WriteString(indent)
		builder.WriteString("  Cause:\n")
		formatError(builder, derr.Cause, depth+2)

		return true
	}

	builder.WriteString("\n")

	return true
}

func formatContextValue(value interface{}) string {
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			return "[]"
		}

		return strings.Join(v, " -> ")
	default:
		return fmt.Sprint(value)
	}
}

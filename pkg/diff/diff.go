package diff

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	maxDiffLines    = 10000
	truncateMessage = "... (diff truncated, exceeds 10,000 lines) ..."
)

// GenerateUnifiedDiff generates a unified diff format output comparing expected and actual content.
// Returns empty string if content is identical.
// Truncates diffs exceeding 10,000 lines with a truncation marker.
func GenerateUnifiedDiff(expected, actual []byte, expectedLabel, actualLabel string) string {
	if bytes.Equal(expected, actual) {
		return ""
	}

	dmp := diffmatchpatch.New()

	// Split into lines for proper diff
	expectedStr := string(expected)
	actualStr := string(actual)

	diffs := dmp.DiffMain(expectedStr, actualStr, false)
	diffs = dmp.DiffCleanupSemantic(diffs)

	// Generate unified diff format
	var buf bytes.Buffer

	// Add headers
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(&buf, "--- %s\t%s\n", expectedLabel, timestamp)
	fmt.Fprintf(&buf, "+++ %s\t%s\n", actualLabel, timestamp)

	// Process diffs line by line
	expectedLines := strings.Split(expectedStr, "\n")
	actualLines := strings.Split(actualStr, "\n")

	// Simple implementation: show all changes
	fmt.Fprintf(&buf, "@@ -1,%d +1,%d @@\n", len(expectedLines), len(actualLines))

	// Build line-by-line diff
	expIdx := 0
	actIdx := 0

	for _, diff := range diffs {
		text := diff.Text
		lines := strings.Split(text, "\n")

		// Remove empty trailing line from split
		if len(lines) > 0 && lines[len(lines)-1] == "" && text[len(text)-1] == '\n' {
			lines = lines[:len(lines)-1]
		}

		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			for _, line := range lines {
				buf.WriteString(" ")
				buf.WriteString(line)
				buf.WriteString("\n")
				expIdx++
				actIdx++
			}
		case diffmatchpatch.DiffDelete:
			for _, line := range lines {
				buf.WriteString("-")
				buf.WriteString(line)
				buf.WriteString("\n")
				expIdx++
			}
		case diffmatchpatch.DiffInsert:
			for _, line := range lines {
				buf.WriteString("+")
				buf.WriteString(line)
				buf.WriteString("\n")
				actIdx++
			}
		}
	}

	// Check line count and truncate if necessary
	result := buf.String()
	lines := strings.Split(result, "\n")
	if len(lines) > maxDiffLines {
		truncated := strings.Join(lines[:maxDiffLines], "\n")
		return truncated + "\n" + truncateMessage + "\n"
	}

	return result
}
